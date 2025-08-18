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

// TokenService 阿里云Token服务
type TokenService struct {
	accessKeyID     string
	accessKeySecret string
	endpoint        string
	client          *http.Client
}

// AliyunTokenResponse 阿里云Token响应
type AliyunTokenResponse struct {
	ErrMsg string `json:"ErrMsg"`
	Token  struct {
		UserId     string `json:"UserId"`
		Id         string `json:"Id"`
		ExpireTime int64  `json:"ExpireTime"`
	} `json:"Token"`
	ErrorCode    string `json:"ErrorCode,omitempty"`
	ErrorMessage string `json:"ErrorMessage,omitempty"`
}

// NewTokenService 创建Token服务
func NewTokenService(accessKeyID, accessKeySecret string) *TokenService {
	return &TokenService{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		endpoint:        "https://nls-meta.cn-shanghai.aliyuncs.com",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetToken 获取访问Token
func (ts *TokenService) GetToken() (string, error) {
	// 构建请求参数
	params := map[string]string{
		"AccessKeyId":      ts.accessKeyID,
		"Action":           "CreateToken",
		"Format":           "JSON",
		"RegionId":         "cn-shanghai",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"SignatureVersion": "1.0",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Version":          "2019-02-28",
	}

	// 生成签名
	signature, err := ts.generateSignature("POST", params)
	if err != nil {
		return "", fmt.Errorf("生成签名失败: %w", err)
	}

	params["Signature"] = signature

	// 构建请求体
	formData := url.Values{}
	for key, value := range params {
		formData.Set(key, value)
	}

	// 发送请求
	resp, err := ts.client.PostForm(ts.endpoint, formData)
	if err != nil {
		return "", fmt.Errorf("发送Token请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取Token响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Token请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var tokenResp AliyunTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("解析Token响应失败: %w, 响应: %s", err, string(body))
	}

	if tokenResp.ErrorCode != "" {
		return "", fmt.Errorf("Token获取失败: %s - %s", tokenResp.ErrorCode, tokenResp.ErrorMessage)
	}

	if tokenResp.ErrMsg != "" {
		return "", fmt.Errorf("Token获取失败: %s", tokenResp.ErrMsg)
	}

	if tokenResp.Token.Id == "" {
		return "", fmt.Errorf("Token为空，响应: %s", string(body))
	}

	return tokenResp.Token.Id, nil
}

// generateSignature 生成阿里云API签名
func (ts *TokenService) generateSignature(method string, params map[string]string) (string, error) {
	// 1. 构建待签名字符串
	sortedKeys := make([]string, 0, len(params))
	for key := range params {
		if key != "Signature" {
			sortedKeys = append(sortedKeys, key)
		}
	}
	sort.Strings(sortedKeys)

	var queryParts []string
	for _, key := range sortedKeys {
		value := params[key]
		queryParts = append(queryParts, url.QueryEscape(key)+"="+url.QueryEscape(value))
	}

	canonicalizedQueryString := strings.Join(queryParts, "&")

	// 2. 构建待签名字符串
	stringToSign := method + "&" + url.QueryEscape("/") + "&" + url.QueryEscape(canonicalizedQueryString)

	// 3. 计算签名
	key := ts.accessKeySecret + "&"
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature, nil
}

// CachedTokenService 带缓存的Token服务
type CachedTokenService struct {
	tokenService *TokenService
	cachedToken  string
	expireTime   time.Time
}

// NewCachedTokenService 创建带缓存的Token服务
func NewCachedTokenService(accessKeyID, accessKeySecret string) *CachedTokenService {
	return &CachedTokenService{
		tokenService: NewTokenService(accessKeyID, accessKeySecret),
	}
}

// GetToken 获取Token（带缓存）
func (cts *CachedTokenService) GetToken() (string, error) {
	// 检查缓存的Token是否有效
	if cts.cachedToken != "" && time.Now().Before(cts.expireTime) {
		return cts.cachedToken, nil
	}

	// 获取新的Token
	token, err := cts.tokenService.GetToken()
	if err != nil {
		return "", err
	}

	// 缓存Token（设置过期时间为23小时，留1小时缓冲）
	cts.cachedToken = token
	cts.expireTime = time.Now().Add(23 * time.Hour)

	return token, nil
}

// ClearCache 清除缓存
func (cts *CachedTokenService) ClearCache() {
	cts.cachedToken = ""
	cts.expireTime = time.Time{}
}
