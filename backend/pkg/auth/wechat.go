package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 微信小程序配置
type WechatConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// 微信会话信息
type WechatSession struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
	ErrCode    int    `json:"errcode,omitempty"`
	ErrMsg     string `json:"errmsg,omitempty"`
}

// 微信用户信息
type WechatUserInfo struct {
	OpenID    string `json:"openId"`
	NickName  string `json:"nickName"`
	Gender    int    `json:"gender"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Country   string `json:"country"`
	AvatarURL string `json:"avatarUrl"`
	UnionID   string `json:"unionId,omitempty"`
	Watermark struct {
		AppID     string `json:"appid"`
		Timestamp int64  `json:"timestamp"`
	} `json:"watermark"`
}

// 微信认证客户端
type WechatClient struct {
	config     WechatConfig
	httpClient *http.Client
}

// NewWechatClient 创建微信认证客户端
func NewWechatClient(appID, appSecret string) *WechatClient {
	return &WechatClient{
		config: WechatConfig{
			AppID:     appID,
			AppSecret: appSecret,
		},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetSession 通过code获取session
func (c *WechatClient) GetSession(code string) (*WechatSession, error) {
	if code == "" {
		return nil, errors.New("code cannot be empty")
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		c.config.AppID, c.config.AppSecret, code)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to request wechat API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var session WechatSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 检查微信API错误
	if session.ErrCode != 0 {
		return nil, fmt.Errorf("wechat API error: %d - %s", session.ErrCode, session.ErrMsg)
	}

	if session.OpenID == "" {
		return nil, errors.New("invalid response: openid is empty")
	}

	return &session, nil
}

// ValidateUserInfo 验证用户信息
func (c *WechatClient) ValidateUserInfo(encryptedData, iv, sessionKey string) (*WechatUserInfo, error) {
	if encryptedData == "" || iv == "" || sessionKey == "" {
		return nil, errors.New("encryptedData, iv and sessionKey cannot be empty")
	}

	userInfo, err := c.decryptUserInfo(encryptedData, iv, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt user info: %w", err)
	}

	// 验证水印
	if err := c.validateWatermark(&userInfo.Watermark); err != nil {
		return nil, fmt.Errorf("watermark validation failed: %w", err)
	}

	return userInfo, nil
}

// decryptUserInfo 解密用户信息
func (c *WechatClient) decryptUserInfo(encryptedData, iv, sessionKey string) (*WechatUserInfo, error) {
	// Base64解码
	cipherText, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	key, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session key: %w", err)
	}

	ivBytes, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, fmt.Errorf("failed to decode iv: %w", err)
	}

	// AES-128-CBC解密
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(cipherText)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, ivBytes)
	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)

	// 去除PKCS7填充
	plainText = removePKCS7Padding(plainText)

	// 解析JSON
	var userInfo WechatUserInfo
	if err := json.Unmarshal(plainText, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &userInfo, nil
}

// validateWatermark 验证水印
func (c *WechatClient) validateWatermark(watermark *struct {
	AppID     string `json:"appid"`
	Timestamp int64  `json:"timestamp"`
}) error {
	if watermark.AppID != c.config.AppID {
		return fmt.Errorf("invalid appid in watermark: expected %s, got %s", c.config.AppID, watermark.AppID)
	}

	// 检查时间戳是否在合理范围内（例如，不超过5分钟）
	now := time.Now().Unix()
	if now-watermark.Timestamp > 300 { // 5分钟
		return fmt.Errorf("watermark timestamp is too old: %d", watermark.Timestamp)
	}

	return nil
}

// removePKCS7Padding 移除PKCS7填充
func removePKCS7Padding(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	padding := int(data[len(data)-1])
	if padding > len(data) || padding == 0 {
		return data
	}

	// 验证填充是否正确
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return data
		}
	}

	return data[:len(data)-padding]
}

// GenerateSignature 生成签名（用于验证请求来源）
func GenerateSignature(params map[string]string, secret string) string {
	// 参数排序并拼接
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}

	// 简单排序
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, params[k]))
	}

	str := strings.Join(pairs, "&") + "&key=" + secret

	// SHA1签名
	h := sha1.New()
	h.Write([]byte(str))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// MockWechatClient 模拟微信客户端（用于测试）
// 支持多用户模拟，根据不同的code生成不同的用户信息
type MockWechatClient struct {
	userCounter int
}

// NewMockWechatClient 创建模拟客户端
func NewMockWechatClient() *MockWechatClient {
	return &MockWechatClient{
		userCounter: 0,
	}
}

// GetSession 模拟获取session - 支持多用户
func (m *MockWechatClient) GetSession(code string) (*WechatSession, error) {
	if code == "" {
		return nil, errors.New("code cannot be empty")
	}

	// 模拟不同的code返回不同结果
	if code == "invalid_code" {
		return nil, errors.New("wechat API error: 40029 - invalid code")
	}

	// 根据code生成不同的用户信息（模拟不同用户）
	m.userCounter++

	// 使用code和计数器生成唯一的OpenID
	mockOpenID := fmt.Sprintf("mock_openid_%s_%d", code, m.userCounter)
	mockSessionKey := fmt.Sprintf("mock_session_key_%s_%d", code, m.userCounter)

	return &WechatSession{
		OpenID:     mockOpenID,
		SessionKey: mockSessionKey,
		UnionID:    fmt.Sprintf("mock_union_id_%d", m.userCounter),
		ErrCode:    0,
		ErrMsg:     "",
	}, nil
}

// ValidateUserInfo 模拟验证用户信息 - 支持多用户
func (m *MockWechatClient) ValidateUserInfo(encryptedData, iv, sessionKey string) (*WechatUserInfo, error) {
	// 简单的模拟验证
	if encryptedData == "" || iv == "" || sessionKey == "" {
		return nil, errors.New("invalid parameters")
	}

	// 从sessionKey解析用户编号
	userNum := "1"
	parts := strings.Split(sessionKey, "_")
	if len(parts) >= 4 {
		userNum = parts[3]
	}

	// 生成对应的OpenID
	openID := strings.Replace(sessionKey, "session_key", "openid", 1)

	return &WechatUserInfo{
		OpenID:    openID,
		NickName:  fmt.Sprintf("测试用户%s", userNum),
		Gender:    1,
		City:      "深圳",
		Province:  "广东",
		Country:   "中国",
		AvatarURL: fmt.Sprintf("https://wx.qlogo.cn/mmopen/vi_32/test_%s.jpg", userNum),
		UnionID:   fmt.Sprintf("mock_union_id_%s", userNum),
		Watermark: struct {
			AppID     string `json:"appid"`
			Timestamp int64  `json:"timestamp"`
		}{
			AppID:     "test_app_id",
			Timestamp: time.Now().Unix(),
		},
	}, nil
}
