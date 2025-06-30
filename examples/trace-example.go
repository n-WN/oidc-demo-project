package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
)

// 使用 httptrace 追踪网络请求的示例
func createTracedContext() context.Context {
	trace := &httptrace.ClientTrace{
		// DNS 查询开始
		DNSStart: func(info httptrace.DNSStartInfo) {
			fmt.Printf("🔍 DNS 查询开始: %s\n", info.Host)
		},

		// DNS 查询完成
		DNSDone: func(info httptrace.DNSDoneInfo) {
			fmt.Printf("✅ DNS 查询完成: %v, 错误: %v\n", info.Addrs, info.Err)
		},

		// 开始连接
		ConnectStart: func(network, addr string) {
			fmt.Printf("🔌 开始连接: %s %s\n", network, addr)
		},

		// 连接完成
		ConnectDone: func(network, addr string, err error) {
			fmt.Printf("✅ 连接完成: %s %s, 错误: %v\n", network, addr, err)
		},

		// 获得连接
		GotConn: func(info httptrace.GotConnInfo) {
			fmt.Printf("🌐 获得连接: 本地地址=%s, 远程地址=%s, 复用=%v\n",
				info.Conn.LocalAddr(), info.Conn.RemoteAddr(), info.Reused)
		},

		// 写入请求
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			fmt.Printf("📤 请求已发送完成, 错误: %v\n", info.Err)
		},

		// 收到第一个响应字节
		GotFirstResponseByte: func() {
			fmt.Printf("📥 收到第一个响应字节\n")
		},
	}

	return httptrace.WithClientTrace(context.Background(), trace)
}

// 演示 httptrace 功能的完整示例
func main() {
	fmt.Println("🚀 HTTP Trace 示例程序")
	fmt.Println("====================================")

	// 创建带有追踪功能的 context
	ctx := createTracedContext()

	// 创建一个简单的 HTTP 请求来演示追踪功能
	fmt.Println("\n📡 发起 HTTP 请求到 httpbin.org...")

	// 使用自定义 context 发起请求
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/ip", nil)
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}

	fmt.Println("\n🔍 开始追踪网络请求细节:")
	fmt.Println("----------------------------------------")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("----------------------------------------")
	fmt.Printf("✅ 请求完成! 状态码: %d\n", resp.StatusCode)

	// 读取响应内容
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	fmt.Printf("📄 响应内容: %s\n", string(body[:n]))

	fmt.Println("\n💡 在 OIDC 客户端中使用方法:")
	fmt.Println("   ctx := createTracedContext()")
	fmt.Println("   oauth2Token, err := oauth2Config.Exchange(ctx, code)")
}
