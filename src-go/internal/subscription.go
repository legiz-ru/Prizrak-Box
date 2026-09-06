package internal

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/pkg/proxy"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/metacubex/mihomo/log"
)

const maxSubscriptionMigrations = 3

// swapHost replaces the host portion of rawURL with newHost.
// Scheme, path, query, and fragment are preserved.
func swapHost(rawURL, newHost string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, err
	}
	u.Host = newHost
	return u.String(), nil
}

// isValidSubscriptionURL returns true if s is an absolute http/https URL with a host.
func isValidSubscriptionURL(s string) bool {
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Host != ""
}

// isValidHost returns true if s is a non-empty bare host/domain (no scheme, no path).
func isValidHost(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "://") || strings.Contains(s, "/") {
		return false
	}
	_, err := url.Parse("http://" + s)
	return err == nil
}

// FetchSubscription tries source, then fallbackUrl, then source-with-fallbackDomain in order.
// Returns (result, successURL) for the first 2xx response, or (nil, "") if all fail.
func FetchSubscription(source, fallbackUrl, fallbackDomain string) (*utils.ResponseResult, string) {
	candidates := []string{source}
	if fallbackUrl != "" {
		candidates = append(candidates, fallbackUrl)
	}
	if fallbackDomain != "" {
		if swapped, err := swapHost(source, fallbackDomain); err == nil {
			candidates = append(candidates, swapped)
		}
	}

	proxyURL := proxy.GetProxyUrl()
	for _, candidate := range candidates {
		result, err := utils.FetchSubscriptionCandidate(candidate, proxyURL)
		if err == nil && result != nil {
			return result, candidate
		}
		log.Warnln("[FetchSubscription] candidate %s: %v", candidate, err)
	}
	return nil, ""
}

// UpdateSubscriptionSource runs the full fetch+migration algorithm from the spec.
// It updates profile.Content, profile.FallbackUrl, and profile.FallbackDomain in-place,
// then returns the final fetch result (body + headers from the stable source URL).
//
// onMigration is called immediately after profile.Content changes, to allow the caller
// to persist the new URL before the next fetch attempt. Pass nil when persistence on
// migration is not needed (e.g. initial profile add).
//
// Returns (nil, error) if all candidates are unreachable.
func UpdateSubscriptionSource(profile *models.Profile, onMigration func(*models.Profile)) (*utils.ResponseResult, error) {
	migrations := 0
	for {
		result, _ := FetchSubscription(profile.Content, profile.FallbackUrl, profile.FallbackDomain)
		if result == nil {
			return nil, fmt.Errorf("subscription unreachable: %s", profile.Content)
		}

		headers := result.Headers

		// Update fallback fields from response headers; absent or invalid header = clear (spec §3)
		fallbackUrl := strings.TrimSpace(headers.Get("fallback-url"))
		if !isValidSubscriptionURL(fallbackUrl) {
			fallbackUrl = ""
		}
		profile.FallbackUrl = fallbackUrl

		fallbackDomain := strings.TrimSpace(headers.Get("fallback-domain"))
		if !isValidHost(fallbackDomain) {
			fallbackDomain = ""
		}
		profile.FallbackDomain = fallbackDomain

		// Compute potential new source; new-url takes priority over new-domain (spec §5.2)
		newSource := profile.Content
		if val := strings.TrimSpace(headers.Get("new-url")); isValidSubscriptionURL(val) {
			newSource = val
		} else if val := strings.TrimSpace(headers.Get("new-domain")); isValidHost(val) {
			if swapped, err := swapHost(profile.Content, val); err == nil {
				newSource = swapped
			}
		}

		// Migrate if source changed and budget not exhausted (spec §5.3)
		if newSource != profile.Content && migrations < maxSubscriptionMigrations {
			log.Infoln("[UpdateSubscriptionSource] migration %d/%d: %s → %s",
				migrations+1, maxSubscriptionMigrations, profile.Content, newSource)
			profile.Content = newSource
			migrations++
			if onMigration != nil {
				onMigration(profile)
			}
			continue
		}

		// Source is stable — return result for content processing (spec §5.4)
		return result, nil
	}
}
