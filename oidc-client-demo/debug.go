// debug.go - HTTP 调试传输层
// 这个文件定义了一个自定义的 http.Transport，用于拦截和打印 HTTP 请求和响应的详细信息。
// 这种实现方式是 Go 中进行网络调试的常用模式，它利用了装饰器模式，在不改变核心业务逻辑的情况下，为 HTTP 客户端增加了日志记录功能。
package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// debugTransport 是一个自定义的 HTTP Transport，它包装了另一个 http.RoundTripper（通常是 http.DefaultTransport）。
// 它的主要作用是在实际发送请求之前和接收到响应之后，记录详细的请求和响应信息。
// 这是一个典型的装饰器模式应用：它为现有的 RoundTripper 添加了新的行为（日志记录）。
type debugTransport struct {
	// Transport 是被包装的底层 http.RoundTripper，负责实际的 HTTP 请求发送。
	// 通过持有接口类型，我们可以包装任何实现了 RoundTripper 接口的对象，例如 http.DefaultTransport 或其他自定义 transport。
	Transport http.RoundTripper
	// Decoder 是一个智能解码器，用于尝试解析和格式化 HTTP body（例如，美化 JSON 或解码 JWT）。
	Decoder *SmartDecoder
}

// NewDebugTransport 创建并初始化一个新的 debugTransport 实例。
// 它将 http.DefaultTransport 设置为底层的 transport，这是 Go HTTP 客户端的默认实现。
func NewDebugTransport() *debugTransport {
	return &debugTransport{
		Transport: http.DefaultTransport,
		Decoder:   NewSmartDecoder(),
	}
}

// RoundTrip 是 http.RoundTripper 接口的核心方法。
// RoundTrip: 往返
// 当 http.Client 使用这个 transport 发送请求时，此方法会被调用。
// 它的职责是“往返”一次 HTTP 事务：接收请求，获取响应。
func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// --- 请求记录 ---
	fmt.Println("\n=== HTTP 请求 ===")
	fmt.Printf("方法: %s\n", req.Method)
	fmt.Printf("URL: %s\n", req.URL.String())
	fmt.Println("请求头:")
	for k, v := range req.Header {
		fmt.Printf("  %s: %s\n", k, strings.Join(v, ", ")) // 将头部的值连接起来，以便更好地显示
	}

	// --- 请求体处理 ---
	// 检查请求方法是否为 "POST"，并且请求体 (req.Body) 是否存在。
	// req.Body 是一个 io.ReadCloser，它的内容只能被读取一次。
	if req.Method == "POST" && req.Body != nil {
		// 使用 io.ReadAll 读取请求体的所有内容。
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			// 打印原始的请求体内容。
			fmt.Printf("请求体 (原始): %s\n", string(bodyBytes))

			// 使用智能解码器尝试解析和格式化请求体。
			// 这对于调试 JSON API 或查看 JWT 内容非常有用。
			d.Decoder.SmartDecode("请求体 (解码后)", bodyBytes)

			// !!! 关键步骤：重建请求体 !!!
			// 因为 req.Body 已经被读取完毕，如果不重新创建它，后续的 http.Transport 将无法读取到任何内容，导致请求失败。
			// 我们使用 strings.NewReader 从已读取的字节切片中创建一个新的 io.Reader，
			// 然后用 io.NopCloser 包装它，使其满足 io.ReadCloser 接口（因为它不需要关闭任何底层资源）。
			req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
	}

	// --- 实际的 HTTP 请求 ---
	// 调用被包装的底层 Transport 的 RoundTrip 方法来实际发送请求。
	// 这是装饰器模式的核心，将工作委托给原始对象。
	resp, err := d.Transport.RoundTrip(req)
	if err != nil {
		// 如果请求过程中发生错误（例如，网络问题），记录错误并返回。
		fmt.Printf("请求错误: %v\n", err)
		return resp, err
	}

	// --- 响应记录 ---
	fmt.Println("\n--- HTTP 响应 ---")
	fmt.Printf("状态码: %s\n", resp.Status)
	fmt.Println("响应头:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
	}

	// --- 响应体处理 ---
	// 同样，检查响应体是否存在。
	if resp.Body != nil {
		// 读取响应体的所有内容。
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			// 打印原始的响应体。
			fmt.Printf("响应体 (原始): %s\n", string(bodyBytes))

			// 使用智能解码器解析和格式化响应体。
			d.Decoder.SmartDecode("响应体 (解码后)", bodyBytes)

			// !!! 关键步骤：重建响应体 !!!
			// 与请求体一样，响应体也被读取完毕。为了让调用 http.Client 的代码能够正常处理响应，
			// 我们必须用相同的内容重建 resp.Body。
			resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
	}
	fmt.Println("==================")
	fmt.Println() // 添加一个空行以分隔日志条目

	// 返回最终的响应和错误（在这里错误为 nil）。
	return resp, err
}
