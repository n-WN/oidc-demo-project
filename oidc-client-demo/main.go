// main.go in oidc-demo
// 这是一个 OIDC 客户端 (Relying Party) 的演示应用。
// 它不管理用户密码，而是依赖于一个外部的 OIDC Provider (在这里是我们自建的服务) 来进行用户认证。
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// UserInfo 用于存储从 ID Token 中解析出的用户信息。
// 使用 omitempty 标签使 Picture 字段成为可选，如果 Provider 没有提供该信息，则不会出错。
type UserInfo struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture,omitempty"`
}

// 全局变量，用于存储 OAuth2 配置和 ID Token 验证器。
// 这些变量在 main 函数中初始化，确保它们在整个应用中可用。
var (
	// --- 连接到我们自己 OP 的配置 ---
	clientID     = "my-client-app"
	clientSecret = "my-client-secret"
	redirectURL  = "http://127.0.0.1:8080/auth/callback"

	// 全局变量，在 main 函数中初始化
	oauth2Config    *oauth2.Config
	idTokenVerifier *oidc.IDTokenVerifier
)

func main() {
	ctx := context.Background()

	// 1. 初始化 OIDC Provider - 连接到我们本地运行的认证服务
	provider, err := oidc.NewProvider(ctx, "http://127.0.0.1:9090")
	if err != nil {
		log.Fatalf("无法连接到 OIDC Provider: %v", err)
	}

	// 2. 配置 OAuth2 客户端
	oauth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		// 向 Provider 请求的权限范围 (scopes)。"openid" 是必须的。
		// 如果没有这个 scope, Provider 不会返回 ID Token, 也就是纯 OAuth2 流程
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}

	// 3. 创建 ID 令牌验证器
	idTokenVerifier = provider.Verifier(&oidc.Config{ClientID: clientID})

	// 4. 设置 HTTP 路由
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/auth/callback", handleCallback)
	http.HandleFunc("/logout", handleLogout)

	fmt.Println("OIDC Client App (客户端应用) 正在监听 http://127.0.0.1:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handleHome 是主页处理器，根据用户是否登录显示不同内容。
func handleHome(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user-info")
	// 如果没有会话 Cookie，显示未登录页面
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `
			<h2>欢迎来到 OIDC 客户端应用</h2>
			<p>您当前未登录。</p>
			<a href="/login" style="font-size: 1.2em; text-decoration: none; background-color: #007BFF; color: white; padding: 10px 15px; border-radius: 5px;">
				使用我们的认证服务登录
			</a>
		`)
		return
	}

	// 如果已登录，解码用户信息并显示欢迎页面
	var userInfo UserInfo
	data, _ := base64.StdEncoding.DecodeString(cookie.Value)
	json.Unmarshal(data, &userInfo)

	var pictureHTML string
	if userInfo.Picture != "" {
		pictureHTML = fmt.Sprintf(`<img src="%s" alt="Profile Picture" style="width:100px; border-radius: 50%%; margin-top: 10px;">`, html.EscapeString(userInfo.Picture))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, fmt.Sprintf(`
		<h2>欢迎, %s!</h2>
		<p>您的身份已由我们自己的 OIDC Provider 成功验证。</p>
		<p>邮箱: %s</p>
		%s
		<p style="margin-top: 20px;"><a href="/logout">退出登录</a></p>
	`, html.EscapeString(userInfo.Name), html.EscapeString(userInfo.Email), pictureHTML))
}

// handleLogin 启动 OIDC 登录流程。
func handleLogin(w http.ResponseWriter, r *http.Request) {
	// w 不是别名，而是：
	// - 参数名称/参数标识符
	// - 接收传入的 http.ResponseWriter 对象的变量名
	// - 在函数内部用来引用这个对象的名字

	// 1. 生成一个随机的 state 字符串，用于防止 CSRF 攻击。
	state, err := generateRandomString(32)
	if err != nil {
		http.Error(w, "生成 state 失败", http.StatusInternalServerError)
		return
	}

	// 2. 将 state 存入一个有时效性的 Cookie。
	http.SetCookie(w, &http.Cookie{
		Name:  "oauth-state",
		Value: state,
		Path:  "/",
		// 设置 Cookie 的有效期为 10 分钟
		MaxAge:   int(10 * time.Minute.Seconds()),
		HttpOnly: true,
	})

	// 3. 将用户重定向到 OIDC Provider 的授权页面。
	target := oauth2Config.AuthCodeURL(state)
	fmt.Printf("重定向用户到 OIDC Provider 的授权页面: %s\n", target)
	http.Redirect(w, r, target, http.StatusFound)
}

// handleCallback 是 OIDC 流程中的回调地址。
func handleCallback(w http.ResponseWriter, r *http.Request) {
	// 创建自定义 HTTP 客户端，用于调试网络请求
	debugClient := &http.Client{
		Transport: NewDebugTransport(),
		Timeout:   30 * time.Second,
	}

	// 将自定义客户端绑定到 context
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, debugClient)

	// 1. 验证 state 参数，确保请求是由我们自己发起的，防止 CSRF。
	stateFromCookie, err := r.Cookie("oauth-state")
	if err != nil {
		http.Error(w, "State cookie 丢失", http.StatusBadRequest)
		return
	}
	// Query(): 解析 URL 中的查询参数
	if r.URL.Query().Get("state") != stateFromCookie.Value {
		http.Error(w, "无效的 state 参数", http.StatusBadRequest)
		return
	}

	// 2. 从 URL 中获取授权码，并用它来向 Provider 交换令牌。
	code := r.URL.Query().Get("code")

	// 添加简单的调试信息
	fmt.Printf("\n🚀 开始令牌交换...\n")
	fmt.Printf("🔑 授权码: %s\n", code)
	fmt.Printf("⏰ 时间: %s\n", time.Now().Format("15:04:05"))

	startTime := time.Now()

	// Exchange() 方法会使用授权码与 OIDC Provider 交换访问令牌和 ID Token
	oauth2Token, err := oauth2Config.Exchange(ctx, code)

	duration := time.Since(startTime)
	fmt.Printf("⏱️ 请求耗时: %v\n", duration)

	if err != nil {
		fmt.Printf("❌ 令牌交换失败: %v\n", err)
		http.Error(w, "交换令牌失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("✅ 令牌交换成功!\n")
	fmt.Printf("🎫 AccessToken 前缀: %s...\n", oauth2Token.AccessToken[:min(20, len(oauth2Token.AccessToken))])
	fmt.Printf("🏷️ TokenType: %s\n", oauth2Token.TokenType)
	fmt.Printf("⏰ 过期时间: %s\n", oauth2Token.Expiry.Format("15:04:05"))

	// 3. 从令牌响应中提取 ID Token。
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "令牌响应中没有找到 id_token", http.StatusInternalServerError)
		return
	}

	// 4. 验证 ID Token。这是 OIDC 的核心安全步骤。
	// Verifier 会检查签名、颁发者(iss)、受众(aud)、有效期等。
	idToken, err := idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, "验证 ID Token 失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. 从验证通过的 ID Token 中提取用户信息 (claims)。
	var claims UserInfo
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "解析用户信息失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 6. 将用户信息存入一个安全的会话 Cookie，标志用户已登录。
	jsonData, err := json.Marshal(claims)
	if err != nil {
		http.Error(w, "序列化用户信息失败", http.StatusInternalServerError)
		return
	}
	encodedData := base64.StdEncoding.EncodeToString(jsonData)
	http.SetCookie(w, &http.Cookie{
		Name:     "user-info",
		Value:    encodedData,
		Path:     "/",
		MaxAge:   int(time.Hour.Seconds()),
		HttpOnly: true,
	})

	// 7. 重定向到主页，此时用户已经是登录状态。
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleLogout 用于清除会话 Cookie，实现退出登录。
func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user-info",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // 设置为过去的某个时间点，使 Cookie立即失效
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// generateRandomString 是一个生成随机字符串的工具函数。
func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
