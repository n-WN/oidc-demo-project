// debug.go - HTTP 调试传输层
package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// debugTransport 是一个自定义的 HTTP Transport，用于记录请求和响应
type debugTransport struct {
	Transport http.RoundTripper
	Decoder   *SmartDecoder // 智能解码器
}

// NewDebugTransport 创建一个新的调试传输层
func NewDebugTransport() *debugTransport {
	return &debugTransport{
		Transport: http.DefaultTransport,
		Decoder:   NewSmartDecoder(),
	}
}

// RoundTrip 实现 http.RoundTripper 接口
func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 记录请求
	fmt.Printf("\n=== HTTP 请求 ===\n")
	fmt.Printf("方法: %s\n", req.Method)
	fmt.Printf("URL: %s\n", req.URL.String())
	fmt.Printf("请求头:\n")
	for k, v := range req.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}

	// 如果是 POST 请求，读取请求体
	if req.Method == "POST" && req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			fmt.Printf("请求体: %s\n", string(bodyBytes))

			// 智能解码和格式化
			d.Decoder.SmartDecode("请求体", bodyBytes)

			// 重新设置请求体，因为它只能读取一次
			req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
	}

	// 发送请求
	resp, err := d.Transport.RoundTrip(req)
	if err != nil {
		fmt.Printf("请求错误: %v\n", err)
		return resp, err
	}

	// 记录响应
	fmt.Printf("\n=== HTTP 响应 ===\n")
	fmt.Printf("状态码: %s\n", resp.Status)
	fmt.Printf("响应头:\n")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}

	// 读取响应体
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			fmt.Printf("响应体: %s\n", string(bodyBytes))

			// 智能解码和格式化
			d.Decoder.SmartDecode("响应体", bodyBytes)

			// 重新设置响应体
			resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
	}
	fmt.Printf("==================\n\n")

	return resp, err
}
