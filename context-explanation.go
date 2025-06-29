// 模拟 OAuth2 库内部的实现逻辑
package main

import (
    "context"
    "net/http"
    "golang.org/x/oauth2"
)

// 这就是 OAuth2 库内部类似的逻辑
func (c *Config) Exchange(ctx context.Context, code string) (*Token, error) {
    // 1. 首先检查 Context 中是否有自定义的 HTTP 客户端
    if client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok {
        // ✅ 找到了！使用我们提供的调试客户端
        return c.exchangeWithClient(client, code)
    } else {
        // ❌ 没找到，使用默认客户端
        return c.exchangeWithClient(http.DefaultClient, code)
    }
}

func (c *Config) exchangeWithClient(client *http.Client, code string) (*Token, error) {
    // 使用传入的客户端发起 HTTP 请求
    // 如果是我们的 debugClient，就会触发调试功能！
    resp, err := client.Post(c.Endpoint.TokenURL, "application/x-www-form-urlencoded", body)
    // ... 处理响应
    return token, err
}
