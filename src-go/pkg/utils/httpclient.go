package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// 全局超时设置
var (
	ConnTimeOut      = 14 * time.Second
	DialTimeOut      = 5 * time.Second
	FastTimeOut      = 15 * time.Second
	defaultUserAgent = "clash-verge/v2.3.0"
	headPattern      = regexp.MustCompile(`204|blank|generate|gstatic`)
)

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
	return sendRequestWithOptions(method, requestURL, headers, proxyURL, timeout, false)
}

// sendRequestWithOptions 发送 HTTP 请求，支持设备头部选项
func sendRequestWithOptions(method, requestURL string, headers map[string]string, proxyURL string, timeout time.Duration, includeDeviceHeaders bool) (*http.Response, error) {
	client, err := newHttpClient(proxyURL, timeout)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 添加传入的头部
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 设置默认 User-Agent
	if _, ok := headers["User-Agent"]; !ok {
		req.Header.Set("User-Agent", defaultUserAgent)
	}

	// 添加设备信息头部（仅用于订阅请求）
	if includeDeviceHeaders {
		deviceHeaders := GetDeviceHeaders()
		for k, v := range deviceHeaders {
			// 只在没有自定义头部时添加设备头部
			if _, exists := headers[k]; !exists {
				req.Header.Set(k, v)
			}
		}
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

	resp, err := sendRequest(method, requestURL, map[string]string{"User-Agent": defaultUserAgent}, proxyURL, 8*time.Second)
	if err != nil {
		return 500, err
	}
	defer closeResponseBody(resp.Body)

	return resp.StatusCode, nil
}

// FastGetWithDeviceHeaders 并发 GET 请求（用于订阅），包含设备信息头部
// 根据 Device Info Headers Specification 添加以下头部：
// - x-hwid: 设备唯一标识符（必需）
// - x-device-os: 操作系统名称（可选）
// - x-ver-os: 操作系统版本（可选）
// - x-device-model: 设备型号（可选）
func FastGetWithDeviceHeaders(requestURL string, headers map[string]string, proxyURL string) (*ResponseResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), FastTimeOut)
	defer cancel()

	results := make(chan *ResponseResult, 2)
	errors := make(chan error, 2)

	send := func(useProxy bool) {
		var proxy string
		if useProxy {
			proxy = proxyURL
		}

		resp, err := sendRequestWithOptions("GET", requestURL, headers, proxy, ConnTimeOut, true)
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
