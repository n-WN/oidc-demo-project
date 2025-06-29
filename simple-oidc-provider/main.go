// main.go in simple-oidc-provider
// 这是一个迷你的、自包含的 OpenID Connect (OIDC) Provider 实现。
// 它用于本地开发和演示，取代像 Google 或 Okta 这样的外部认证服务。
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// --- 全局变量和配置 ---

var (
	// 用于签署 JWT 的 RSA 密钥对。在真实应用中，应从安全位置加载。
	privateKey *rsa.PrivateKey

	// 我们的 OP 的地址 (颁发者 URL)
	issuerURL = "http://127.0.0.1:9090"

	// 存储已注册的客户端信息 (代替数据库)
	clients = map[string]Client{
		"my-client-app": {
			ID:           "my-client-app",
			Secret:       "my-client-secret",
			RedirectURIs: []string{"http://127.0.0.1:8080/auth/callback"},
		},
	}

	// 存储用户信息 (代替数据库)
	users = map[string]User{
		"demo": {
			ID:       "user-123",
			Username: "demo",
			Password: "password", // 在真实应用中，请务必使用哈希存储！
			Name:     "本地认证的用户",
			Email:    "demo.user@example.com",
			Picture:  "https://www.gravatar.com/avatar/?d=mp", // 一个默认头像
		},
	}

	// 存储授权码 (代替数据库或 Redis)
	authCodes = make(map[string]AuthCodeData)
	mu        sync.Mutex
)

// --- 数据结构定义 ---

type Client struct {
	ID           string
	Secret       string
	RedirectURIs []string
}

type User struct {
	ID       string
	Username string
	Password string
	Name     string
	Email    string
	Picture  string
}

type AuthCodeData struct {
	ClientID string
	UserID   string
	Expiry   time.Time
}

// --- 主函数和服务器设置 ---

func main() {
	var err error
	// 1. 生成 RSA 密钥对用于 JWT 签名
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("无法生成 RSA 密钥: %v", err)
	}

	// 2. 设置 HTTP 路由
	http.HandleFunc("/.well-known/openid-configuration", handleDiscovery)
	http.HandleFunc("/jwks.json", handleJWKS)
	http.HandleFunc("/authorize", handleAuthorize)
	http.HandleFunc("/token", handleToken)
	http.HandleFunc("/login", handleLoginPage)
	http.HandleFunc("/consent", handleConsentPage)

	fmt.Println("OIDC Provider (认证服务) 正在监听 " + issuerURL)
	log.Fatal(http.ListenAndServe(":9090", nil))
}

// --- OIDC 核心端点实现 ---

// Endpoint 1: Discovery - 告诉客户端其他端点的位置
func handleDiscovery(w http.ResponseWriter, r *http.Request) {
	discovery := map[string]interface{}{
		"issuer":                 issuerURL,
		"authorization_endpoint": issuerURL + "/authorize",
		"token_endpoint":         issuerURL + "/token",
		"jwks_uri":               issuerURL + "/jwks.json",
		"userinfo_endpoint":      issuerURL + "/userinfo", // (本 Demo 未实现)
		"response_types_supported": []string{"code"},
		"subject_types_supported":  []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		// RS256 是我们使用的签名算法: RSA SHA-256
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discovery)
}

// Endpoint 2: JWKS - 提供用于验证 JWT 签名的公钥
func handleJWKS(w http.ResponseWriter, r *http.Request) {
	publicKey := &privateKey.PublicKey
	jwk := jose.JSONWebKey{
		Key:       publicKey,
		KeyID:     "my-signing-key-id", // 密钥 ID
		Algorithm: string(jose.RS256),
		Use:       "sig", // 用于签名
	}
	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{jwk},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

// Endpoint 3: Authorization - 用户登录和授权的入口
func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	
	// 验证客户端 ID 和重定向 URI 是否已注册
	if client, ok := clients[clientID]; !ok || !isValidRedirectURI(client, redirectURI) {
		http.Error(w, "无效的 client_id 或 redirect_uri", http.StatusBadRequest)
		return
	}

	// 重定向到登录页面，并将所有原始查询参数（如 state, scope 等）都传递过去
	loginURL := fmt.Sprintf("/login?%s", r.URL.RawQuery)
	http.Redirect(w, r, loginURL, http.StatusFound)
}

// Endpoint 4: Token - 客户端用授权码换取 ID Token
func handleToken(w http.ResponseWriter, r *http.Request) {
	// 1. 解析表单参数
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "无法解析表单", http.StatusBadRequest)
		return
	}
	code := r.PostForm.Get("code")
	clientID := r.PostForm.Get("client_id")
	clientSecret := r.PostForm.Get("client_secret")

	// 2. 验证客户端凭据
	client, ok := clients[clientID]
	if !ok || client.Secret != clientSecret {
		http.Error(w, "无效的客户端凭据", http.StatusUnauthorized)
		return
	}

	// 3. 验证授权码 (Authorization Code)
	mu.Lock()
	authData, ok := authCodes[code]
	delete(authCodes, code) // 授权码是一次性的，用完即删
	mu.Unlock()

	if !ok || authData.ClientID != clientID || time.Now().After(authData.Expiry) {
		http.Error(w, "无效或已过期的授权码", http.StatusBadRequest)
		return
	}

	// 4. 获取授权的用户信息
	user, ok := users[authData.UserID]
	if !ok {
		http.Error(w, "找不到用户", http.StatusInternalServerError)
		return
	}

	// 5. 创建并签名 ID Token (JWT)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		http.Error(w, "创建签名器失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	claims := map[string]interface{}{
		"iss":     issuerURL,
		"sub":     user.ID,
		"aud":     clientID,
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
		"name":    user.Name,
		"email":   user.Email,
		"picture": user.Picture,
	}

	rawJWT, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		http.Error(w, "创建 JWT 失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 6. 返回令牌
	tokenResponse := map[string]interface{}{
		"access_token": "dummy-access-token-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		"token_type":   "Bearer",
		"id_token":     rawJWT,
		"expires_in":   3600,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse)
}


// --- 辅助页面和函数 ---

// Page 1: 登录页面
func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `
			<h2>认证服务登录</h2>
			<form method="post" action="/login?%s">
				Username: <input type="text" name="username" value="demo"><br>
				Password: <input type="password" name="password" value="password"><br>
				<input type="submit" value="登录">
			</form>
		`, r.URL.RawQuery)
		return
	}

	// 处理登录逻辑
	r.ParseForm()
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	
	user, ok := users[username]
	if !ok || user.Password != password {
		http.Error(w, "无效的用户名或密码", http.StatusUnauthorized)
		return
	}

	// 登录成功，重定向到同意页面
	consentURL := fmt.Sprintf("/consent?%s", r.URL.RawQuery)
	http.Redirect(w, r, consentURL, http.StatusFound)
}

// Page 2: 同意授权页面
func handleConsentPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `
			<h2>授权请求</h2>
			<p>应用 <strong>%s</strong> 希望访问您的基本信息 (姓名, 邮箱, 头像)。</p>
			<form method="post" action="/consent?%s">
				<input type="submit" name="action" value="同意授权" style="background-color: #4CAF50; color: white; padding: 10px 20px; border: none; cursor: pointer;">
				<input type="submit" name="action" value="拒绝" style="padding: 10px 20px; cursor: pointer;">
			</form>
		`, q.Get("client_id"), r.URL.RawQuery)
		return
	}

	// 用户点击"同意授权"
	r.ParseForm()
	if r.FormValue("action") == "同意授权" {
		code := "code-" + fmt.Sprintf("%d", time.Now().UnixNano()) // 简单生成 code
		mu.Lock()
		authCodes[code] = AuthCodeData{
			ClientID: q.Get("client_id"),
			UserID:   "demo", // 简化：总是 demo 用户
			Expiry:   time.Now().Add(5 * time.Minute),
		}
		mu.Unlock()

		// 重定向回客户端应用的回调地址，并带上 code 和 state
		redirectURI := fmt.Sprintf("%s?code=%s&state=%s", q.Get("redirect_uri"), code, q.Get("state"))
		http.Redirect(w, r, redirectURI, http.StatusFound)
	} else {
		http.Error(w, "用户拒绝授权", http.StatusForbidden)
	}
}

// Helper: 验证重定向 URI 是否合法
func isValidRedirectURI(client Client, uri string) bool {
	for _, validURI := range client.RedirectURIs {
		if uri == validURI {
			return true
		}
	}
	return false
}