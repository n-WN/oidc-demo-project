// decoder.go - 智能解码器
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// SmartDecoder 智能解码器结构体
type SmartDecoder struct{}

// NewSmartDecoder 创建一个新的智能解码器
func NewSmartDecoder() *SmartDecoder {
	return &SmartDecoder{}
}

// SmartDecode 智能解码和格式化数据
func (s *SmartDecoder) SmartDecode(label string, data []byte) {
	content := string(data)
	
	// 1. 尝试 JSON 格式化 (最常见)
	if s.tryJSONFormat(label, data) {
		return
	}
	
	// 2. 尝试 URL 解码
	if s.tryURLDecode(label, content) {
		return
	}
	
	// 3. 尝试 Base64 解码
	if s.tryBase64Decode(label, content) {
		return
	}
	
	// 4. 尝试 JWT 解码
	if s.tryJWTDecode(label, content) {
		return
	}
}

// tryJSONFormat 尝试 JSON 格式化
func (s *SmartDecoder) tryJSONFormat(label string, data []byte) bool {
	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "{") && !strings.HasPrefix(content, "[") {
		return false
	}
	
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
		fmt.Printf("🎨 格式化后的%s:\n%s\n", label, prettyJSON)
		
		// 检查JSON中是否包含JWT令牌字段
		s.findAndDecodeJWTsInJSON(jsonData)
		return true
	}
	return false
}

// tryURLDecode 尝试 URL 解码
func (s *SmartDecoder) tryURLDecode(label string, content string) bool {
	if !strings.Contains(content, "%") {
		return false
	}
	
	if decoded, err := url.QueryUnescape(content); err == nil && decoded != content {
		fmt.Printf("🔓 URL解码后的%s: %s\n", label, decoded)
		
		// 递归尝试解码解码后的内容
		if strings.Contains(decoded, "%") {
			s.tryURLDecode(label+"(再次解码)", decoded)
		}
		return true
	}
	return false
}

// tryBase64Decode 尝试 Base64 解码
func (s *SmartDecoder) tryBase64Decode(label string, content string) bool {
	// 简单检查是否可能是 Base64
	if len(content) < 4 || len(content)%4 != 0 {
		return false
	}
	
	// 检查是否包含 Base64 字符
	base64Chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	if !s.containsOnlyChars(content, base64Chars) {
		return false
	}
	
	if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
		decodedStr := string(decoded)
		fmt.Printf("🔐 Base64解码后的%s: %s\n", label, decodedStr)
		
		// 递归尝试解码解码后的内容
		s.SmartDecode(label+"(Base64解码后)", decoded)
		return true
	}
	return false
}

// tryJWTDecode 尝试 JWT 解码
func (s *SmartDecoder) tryJWTDecode(label string, content string) bool {
	// JWT 格式: header.payload.signature
	parts := strings.Split(content, ".")
	if len(parts) != 3 {
		return false
	}
	
	fmt.Printf("🎫 检测到JWT令牌:\n")
	
	// 解码 header
	if headerData, err := base64.RawURLEncoding.DecodeString(parts[0]); err == nil {
		fmt.Printf("  📋 Header: %s\n", headerData)
	}
	
	// 解码 payload
	if payloadData, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
		fmt.Printf("  📦 Payload: %s\n", payloadData)
		
		// 尝试格式化 payload JSON
		var payloadJSON interface{}
		if err := json.Unmarshal(payloadData, &payloadJSON); err == nil {
			prettyPayload, _ := json.MarshalIndent(payloadJSON, "    ", "  ")
			fmt.Printf("  🎨 格式化的Payload:\n%s\n", prettyPayload)
		}
	}
	
	fmt.Printf("  🔏 Signature: %s\n", parts[2])
	return true
}

// containsOnlyChars 检查字符串是否只包含指定字符
func (s *SmartDecoder) containsOnlyChars(str, chars string) bool {
	for _, r := range str {
		if !strings.ContainsRune(chars, r) {
			return false
		}
	}
	return true
}

// findAndDecodeJWTsInJSON 在JSON数据中查找并解码JWT令牌
func (s *SmartDecoder) findAndDecodeJWTsInJSON(data interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if str, ok := value.(string); ok {
				// 检查常见的JWT字段名
				if s.isJWTField(key) && s.looksLikeJWT(str) {
					fmt.Printf("\n🎫 发现%s中的JWT令牌: %s\n", key, key)
					s.decodeJWTDetailed(str)
				}
			} else {
				// 递归检查嵌套对象
				s.findAndDecodeJWTsInJSON(value)
			}
		}
	case []interface{}:
		for _, item := range v {
			s.findAndDecodeJWTsInJSON(item)
		}
	}
}

// isJWTField 检查字段名是否可能包含JWT
func (s *SmartDecoder) isJWTField(fieldName string) bool {
	jwtFields := []string{
		"id_token", "access_token", "refresh_token", 
		"token", "jwt", "bearer", "authorization",
	}
	
	fieldLower := strings.ToLower(fieldName)
	for _, jwtField := range jwtFields {
		if strings.Contains(fieldLower, jwtField) {
			return true
		}
	}
	return false
}

// looksLikeJWT 检查字符串是否看起来像JWT
func (s *SmartDecoder) looksLikeJWT(str string) bool {
	parts := strings.Split(str, ".")
	return len(parts) == 3 && len(str) > 50 // JWT通常比较长
}

// decodeJWTDetailed 详细解码JWT令牌
func (s *SmartDecoder) decodeJWTDetailed(token string) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return
	}
	
	fmt.Printf("┌─ JWT解码详情 ─────────────────────────────────\n")
	
	// 解码 Header
	if headerData, err := base64.RawURLEncoding.DecodeString(parts[0]); err == nil {
		var headerJSON interface{}
		if err := json.Unmarshal(headerData, &headerJSON); err == nil {
			prettyHeader, _ := json.MarshalIndent(headerJSON, "│ ", "  ")
			fmt.Printf("│ 📋 Header:\n│ %s\n", prettyHeader)
		} else {
			fmt.Printf("│ 📋 Header: %s\n", headerData)
		}
	}
	
	fmt.Printf("├─────────────────────────────────────────────────\n")
	
	// 解码 Payload
	if payloadData, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
		var payloadJSON interface{}
		if err := json.Unmarshal(payloadData, &payloadJSON); err == nil {
			prettyPayload, _ := json.MarshalIndent(payloadJSON, "│ ", "  ")
			fmt.Printf("│ 📦 Payload:\n│ %s\n", prettyPayload)
			
			// 解析常见的JWT声明
			if claims, ok := payloadJSON.(map[string]interface{}); ok {
				s.explainJWTClaims(claims)
			}
		} else {
			fmt.Printf("│ 📦 Payload: %s\n", payloadData)
		}
	}
	
	fmt.Printf("├─────────────────────────────────────────────────\n")
	fmt.Printf("│ 🔏 Signature: %s\n", parts[2])
	fmt.Printf("└─────────────────────────────────────────────────\n\n")
}

// explainJWTClaims 解释JWT中的常见声明
func (s *SmartDecoder) explainJWTClaims(claims map[string]interface{}) {
	fmt.Printf("│ 💡 JWT声明解释:\n")
	
	claimExplanations := map[string]string{
		"iss": "颁发者 (Issuer)",
		"sub": "主题/用户ID (Subject)", 
		"aud": "受众/客户端ID (Audience)",
		"exp": "过期时间 (Expiration)",
		"iat": "颁发时间 (Issued At)",
		"nbf": "生效时间 (Not Before)",
		"jti": "JWT ID",
		"name": "用户姓名",
		"email": "用户邮箱",
		"picture": "用户头像",
		"preferred_username": "首选用户名",
	}
	
	for claim, value := range claims {
		if explanation, exists := claimExplanations[claim]; exists {
			// 处理时间戳
			if claim == "exp" || claim == "iat" || claim == "nbf" {
				if timestamp, ok := value.(float64); ok {
					t := time.Unix(int64(timestamp), 0)
					fmt.Printf("│   • %s (%s): %v → %s\n", claim, explanation, value, t.Format("2006-01-02 15:04:05"))
					continue
				}
			}
			fmt.Printf("│   • %s (%s): %v\n", claim, explanation, value)
		}
	}
}
