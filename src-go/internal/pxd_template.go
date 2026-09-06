package internal

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/pkg/cache"
	"github.com/legiz-ru/prizrak-box/pkg/proxy"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/metacubex/mihomo/config"
	"github.com/metacubex/mihomo/log"
	"gopkg.in/yaml.v3"
)

// templateCacheEntry stores downloaded template content with an expiry time.
type templateCacheEntry struct {
	Content   string    `json:"content"`
	ExpiresAt time.Time `json:"expiresAt"`
}

const templateCacheTTL = time.Hour

// templateCacheMem is a process-lifetime in-memory cache (avoids BBolt overhead on repeated switches).
var (
	templateCacheMem   = map[string]templateCacheEntry{}
	templateCacheMu    sync.RWMutex
)

// buildProxyProviderConfig handles pxd-template-scheme=proxy-providers:
// downloads the template, substitutes placeholders, and returns the RawConfig.
func buildProxyProviderConfig(profile models.Profile) (*config.RawConfig, error) {
	templateContent, err := downloadPxdTemplate(profile.PxdTemplateUrl)
	if err != nil {
		return nil, fmt.Errorf("pxd proxy-providers: download template: %w", err)
	}

	result := substitutePlaceholders(templateContent, profile)
	result = strings.ReplaceAll(result, "$payload$", "")

	rawCfg, err := config.UnmarshalRawConfig([]byte(result))
	if err != nil {
		return nil, fmt.Errorf("pxd proxy-providers: parse template: %w", err)
	}

	return rawCfg, nil
}

// buildPayloadConfig handles pxd-template-scheme=payload:
// extracts the proxy list from the profile, downloads the template,
// substitutes $payload$ and other placeholders, and returns the RawConfig.
func buildPayloadConfig(profile models.Profile) (*config.RawConfig, error) {
	// Extract proxy list from stored profile file
	profileRaw, err := loadProfileRawConfig(profile)
	if err != nil {
		return nil, fmt.Errorf("pxd payload: load profile: %w", err)
	}

	proxyYAML, err := marshalProxyList(profileRaw.Proxy)
	if err != nil {
		return nil, fmt.Errorf("pxd payload: marshal proxies: %w", err)
	}

	templateContent, err := downloadPxdTemplate(profile.PxdTemplateUrl)
	if err != nil {
		return nil, fmt.Errorf("pxd payload: download template: %w", err)
	}

	// First substitute $payload$ with proper indentation, then other placeholders
	result := substitutePayloadPlaceholder(templateContent, proxyYAML)
	result = substitutePlaceholders(result, profile)

	rawCfg, err := config.UnmarshalRawConfig([]byte(result))
	if err != nil {
		return nil, fmt.Errorf("pxd payload: parse template: %w", err)
	}

	return rawCfg, nil
}

// downloadPxdTemplate fetches the template YAML from the given URL.
// Results are cached in memory and in BBolt DB with a 1-hour TTL so that
// repeated profile switches and app restarts do not re-download the template.
func downloadPxdTemplate(templateURL string) (string, error) {
	cacheKey := "pxd_tmpl_" + utils.MD5(templateURL)

	// 1. In-memory cache (hot path, no I/O)
	templateCacheMu.RLock()
	if entry, ok := templateCacheMem[cacheKey]; ok && time.Now().Before(entry.ExpiresAt) {
		templateCacheMu.RUnlock()
		log.Debugln("[pxd-template] in-memory cache hit for %s", templateURL)
		return entry.Content, nil
	}
	templateCacheMu.RUnlock()

	// 2. Persistent BBolt cache (survives restarts)
	var cached templateCacheEntry
	if err := cache.Get(cacheKey, &cached); err == nil && time.Now().Before(cached.ExpiresAt) {
		log.Debugln("[pxd-template] BBolt cache hit for %s", templateURL)
		templateCacheMu.Lock()
		templateCacheMem[cacheKey] = cached
		templateCacheMu.Unlock()
		return cached.Content, nil
	}

	// 3. Download — FastGet tries proxy and direct connection simultaneously,
	//    returning whichever responds first (faster than SendGet on first load
	//    when no proxy is running yet).
	log.Infoln("[pxd-template] downloading template from %s", templateURL)
	res, err := utils.FastGet(templateURL, map[string]string{}, proxy.GetProxyUrl())
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Body) == "" {
		return "", fmt.Errorf("empty template response from %s", templateURL)
	}

	// 4. Store in both caches
	entry := templateCacheEntry{
		Content:   res.Body,
		ExpiresAt: time.Now().Add(templateCacheTTL),
	}
	templateCacheMu.Lock()
	templateCacheMem[cacheKey] = entry
	templateCacheMu.Unlock()
	_ = cache.Put(cacheKey, entry)

	return res.Body, nil
}

// marshalProxyList converts a slice of proxy maps into a YAML list string
// (the lines that follow the "proxies:" key, without the key itself).
func marshalProxyList(proxies []map[string]any) (string, error) {
	if len(proxies) == 0 {
		return "", nil
	}
	out, err := yaml.Marshal(proxies)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// substitutePlaceholders replaces all supported template placeholders.
func substitutePlaceholders(template string, profile models.Profile) string {
	result := template

	// $subscription_url$
	result = strings.ReplaceAll(result, "$subscription_url$", profile.Content)

	// $User-Agent$
	result = strings.ReplaceAll(result, "$User-Agent$", utils.GetUserAgent())

	// $profile-update-interval$ — convert hours to seconds
	interval := parseIntervalHoursToSeconds(profile.Interval)
	result = strings.ReplaceAll(result, "$profile-update-interval$", strconv.Itoa(interval))

	// HWID placeholders
	if utils.IsHWIDEnabled() {
		details := utils.GetResolvedDeviceDetails()
		result = strings.ReplaceAll(result, "$x-hwid$", details.HWID)
		result = strings.ReplaceAll(result, "$x-device-os$", details.OS)
		result = strings.ReplaceAll(result, "$x-ver-os$", details.OSVersion)
		result = strings.ReplaceAll(result, "$x-device-model$", details.Model)
	} else {
		result = strings.ReplaceAll(result, "$x-hwid$", "")
		result = strings.ReplaceAll(result, "$x-device-os$", "")
		result = strings.ReplaceAll(result, "$x-ver-os$", "")
		result = strings.ReplaceAll(result, "$x-device-model$", "")
	}

	return result
}

// substitutePayloadPlaceholder replaces lines containing "$payload$" with the
// proxy YAML list, using the correct indentation derived from context.
// If $payload$ itself has no indentation, the previous non-empty line's
// indentation + 2 spaces is used so the result is valid YAML under its parent key.
func substitutePayloadPlaceholder(template string, proxyYAML string) string {
	if !strings.Contains(template, "$payload$") {
		return template
	}

	lines := strings.Split(template, "\n")
	var result []string

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.TrimSpace(trimmed) == "$payload$" {
			// Indentation from the placeholder line itself
			indent := line[:len(line)-len(trimmed)]

			// If the placeholder has no indentation, derive it from the parent key
			if indent == "" {
				for j := i - 1; j >= 0; j-- {
					prev := lines[j]
					if strings.TrimSpace(prev) != "" {
						prevTrimmed := strings.TrimLeft(prev, " \t")
						indent = prev[:len(prev)-len(prevTrimmed)] + "  "
						break
					}
				}
			}

			// Apply indent to every line of the proxy YAML
			payloadLines := strings.Split(proxyYAML, "\n")
			for _, pl := range payloadLines {
				if pl != "" {
					result = append(result, indent+pl)
				}
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// parseIntervalHoursToSeconds parses the profile interval (in hours as a string)
// and returns the equivalent in seconds. Returns 0 if parsing fails.
func parseIntervalHoursToSeconds(interval string) int {
	if interval == "" {
		return 0
	}
	hours, err := strconv.Atoi(strings.TrimSpace(interval))
	if err != nil {
		return 0
	}
	return hours * 3600
}

// getPxdTemplateBytes downloads and returns the pxd-template bytes for use
// as a regular conversion template (pxd-template without a special scheme).
func getPxdTemplateBytes(profile models.Profile) ([]byte, bool) {
	if profile.PxdTemplateUrl == "" || profile.PxdTemplateScheme != "" {
		return nil, false
	}

	body, err := downloadPxdTemplate(profile.PxdTemplateUrl)
	if err != nil {
		log.Warnln("[pxd-template] failed to download template from %s: %v", profile.PxdTemplateUrl, err)
		return nil, false
	}

	return []byte(body), true
}
