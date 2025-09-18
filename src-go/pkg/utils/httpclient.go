package utils

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

// 全局超时设置
var (
	ConnTimeOut      = 14 * time.Second
	DialTimeOut      = 5 * time.Second
	FastTimeOut      = 15 * time.Second
	defaultUserAgent = "clash-verge/v2.3.0"
	headPattern      = regexp.MustCompile(`204|blank|generate|gstatic`)
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
var globalConfig = &HTTPClientConfig{
	EnableHWID: false,
	UserAgent:  defaultUserAgent,
}

// generateHWID 生成唯一设备标识符
func generateHWID() string {
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

func hashString(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

// 保存生成的HWID，避免每次重新生成
var deviceHWID = generateHWID()

// UpdateHTTPClientConfig 更新HTTP客户端配置
func UpdateHTTPClientConfig(config *HTTPClientConfig) {
	globalConfig = config
	if globalConfig.UserAgent == "" {
		if globalConfig.EnableHWID && globalConfig.Version != "" {
			globalConfig.UserAgent = fmt.Sprintf("prizrak-box/%s", globalConfig.Version)
		} else {
			globalConfig.UserAgent = defaultUserAgent
		}
	}
}

// buildDeviceHeaders 构建设备信息头部
func buildDeviceHeaders() map[string]string {
	if !globalConfig.EnableHWID {
		return nil
	}

	headers := make(map[string]string)
	headers["x-hwid"] = deviceHWID

	if osName := normalizeDeviceOS(globalConfig.DeviceOS); osName != "" {
		headers["x-device-os"] = osName
	}
	if globalConfig.DeviceOSVer != "" {
		headers["x-ver-os"] = globalConfig.DeviceOSVer
	}
	if globalConfig.DeviceModel != "" {
		headers["x-device-model"] = globalConfig.DeviceModel
	}

	return headers
}

func normalizeDeviceOS(osName string) string {
	if osName == "" {
		return ""
	}

	lower := strings.ToLower(osName)
	switch {
	case strings.Contains(lower, "windows"):
		return "Windows"
	case strings.Contains(lower, "linux"):
		return "Linux"
	case strings.Contains(lower, "mac") || strings.Contains(lower, "darwin"):
		return "macOS"
	default:
		return osName
	}
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
	client, err := newHttpClient(proxyURL, timeout)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 添加设备信息头部（如果启用HWID）
	deviceHeaders := buildDeviceHeaders()
	for k, v := range deviceHeaders {
		req.Header.Set(k, v)
	}

	// 添加用户自定义头部
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 设置User-Agent（优先级：用户自定义 > 全局配置 > 默认值）
	if _, ok := headers["User-Agent"]; !ok {
		userAgent := globalConfig.UserAgent
		if userAgent == "" {
			userAgent = defaultUserAgent
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

	// 使用当前配置的User-Agent
	userAgent := globalConfig.UserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	resp, err := sendRequest(method, requestURL, map[string]string{"User-Agent": userAgent}, proxyURL, 8*time.Second)
	if err != nil {
		return 500, err
	}
	defer closeResponseBody(resp.Body)

	return resp.StatusCode, nil
}
