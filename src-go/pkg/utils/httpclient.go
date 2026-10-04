package utils

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

// 全局超时设置
var (
	ConnTimeOut = 14 * time.Second
	DialTimeOut = 5 * time.Second
	FastTimeOut = 15 * time.Second
	headPattern = regexp.MustCompile(`204|blank|generate|gstatic`)
)

// HTTPClientConfig 配置结构
type HTTPClientConfig struct {
	EnableHWID  bool
	Version     string
	DeviceOS    string
	DeviceOSVer string
	DeviceModel string
	UserAgent   string
}

// 全局配置，可通过API更新
var (
	globalConfigMu sync.RWMutex
	globalConfig   = &HTTPClientConfig{}
)

// coreVersion — версия Prizrak-Core, вшитая в бинарник (из replace модуля mihomo).
// Пустая строка, если определить не удалось. Переменная, чтобы тесты могли подменить.
var coreVersion = detectCoreVersion()

func detectCoreVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, dep := range info.Deps {
		if dep.Path != "github.com/metacubex/mihomo" {
			continue
		}
		if dep.Replace != nil && dep.Replace.Version != "" {
			return dep.Replace.Version
		}
		if dep.Version != "" && dep.Version != "(devel)" {
			return dep.Version
		}
	}
	return ""
}

// buildUserAgent формирует единый UA вида:
// prizrak-box/{version} (Desktop Build; {OS} OS; Prizrak-Core {coreVersion})
// независимо от настройки HWID. Версия приложения стоит сразу после
// "prizrak-box/": по этому префиксу панель Remnawave узнаёт расширенного клиента
// (и отдаёт serverDescription), поэтому слэш и версия есть всегда — "unknown",
// если версия не задана. Суффикс ядра опускается, если его версия неизвестна.
func buildUserAgent(version, deviceOS string) string {
	if version == "" {
		version = "unknown"
	}
	parts := []string{"Desktop Build", normalizeOSName(deviceOS) + " OS"}
	if coreVersion != "" {
		parts = append(parts, "Prizrak-Core "+coreVersion)
	}
	return fmt.Sprintf("prizrak-box/%s (%s)", version, strings.Join(parts, "; "))
}

func hashString(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func generateRawHWID() string {
	if id, err := machineid.ProtectedID("prizrak-box"); err == nil && id != "" {
		return id
	}

	if id, err := machineid.ID(); err == nil && id != "" {
		return hashString(id)
	}

	if host, err := os.Hostname(); err == nil && host != "" {
		return hashString(host)
	}

	return uuid.New().String()
}

// UpdateHTTPClientConfig 更新HTTP客户端配置
func UpdateHTTPClientConfig(config *HTTPClientConfig) {
	globalConfigMu.Lock()
	defer globalConfigMu.Unlock()

	globalConfig = config
	// UA всегда строится в едином формате, независимо от EnableHWID.
	if globalConfig.UserAgent == "" {
		globalConfig.UserAgent = buildUserAgent(globalConfig.Version, globalConfig.DeviceOS)
	}
}

// getConfigSnapshot возвращает атомарную копию текущего конфига,
// чтобы все заголовки одного запроса использовали согласованные данные.
func getConfigSnapshot() HTTPClientConfig {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return *globalConfig
}

// buildDeviceHeaders 构建设备信息头部
func buildDeviceHeaders() map[string]string {
	cfg := getConfigSnapshot()
	headers, _ := resolveDeviceHeadersFromConfig(cfg, cfg.EnableHWID)
	return headers
}

func GetResolvedDeviceDetails() DeviceDetails {
	_, details := resolveDeviceHeaders(true)
	return details
}

// GetUserAgent returns the current User-Agent string.
func GetUserAgent() string {
	cfg := getConfigSnapshot()
	if cfg.UserAgent != "" {
		return cfg.UserAgent
	}
	return buildUserAgent("", "")
}

// IsHWIDEnabled returns whether HWID headers are currently enabled.
func IsHWIDEnabled() bool {
	return getConfigSnapshot().EnableHWID
}

func resolveDeviceHeaders(enable bool) (map[string]string, DeviceDetails) {
	cfg := getConfigSnapshot()
	return resolveDeviceHeadersFromConfig(cfg, enable)
}

// resolveDeviceHeadersFromConfig строит заголовки устройства на основе
// снимка конфига, чтобы избежать race condition при параллельных запросах.
func resolveDeviceHeadersFromConfig(cfg HTTPClientConfig, enable bool) (map[string]string, DeviceDetails) {
	details := GetDeviceDetails()
	resolved := DeviceDetails{
		HWID:      details.HWID,
		OS:        firstNonEmpty(cfg.DeviceOS, details.OS),
		OSVersion: firstNonEmpty(cfg.DeviceOSVer, details.OSVersion),
		Model:     firstNonEmpty(cfg.DeviceModel, details.Model),
	}

	resolved.OS = normalizeOSName(resolved.OS)

	if !enable {
		return nil, resolved
	}

	headers := make(map[string]string)

	if resolved.HWID != "" {
		headers["x-hwid"] = resolved.HWID
	}
	if resolved.OS != "" {
		headers["x-device-os"] = resolved.OS
	}
	if resolved.OSVersion != "" {
		headers["x-ver-os"] = resolved.OSVersion
	}
	if resolved.Model != "" {
		headers["x-device-model"] = resolved.Model
	}

	return headers, resolved
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// closeResponseBody 关闭 resp.Body 并处理错误（建议放在 defer 中）
func closeResponseBody(body io.Closer) {
	if body == nil {
		return
	}
	if err := body.Close(); err != nil {

	}
}

// newHttpClient 创建带代理和超时的 http.Client
func newHttpClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	var proxyFunc func(*http.Request) (*url.URL, error)
	if proxyURL != "" {
		parsedProxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("解析代理路径失败: %w", err)
		}
		proxyFunc = http.ProxyURL(parsedProxy)
	}

	transport := &http.Transport{
		Proxy: proxyFunc,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DialContext: (&net.Dialer{
			Timeout: DialTimeOut,
		}).DialContext,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// sendRequest 发送 HTTP 请求，返回响应对象，由调用方负责关闭 Body
func sendRequest(method, requestURL string, headers map[string]string, proxyURL string, timeout time.Duration) (*http.Response, error) {
	return sendRequestCtx(context.Background(), method, requestURL, headers, proxyURL, timeout)
}

// sendRequestCtx is sendRequest whose request can be cancelled through ctx —
// a cancelled request is dropped before (or while) it reaches the server.
func sendRequestCtx(ctx context.Context, method, requestURL string, headers map[string]string, proxyURL string, timeout time.Duration) (*http.Response, error) {
	// Снимаем конфиг один раз, чтобы все заголовки запроса
	// были согласованы даже при параллельной смене globalConfig.
	cfg := getConfigSnapshot()

	client, err := newHttpClient(proxyURL, timeout)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// Добавляем HWID-заголовки если опция включена.
	// Используем тот же снимок конфига для консистентности.
	if cfg.EnableHWID {
		deviceHeaders, _ := resolveDeviceHeadersFromConfig(cfg, true)
		for k, v := range deviceHeaders {
			req.Header.Set(k, v)
		}
	}

	// 添加用户自定义头部
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Устанавливаем User-Agent (приоритет: пользовательские заголовки > конфиг).
	if _, ok := headers["User-Agent"]; !ok {
		userAgent := cfg.UserAgent
		if userAgent == "" {
			userAgent = buildUserAgent("", "")
		}
		req.Header.Set("User-Agent", userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	return resp, nil
}

// SendGet 发送 GET 请求，返回响应内容和头部
func SendGet(requestURL string, headers map[string]string, proxyURL string) (string, http.Header, error) {
	resp, err := sendRequest("GET", requestURL, headers, proxyURL, ConnTimeOut)
	if err != nil {
		return "", nil, err
	}
	defer closeResponseBody(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("读取响应内容失败: %w", err)
	}

	return html.UnescapeString(string(bodyBytes)), resp.Header, nil
}

// SendGetBytes 发送 GET 请求并返回原始字节和头部
func SendGetBytes(requestURL string, headers map[string]string, proxyURL string) ([]byte, http.Header, error) {
	resp, err := sendRequest("GET", requestURL, headers, proxyURL, ConnTimeOut)
	if err != nil {
		return nil, nil, err
	}
	defer closeResponseBody(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取响应内容失败: %w", err)
	}

	return bodyBytes, resp.Header, nil
}

type ResponseResult struct {
	Body       string
	Headers    http.Header
	StatusCode int
}

// FastGet 并发 GET 请求，代理和直连同时发，谁先成功返回
func FastGet(requestURL string, headers map[string]string, proxyURL string) (*ResponseResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), FastTimeOut)
	defer cancel()

	results := make(chan *ResponseResult, 2)
	errors := make(chan error, 2)

	send := func(useProxy bool) {
		var proxy string
		if useProxy {
			proxy = proxyURL
		}

		resp, err := sendRequest("GET", requestURL, headers, proxy, ConnTimeOut)
		if err != nil {
			errors <- err
			return
		}
		defer closeResponseBody(resp.Body)

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil || len(bodyBytes) == 0 {
			if err == nil {
				err = fmt.Errorf("响应内容为空")
			}
			errors <- err
			return
		}

		select {
		case results <- &ResponseResult{Body: html.UnescapeString(string(bodyBytes)), Headers: resp.Header, StatusCode: resp.StatusCode}:
		case <-ctx.Done():
		}
	}

	go send(true)
	go send(false)

	var errList []string
	for i := 0; i < 2; i++ {
		select {
		case result := <-results:
			return result, nil
		case err := <-errors:
			errList = append(errList, err.Error())
			// 如果两个都失败，立即返回
			if len(errList) == 2 {
				return nil, fmt.Errorf("请求失败[1]: %s", strings.Join(errList, " | "))
			}
		case <-ctx.Done():
			if len(errList) == 0 {
				return nil, fmt.Errorf("请求超时，未收到任何响应")
			}
			return nil, fmt.Errorf("请求失败[2]: %s", strings.Join(errList, " | "))
		}
	}

	// 理论上不会到这里，但作为兜底处理
	if len(errList) > 0 {
		return nil, fmt.Errorf("请求失败[3]: %s", strings.Join(errList, " | "))
	}

	return nil, fmt.Errorf("请求失败，未知原因")
}

// SubscriptionTimeout is the per-candidate timeout for fallback subscription fetching.
const SubscriptionTimeout = 9 * time.Second

// Subscription routes: how a request reaches the panel.
const (
	SubscriptionRouteDirect = "direct"
	SubscriptionRouteProxy  = "proxy"
)

// subscriptionRouteDelay is how long the preferred route has to answer before
// the other one is started as well. A variable so tests can shorten it.
var subscriptionRouteDelay = 1500 * time.Millisecond

type routeOutcome struct {
	result *ResponseResult
	route  string
	err    error
}

// fetchSubscriptionRoute fetches rawURL over one route, treating anything but
// 2xx and an empty body as a failure.
func fetchSubscriptionRoute(ctx context.Context, rawURL, route, proxyURL string) routeOutcome {
	pURL := ""
	if route == SubscriptionRouteProxy {
		pURL = proxyURL
	}

	resp, err := sendRequestCtx(ctx, "GET", rawURL, map[string]string{}, pURL, SubscriptionTimeout)
	if err != nil {
		return routeOutcome{route: route, err: err}
	}
	defer closeResponseBody(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return routeOutcome{route: route, err: fmt.Errorf("HTTP %d", resp.StatusCode)}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(bodyBytes) == 0 {
		if err == nil {
			err = fmt.Errorf("empty response body")
		}
		return routeOutcome{route: route, err: err}
	}

	return routeOutcome{
		route: route,
		result: &ResponseResult{
			Body:       html.UnescapeString(string(bodyBytes)),
			Headers:    resp.Header,
			StatusCode: resp.StatusCode,
		},
	}
}

// FetchSubscriptionCandidate fetches rawURL, treating HTTP 300-599 as failure,
// and sends ONE request to the panel in the normal case.
//
// There are two routes: direct, and through the local proxy (proxyURL, empty =
// none). The preferred one — the route that worked last time, "" = direct —
// goes first; the other starts only if the first fails or has not answered
// within subscriptionRouteDelay, and the first success cancels whatever is
// still in flight. (It used to start both at once and let the loser finish,
// so the panel saw every update twice — once from the user's own address, once
// from the VPN's exit.)
//
// Returns the response and the route that produced it. The route is "" when
// no proxy was configured: there was nothing to choose between, so there is
// nothing worth remembering.
func FetchSubscriptionCandidate(rawURL, proxyURL, preferred string) (*ResponseResult, string, error) {
	routes := []string{SubscriptionRouteDirect}
	if proxyURL != "" {
		if preferred == SubscriptionRouteProxy {
			routes = []string{SubscriptionRouteProxy, SubscriptionRouteDirect}
		} else {
			routes = []string{SubscriptionRouteDirect, SubscriptionRouteProxy}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), SubscriptionTimeout)
	defer cancel()

	outcomes := make(chan routeOutcome, len(routes))
	start := func(route string) {
		go func() { outcomes <- fetchSubscriptionRoute(ctx, rawURL, route, proxyURL) }()
	}

	started := 1
	start(routes[0])

	var delay <-chan time.Time
	if len(routes) > 1 {
		timer := time.NewTimer(subscriptionRouteDelay)
		defer timer.Stop()
		delay = timer.C
	}

	var errList []string
	finished := 0
	for finished < started || started < len(routes) {
		select {
		case o := <-outcomes:
			finished++
			if o.err == nil {
				cancel()
				if proxyURL == "" {
					return o.result, "", nil
				}
				return o.result, o.route, nil
			}
			errList = append(errList, fmt.Sprintf("%s: %v", o.route, o.err))
			// The first route failed outright: no point waiting out the delay.
			if started < len(routes) {
				start(routes[started])
				started++
				delay = nil
			}
		case <-delay:
			start(routes[started])
			started++
			delay = nil
		case <-ctx.Done():
			if len(errList) == 0 {
				return nil, "", fmt.Errorf("timed out after %s", SubscriptionTimeout)
			}
			return nil, "", fmt.Errorf("timed out: %s", strings.Join(errList, " | "))
		}
	}

	return nil, "", fmt.Errorf("%s", strings.Join(errList, " | "))
}

// SendHead 根据 URL 内容判断用 HEAD 还是 GET 请求，返回状态码
func SendHead(requestURL string, proxyURL string) (int, error) {
	method := "GET"
	if headPattern.MatchString(requestURL) {
		method = "HEAD"
	}

	resp, err := sendRequest(method, requestURL, map[string]string{}, proxyURL, 8*time.Second)
	if err != nil {
		return 500, err
	}
	defer closeResponseBody(resp.Body)

	return resp.StatusCode, nil
}
