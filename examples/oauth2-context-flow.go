// oauth2-context-flow.go
// 完整追踪 OAuth2 库中 Context 的流转过程
// 基于真实的 golang.org/x/oauth2/internal/transport.go 源码分析

package main

import (
	"context"
	"fmt"
	"net/http"
)

// ===== 1. 真实 OAuth2 库的 internal/transport.go 实现 =====

// ContextKey 这是 OAuth2 库真实使用的类型
// 源码位置: golang.org/x/oauth2/internal/transport.go:18
type ContextKey struct{}

// HTTPClient 这是 OAuth2 库导出的 Context 键
// 源码位置: golang.org/x/oauth2/internal/transport.go:14-15
var HTTPClient ContextKey

// ContextClient 这就是您追踪到的关键函数！
// 源码位置: golang.org/x/oauth2/internal/transport.go:20-27
// 这个函数展示了 Context.Value() 方法的实际使用
func ContextClient(ctx context.Context) *http.Client {
	fmt.Println("🔍 [transport.go:ContextClient] 函数被调用")
	fmt.Println("🔍 [transport.go:ContextClient] 这是 OAuth2 库的核心依赖注入机制")
	
	if ctx != nil {
		fmt.Println("🔍 [transport.go:ContextClient] 检查 context 非空 ✓")
		
		// 🎯 关键代码：这里调用了 Context 接口的 Value() 方法
		fmt.Println("🔍 [transport.go:ContextClient] 调用 ctx.Value(HTTPClient)")
		fmt.Println("🔍 [transport.go:ContextClient] 这就是 Context.Value() 的实际使用场景！")
		
		if hc, ok := ctx.Value(HTTPClient).(*http.Client); ok {
			fmt.Println("✅ [transport.go:ContextClient] 成功提取自定义 HTTP 客户端")
			fmt.Printf("✅ [transport.go:ContextClient] 客户端类型: %T\n", hc)
			fmt.Printf("✅ [transport.go:ContextClient] Transport 类型: %T\n", hc.Transport)
			return hc  // 返回我们注入的调试客户端
		} else {
			fmt.Println("❌ [transport.go] 未找到自定义 HTTP 客户端")
		}
	} else {
		fmt.Println("❌ [transport.go] context 为 nil")
	}
	
	fmt.Println("🔄 [transport.go] 回退到 http.DefaultClient")
	return http.DefaultClient
}

// ===== 2. 模拟 OAuth2 库的主要 Exchange 函数 =====

type Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
}

type Token struct {
	AccessToken string
	TokenType   string
}

// 模拟 OAuth2 的 Exchange 方法
func (c *Config) Exchange(ctx context.Context, code string) (*Token, error) {
	fmt.Println("\n🚀 [OAuth2.Exchange] 开始令牌交换流程")
	fmt.Printf("🚀 [OAuth2.Exchange] 授权码: %s\n", code)
	
	// 🎯 关键步骤：调用您发现的 ContextClient 函数
	fmt.Println("🚀 [OAuth2.Exchange] 调用 ContextClient(ctx) 获取 HTTP 客户端")
	client := ContextClient(ctx)
	
	fmt.Printf("🚀 [OAuth2.Exchange] 获得 HTTP 客户端: %T\n", client)
	
	// 模拟使用客户端发起请求
	fmt.Println("🚀 [OAuth2.Exchange] 使用客户端发起 POST 请求到令牌端点")
	
	// 这里会触发我们的调试传输层
	resp, err := client.Post(c.TokenURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		fmt.Printf("❌ [OAuth2.Exchange] 请求失败: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()
	
	fmt.Println("✅ [OAuth2.Exchange] 请求成功，解析令牌")
	
	return &Token{
		AccessToken: "mock_token_from_" + code,
		TokenType:   "Bearer",
	}, nil
}

// ===== 3. 模拟我们的调试传输层 =====

type DebugTransport struct {
	Transport http.RoundTripper
}

func (d *DebugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println("\n🔍 [DebugTransport] 拦截到 HTTP 请求!")
	fmt.Printf("🔍 [DebugTransport] 方法: %s\n", req.Method)
	fmt.Printf("🔍 [DebugTransport] URL: %s\n", req.URL.String())
	fmt.Println("🔍 [DebugTransport] 这里可以记录详细的请求信息")
	
	// 调用底层传输层
	resp, err := d.Transport.RoundTrip(req)
	
	if err == nil {
		fmt.Printf("🔍 [DebugTransport] 响应状态: %s\n", resp.Status)
		fmt.Println("🔍 [DebugTransport] 这里可以记录详细的响应信息")
	}
	
	return resp, err
}

// ===== 4. 完整流程演示 =====

func demonstrateFullFlow() {
	fmt.Println("=== OAuth2 Context 流转完整追踪 ===\n")
	
	// 步骤 1: 我们创建调试客户端
	fmt.Println("👨‍💻 [用户代码] 创建调试 HTTP 客户端")
	debugClient := &http.Client{
		Transport: &DebugTransport{
			Transport: http.DefaultTransport,
		},
	}
	
	// 步骤 2: 使用 WithValue 注入客户端
	fmt.Println("👨‍💻 [用户代码] 使用 context.WithValue() 注入客户端")
	ctx := context.WithValue(context.Background(), HTTPClient, debugClient)
	
	// 步骤 3: 创建 OAuth2 配置
	fmt.Println("👨‍💻 [用户代码] 创建 OAuth2 配置")
	config := &Config{
		ClientID:     "demo-client",
		ClientSecret: "demo-secret",
		TokenURL:     "https://provider.example/token",
	}
	
	// 步骤 4: 调用 Exchange - 这里会触发整个流程
	fmt.Println("👨‍💻 [用户代码] 调用 oauth2Config.Exchange()")
	token, err := config.Exchange(ctx, "auth_code_123")
	
	if err != nil {
		fmt.Printf("❌ 最终错误: %v\n", err)
		return
	}
	
	fmt.Printf("\n🎉 最终结果: 成功获得令牌 %s\n", token.AccessToken)
	fmt.Println("🎉 调试功能成功触发!")
}

// ===== 5. 对比：不使用自定义客户端的情况 =====

func demonstrateWithoutCustomClient() {
	fmt.Println("\n=== 对比：不注入自定义客户端 ===\n")
	
	// 使用空的 context
	ctx := context.Background()
	
	config := &Config{
		ClientID:     "demo-client",
		ClientSecret: "demo-secret",
		TokenURL:     "https://provider.example/token",
	}
	
	fmt.Println("👨‍💻 [用户代码] 使用空 context 调用 Exchange")
	token, err := config.Exchange(ctx, "auth_code_456")
	
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		return
	}
	
	fmt.Printf("✅ 结果: 获得令牌 %s（使用默认客户端）\n", token.AccessToken)
}

// ===== 6. Context 键的唯一性演示 =====

func demonstrateKeyUniqueness() {
	fmt.Println("\n=== Context 键的唯一性演示 ===\n")
	
	// 创建两个相同内容但不同类型的键
	type MyContextKey struct{}
	var myKey MyContextKey
	
	ctx := context.Background()
	
	// 使用不同的键存储值
	ctx = context.WithValue(ctx, HTTPClient, "oauth2-client")
	ctx = context.WithValue(ctx, myKey, "my-client")
	
	// 尝试读取
	oauth2Value := ctx.Value(HTTPClient)
	myValue := ctx.Value(myKey)
	
	fmt.Printf("OAuth2 键的值: %v\n", oauth2Value)
	fmt.Printf("我的键的值: %v\n", myValue)
	
	// 证明键的唯一性
	fmt.Println("✅ 不同类型的键可以共存，不会冲突")
}

func main() {
	demonstrateFullFlow()
	demonstrateWithoutCustomClient()
	demonstrateKeyUniqueness()
	
	fmt.Println("\n🎯 您的发现总结:")
	fmt.Println("• OAuth2 库通过 internal/transport.go 的 ContextClient() 函数")
	fmt.Println("• 调用 ctx.Value(HTTPClient) 读取我们注入的客户端")
	fmt.Println("• 这就是 Context.Value() 方法的实际使用场景!")
	fmt.Println("• WithValue() 写入，Value() 读取，完美配合!")
	
	fmt.Println("\n🪝 Context vs Hook 模式对比:")
	fmt.Println("相似之处:")
	fmt.Println("  • 都是非侵入式扩展机制")
	fmt.Println("  • 都允许运行时改变行为") 
	fmt.Println("  • 都实现了松耦合设计")
	fmt.Println("不同之处:")
	fmt.Println("  • Context: 数据传递 + 类型安全")
	fmt.Println("  • Hook: 函数回调 + 动态灵活")
	fmt.Println("  • Context 更适合 Go 的静态类型系统")
}
