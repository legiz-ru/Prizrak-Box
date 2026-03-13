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

// buildUserAgent формирует единый UA вида:
// Clash-Meta/Prizrak-Box (Desktop Build {version} {OS})
// независимо от настройки HWID.
func buildUserAgent(version, deviceOS string) string {
	osName := normalizeOSName(deviceOS)
	if version != "" {
		return fmt.Sprintf("Clash-Meta/Prizrak-Box (Desktop Build %s %s)", version, osName)
	}
	return fmt.Sprintf("Clash-Meta/Prizrak-Box (Desktop Build %s)", osName)
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
	// Снимаем конфиг один раз, чтобы все заголовки запроса
	// были согласованы даже при параллельной смене globalConfig.
	cfg := getConfigSnapshot()

	client, err := newHttpClient(proxyURL, timeout)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, requestURL, nil)
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
	Body    string
	Headers http.Header
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
		case results <- &ResponseResult{Body: html.UnescapeString(string(bodyBytes)), Headers: resp.Header}:
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
