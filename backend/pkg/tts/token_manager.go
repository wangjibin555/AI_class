package tts

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// TokenManager 阿里云TTS Token管理器
type TokenManager struct {
	AccessKeyID     string
	AccessKeySecret string
	AppKey          string
	BaseURL         string
	client          *http.Client
}

// TokenResponse Token响应结构
type TokenResponse struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

// NewTokenManager 创建Token管理器
func NewTokenManager(accessKeyID, accessKeySecret, appKey, baseURL string) *TokenManager {
	return &TokenManager{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		AppKey:          appKey,
		BaseURL:         baseURL,
		client:          &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAccessToken 获取Access Token
func (tm *TokenManager) GetAccessToken() (string, error) {
	// 构建请求URL
	tokenURL := fmt.Sprintf("%s/token", strings.TrimSuffix(tm.BaseURL, "/stream/v1/tts"))

	// 构建请求参数
	params := map[string]string{
		"appkey": tm.AppKey,
	}

	// 生成签名
	signature := tm.generateSignature("GET", tokenURL, params)

	// 构建完整URL
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	query.Set("signature", signature)

	fullURL := fmt.Sprintf("%s?%s", tokenURL, query.Encode())

	// 发送请求
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NLS-Token", tm.AccessKeyID)

	// 发送请求
	resp, err := tm.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP错误 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	return tokenResp.Token, nil
}

// generateSignature 生成签名
func (tm *TokenManager) generateSignature(method, url string, params map[string]string) string {
	// 按参数名排序
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建参数字符串
	var paramStr strings.Builder
	for _, key := range keys {
		if paramStr.Len() > 0 {
			paramStr.WriteString("&")
		}
		paramStr.WriteString(fmt.Sprintf("%s=%s", key, params[key]))
	}

	// 构建签名字符串
	signStr := fmt.Sprintf("%s&%s&%s", method, url, paramStr.String())

	// 使用HMAC-SHA1生成签名
	h := hmac.New(sha1.New, []byte(tm.AccessKeySecret))
	h.Write([]byte(signStr))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// ValidateToken 验证Token是否有效
func (tm *TokenManager) ValidateToken(token string) error {
	// 构建测试请求
	testURL := fmt.Sprintf("%s/stream/v1/tts", tm.BaseURL)

	// 构建测试参数
	params := map[string]interface{}{
		"appkey":      tm.AppKey,
		"text":        "测试",
		"format":      "mp3",
		"sample_rate": 16000,
		"voice":       "zhimao",
	}

	// 序列化参数
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("序列化参数失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", testURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("X-NLS-Token", token)

	// 发送请求
	resp, err := tm.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应
	if resp.StatusCode == http.StatusOK {
		contentType := resp.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "audio/") {
			return nil // Token有效
		}
	}

	return fmt.Errorf("Token验证失败: %s", string(respBody))
}
