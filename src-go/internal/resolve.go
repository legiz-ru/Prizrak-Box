package internal

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/legiz-ru/prizrak-box/api/models"
	"github.com/legiz-ru/prizrak-box/pkg/constant"
	"github.com/legiz-ru/prizrak-box/pkg/proxy"
	"github.com/legiz-ru/prizrak-box/pkg/utils"
	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/common/convert"
	"github.com/metacubex/mihomo/config"
	"github.com/metacubex/mihomo/log"
	"gopkg.in/yaml.v3"
)

// 保存文件
func saveProfile(proxies []map[string]any, profile *models.Profile) {
	yml := models.Yml{Proxies: proxies}
	out, _ := yaml.Marshal(yml)
	savePath := utils.GetUserHomeDir(profile.Path)
	_, _ = utils.SaveFile(savePath, out)
}

// MapsToProxies 将任意数量的 map[string]any 切片转换为任意数量的 map[string]any 切片，
// 仅包含通过 adapter.ParseProxy 解析成功的元素。
func MapsToProxies(ray []map[string]any) ([]map[string]any, error) {
	pool := utils.NewTimeoutPoolWithDefaults()
	pool.WaitCount(len(ray))
	mutex := sync.Mutex{}

	proxies := make([]map[string]any, 0)
	for _, m := range ray {
		proxy := m
		pool.SubmitWithTimeout(func(done chan struct{}) {
			defer func() {
				if e := recover(); e != nil {
					log.Errorln("[MapsToProxies] Error:%v", e)
				}
				done <- struct{}{}
			}()
			proxy["skip-cert-verify"] = true
			_, err := adapter.ParseProxy(proxy)
			if err == nil {
				mutex.Lock()
				proxies = append(proxies, proxy)
				mutex.Unlock()
			} else {
				marshal, err2 := json.Marshal(proxy)
				if err2 == nil {
					log.Warnln("[MapsToProxies] proxy: %s ,err: %s", string(marshal), err.Error())
				}
			}
		}, 2*time.Second)
	}
	pool.StartAndWait()

	if len(proxies) == 0 {
		return proxies, errors.New("no nodes available, please check the profile")
	}

	return proxies, nil
}

// Resolve 解析内容，保存成 profile
func Resolve(content string, profile *models.Profile, refresh bool) error {
	// 解析内容预处理
	tempStr := strings.TrimSpace(content)

	// age-encrypted content: decrypt before further processing
	if IsAgeEncrypted(tempStr) {
		if profile.AgeSecretKey == "" {
			return errors.New("age-secret-key required to decrypt this profile")
		}
		decrypted, err := DecryptAge(tempStr, profile.AgeSecretKey)
		if err != nil {
			return err
		}
		tempStr = strings.TrimSpace(decrypted)
	}

	tempBytes := []byte(tempStr)

	// 如果不是刷新则创建 id
	if !refresh {
		snowflakeId := utils.SnowflakeId()
		profile.Id = fmt.Sprintf("%s%d", constant.PrefixProfile, snowflakeId)
		profile.Order = strconv.FormatInt(snowflakeId, 10)
		profile.Path = "./profiles/" + profile.Id + ".yaml"
	}

	// Base64解析
	if utils.IsBase64(tempStr) {
		v2ray, err := convert.ConvertsV2Ray(tempBytes)
		if err == nil {
			// 提取正确配置的节点
			v2ray, err = MapsToProxies(v2ray)
			if err != nil {
				return err
			}
			saveProfile(v2ray, profile)
			return nil
		}

		return err
	}

	// 分享链接解析
	shareLinks := ScanShareLinks(tempStr)
	var builder strings.Builder
	for _, link := range shareLinks {
		builder.WriteString(link + "\n")
	}
	if builder.Len() > 0 {
		share, err := convert.ConvertsV2Ray([]byte(builder.String()))
		if err == nil {
			// 提取正确配置的节点
			share, err = MapsToProxies(share)
			if err != nil {
				return err
			}
			saveProfile(share, profile)
			return nil
		}

		return err
	}

	// Sing解析
	// 解析不到节点不退出 有可能是yaml 保存成json了 继续尝试yaml解析
	if utils.IsJSON(tempStr) {
		sing, err := convert.ConvertsSingBox(tempBytes)
		if err == nil {
			// 提取正确配置的节点
			sing, err = MapsToProxies(sing)
			if err != nil {
				return err
			}
			saveProfile(sing, profile)
			return nil
		}
	}

	// Yaml解析
	rawCfg, err := config.UnmarshalRawConfig(tempBytes)
	if err == nil {
		_, yamlError := config.ParseRawConfig(rawCfg)
		if yamlError != nil {
			// 配置校验失败，尝试提取可用节点
			rails, err1 := MapsToProxies(rawCfg.Proxy)
			if err1 != nil {
				return yamlError
			} else {
				saveProfile(rails, profile)
				return nil
			}
		}

		// 保存yaml
		if len(rawCfg.ProxyProvider) > 0 || len(rawCfg.Proxy) > 0 {
			// 防止重排序，重新赋值
			rawCfg, _ = config.UnmarshalRawConfig(tempBytes)
			// 对 provider 进行路径替换
			findProvider := false
			rawConfigMap := map[string]any{}
			if err := yaml.Unmarshal(tempBytes, &rawConfigMap); err == nil {
				findProvider = updateProvidersPathRaw("profiles", profile.Order, rawConfigMap)
			} else {
				findProvider = changeProvidersPath("profiles", profile.Order, rawCfg)
			}
			var yml []byte
			if findProvider {
				if len(rawConfigMap) > 0 {
					yml, _ = yaml.Marshal(rawConfigMap)
				} else {
					yml, _ = yaml.Marshal(rawCfg)
				}
				profile.Path = fmt.Sprintf("./profiles/%s/%s.yaml", profile.Order, profile.Id)
			} else {
				yml = tempBytes
			}

			// 保存操作
			savePath := utils.GetUserHomeDir(profile.Path)
			_, _ = utils.SaveFile(savePath, yml)
			return nil
		} else {
			return fmt.Errorf("proxy or provider is 0")
		}

	}

	return err
}

func changeProvidersPath(baseDir, subDir string, config *config.RawConfig) (findProvider bool) {
	findProvider = false

	dir := fmt.Sprintf("./%s/%s/", baseDir, subDir)
	proxyProviders := config.ProxyProvider
	for _, provider := range proxyProviders {

		if path, findPath := provider["path"]; findPath {
			provider["path"] = dir + getProviderBase("provider", path.(string))
		} else {
			if u, findUrl := provider["url"]; findUrl {
				provider["path"] = dir + "provider/" + utils.MD5(u.(string))
			}
		}

		findProvider = true
	}

	ruleProviders := config.RuleProvider
	for _, ruleProvider := range ruleProviders {

		if path, findPath := ruleProvider["path"]; findPath {
			ruleProvider["path"] = dir + getProviderBase("ruleset", path.(string))
		} else {
			if u, findUrl := ruleProvider["url"]; findUrl {
				ruleProvider["path"] = dir + "ruleset/" + utils.MD5(u.(string))
			}
		}

		findProvider = true
	}

	return
}

func updateProvidersPathRaw(baseDir, subDir string, raw map[string]any) (findProvider bool) {
	findProvider = false
	dir := fmt.Sprintf("./%s/%s/", baseDir, subDir)

	if providers, ok := raw["proxy-providers"]; ok {
		if updateProviderPaths(dir, "provider", providers) {
			findProvider = true
		}
	}
	if providers, ok := raw["proxyProviders"]; ok {
		if updateProviderPaths(dir, "provider", providers) {
			findProvider = true
		}
	}
	if providers, ok := raw["rule-providers"]; ok {
		if updateProviderPaths(dir, "ruleset", providers) {
			findProvider = true
		}
	}
	if providers, ok := raw["ruleProviders"]; ok {
		if updateProviderPaths(dir, "ruleset", providers) {
			findProvider = true
		}
	}

	return
}

func updateProviderPaths(dir, kind string, providers any) bool {
	providerMap, ok := providers.(map[string]any)
	if !ok {
		return false
	}
	updated := false
	for key, value := range providerMap {
		provider, ok := value.(map[string]any)
		if !ok {
			continue
		}

		if path, findPath := provider["path"].(string); findPath {
			provider["path"] = dir + getProviderBase(kind, path)
			updated = true
		} else if u, findUrl := provider["url"].(string); findUrl {
			provider["path"] = dir + kind + "/" + utils.MD5(u)
			updated = true
		}

		providerMap[key] = provider
	}

	return updated
}

func getProviderBase(provider, path string) string {
	return provider + "/" + filepath.Base(path)
}

func parseFields(input string) map[string]*big.Int {
	// 分割字段
	pairs := strings.Split(input, ";")
	result := make(map[string]*big.Int)

	// 处理每个键值对
	for _, pair := range pairs {
		// 去掉可能的空格
		pair = strings.TrimSpace(pair)
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			bigInt := new(big.Int)
			bigInt, ok := bigInt.SetString(value, 10)
			if ok {
				result[key] = bigInt
			}
		}
	}

	// 最后补齐缺失字段
	for _, key := range []string{"total", "upload", "download", "expire"} {
		if val, ok := result[key]; !ok || val == nil {
			result[key] = big.NewInt(0)
		}
	}

	return result
}

func parseContentDisposition(header http.Header, urlStr string) string {
	disposition := header.Get("Content-Disposition")
	if disposition != "" {
		disposition, _ = url.QueryUnescape(disposition)
		split := strings.Split(disposition, "=")
		fileName := split[len(split)-1]
		if strings.Contains(fileName, "''") {
			fileName = strings.Split(fileName, "''")[1]
		}
		if strings.TrimSpace(fileName) != "" {
			return strings.TrimSpace(fileName)
		}
	}

	// Fallback: extract the last part of the URL
	if parsedURL, err := url.Parse(urlStr); err == nil {
		segments := strings.Split(parsedURL.Path, "/")
		return segments[len(segments)-1]
	}

	return "Remote-" + utils.GetDateTime()
}

func parseProfileTitle(header http.Header) string {
	raw := header.Get("Profile-Title")
	if raw == "" {
		return ""
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "base64:") {
		encoded := strings.TrimSpace(trimmed[len("base64:"):])
		if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
			title := strings.TrimSpace(string(decoded))
			if title != "" {
				return title
			}
		}
	}

	return trimmed
}

// ParseInlineHeaders scans the subscription content for metadata style lines such as
// "#profile-title: example" and converts them into an http.Header instance.
// These inline headers act as a fallback when the upstream server cannot set
// the corresponding HTTP headers directly.
func ParseInlineHeaders(content string) http.Header {
	headers := http.Header{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimLeft(line[1:], " \t")
		if line == "" {
			continue
		}

		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}

		key := http.CanonicalHeaderKey(strings.TrimSpace(line[:idx]))
		value := strings.TrimSpace(line[idx+1:])
		if key == "" || value == "" {
			continue
		}

		headers.Add(key, value)
	}

	return headers
}

// MergeHeaders combines a primary HTTP header map with fallback values. The primary
// header values (typically obtained from the HTTP response) are preserved, while
// fallback values (usually parsed from inline subscription metadata) are only
// applied when a corresponding key is absent in the primary headers.
func MergeHeaders(primary http.Header, fallback http.Header) http.Header {
	merged := http.Header{}

	if primary != nil {
		for key, values := range primary {
			if len(values) == 0 {
				continue
			}

			copied := make([]string, len(values))
			copy(copied, values)
			merged[key] = copied
		}
	}

	if fallback != nil {
		for key, values := range fallback {
			if len(values) == 0 {
				continue
			}

			if existing, ok := merged[key]; ok && len(existing) > 0 {
				continue
			}

			copied := make([]string, len(values))
			copy(copied, values)
			merged[key] = copied
		}
	}

	return merged
}

func parseProfileLogo(header http.Header, profile *models.Profile) {
	logoURL := strings.TrimSpace(header.Get("Profile-Logo"))
	if logoURL == "" {
		profile.Logo = ""
		return
	}

	data, contentType, err := downloadProfileLogo(logoURL)
	if err != nil {
		log.Warnln("[ParseHeaders] failed to download profile logo:", err)
		return
	}

	dataURL := buildLogoDataURL(contentType, data)
	if dataURL != "" {
		profile.Logo = dataURL
	}
}

func downloadProfileLogo(logoURL string) ([]byte, string, error) {
	parsed, err := url.Parse(logoURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid logo url: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, "", fmt.Errorf("unsupported logo url scheme: %s", parsed.Scheme)
	}

	// Try a direct connection first: at profile add-time the proxy core may not
	// be up yet (the old proxy-only fetch failed with "connection refused" and
	// the logo only appeared after a manual refresh). Fall back to the proxy
	// for CDNs that are only reachable through it.
	data, headers, err := utils.SendGetBytes(logoURL, map[string]string{}, "")
	if err != nil || len(data) == 0 {
		if proxyURL := proxy.GetProxyUrl(); proxyURL != "" {
			if d, h, e := utils.SendGetBytes(logoURL, map[string]string{}, proxyURL); e == nil && len(d) > 0 {
				data, headers, err = d, h, nil
			}
		}
	}
	if err != nil {
		return nil, "", err
	}

	if len(data) == 0 {
		return nil, "", errors.New("logo response is empty")
	}

	if len(data) > 2*1024*1024 {
		return nil, "", errors.New("logo file is too large")
	}

	contentType := normalizeLogoContentType(headers.Get("Content-Type"), data)
	if contentType == "" {
		return nil, "", errors.New("logo content type is not supported")
	}

	return data, contentType, nil
}

func normalizeLogoContentType(contentType string, data []byte) string {
	trimmed := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if trimmed == "" && len(data) > 0 {
		trimmed = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0]))
	}

	if strings.HasPrefix(trimmed, "image/") {
		return trimmed
	}

	return ""
}

func buildLogoDataURL(contentType string, data []byte) string {
	if contentType == "" || len(data) == 0 {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
}

// limitAnnounceText limits the announce text to maxVisible displayed characters
// excluding color codes in format #RRGGBB
func limitAnnounceText(text string, maxVisible int) string {
	if text == "" {
		return text
	}

	runes := []rune(text)
	visibleCount := 0
	i := 0

	for i < len(runes) {
		// Check if this is a color code #RRGGBB
		if runes[i] == '#' && i+6 < len(runes) {
			isColorCode := true
			for j := 1; j <= 6; j++ {
				r := runes[i+j]
				if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
					isColorCode = false
					break
				}
			}
			if isColorCode {
				// Skip the color code (#RRGGBB = 7 characters)
				i += 7
				continue
			}
		}

		// This is a visible character
		visibleCount++
		if visibleCount > maxVisible {
			// Return text up to this position
			return string(runes[:i])
		}
		i++
	}

	return text
}

// ParseHeaders 对请求头进行解析
func ParseHeaders(header http.Header, url string, profile *models.Profile) {
	// 流量
	if value := header.Get("Subscription-Userinfo"); value != "" {
		subInfo := parseFields(value)
		zero := big.NewInt(0)

		total := subInfo["total"]
		if total == nil {
			total = zero
		}

		upload := subInfo["upload"]
		if upload == nil {
			upload = zero
		}

		download := subInfo["download"]
		if download == nil {
			download = zero
		}

		if total.Cmp(zero) <= 0 {
			profile.Total = nil
			profile.Used = nil
			profile.Available = nil
		} else {
			profile.Total = new(big.Int).Set(total)
			used := new(big.Int).Add(upload, download)
			profile.Used = used

			available := new(big.Int).Sub(total, used)
			if available.Cmp(zero) <= 0 {
				available = new(big.Int).Set(zero)
			}
			profile.Available = available
		}

		expire := subInfo["expire"]
		if expire != nil && expire.Cmp(zero) > 0 {
			// 转换为时间
			t := time.Unix(expire.Int64(), 0)
			profile.Expire = t.Local().Format("2006-01-02 15:04")
		}
	}

	// 文件名
	nameFromDisposition := parseContentDisposition(header, url)
	if profileTitle := parseProfileTitle(header); profileTitle != "" {
		profile.HeaderTitle = profileTitle

		baseName := strings.TrimSpace(nameFromDisposition)
		if baseName == "" {
			baseName = strings.TrimSpace(profile.Title)
		}

		if baseName != "" && !strings.EqualFold(profileTitle, baseName) {
			profile.Title = fmt.Sprintf("%s (%s)", profileTitle, baseName)
		} else {
			profile.Title = profileTitle
		}
	} else {
		profile.HeaderTitle = ""
		if profile.Title == "" {
			profile.Title = nameFromDisposition
		}
	}

	// 更新间隔
	if val := header.Get("Profile-Update-Interval"); val != "" {
		profile.Interval = val
	}

	// 主页
	if val := header.Get("Profile-Web-Page-Url"); val != "" {
		profile.Home = val
	}

	if val := header.Get("Support-Url"); val != "" {
		profile.Support = val
	}

	// Announce
	if val := header.Get("Announce"); val != "" {
		announce := val
		// Check if base64 encoded
		if strings.HasPrefix(val, "base64:") {
			decoded, err := base64.StdEncoding.DecodeString(val[7:])
			if err == nil {
				announce = string(decoded)
			}
		}
		// Limit to 200 visible characters (excluding color codes)
		announce = limitAnnounceText(announce, 200)
		profile.Announce = announce
	}

	if val := header.Get("Announce-Url"); val != "" {
		profile.AnnounceUrl = val
	}

	parseProfileLogo(header, profile)

	// pxd-template
	if val := strings.TrimSpace(header.Get("Pxd-Template")); val != "" {
		profile.PxdTemplateUrl = val
	} else {
		profile.PxdTemplateUrl = ""
	}

	// pxd-template-scheme
	if val := strings.TrimSpace(header.Get("Pxd-Template-Scheme")); val != "" {
		profile.PxdTemplateScheme = val
	} else {
		profile.PxdTemplateScheme = ""
	}

	// HWID active flag — сохраняется в профиле чтобы edit-диалог мог показать индикатор
	profile.HwidActive = strings.EqualFold(strings.TrimSpace(header.Get("X-Hwid-Active")), "true")

	// global-mode: значение строго "false" (без учёта регистра) скрывает режим
	// Global в переключателе режимов слева. Любое другое значение или отсутствие
	// заголовка оставляет режим доступным. Применяется только к десктопным версиям.
	profile.GlobalModeDisabled = strings.EqualFold(strings.TrimSpace(header.Get("global-mode")), "false")
}
