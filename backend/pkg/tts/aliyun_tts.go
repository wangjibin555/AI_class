package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AliyunTTSClient 阿里云TTS客户端
type AliyunTTSClient struct {
	accessKeyID     string
	accessKeySecret string
	appKey          string
	baseURL         string
	timeout         time.Duration
	client          *http.Client
	token           string    // 缓存的访问token
	tokenExpireTime time.Time // token过期时间
}

// VoiceType 语音类型
type VoiceType struct {
	Name       string `json:"name"`        // 语音名称
	Language   string `json:"language"`    // 语言
	Gender     string `json:"gender"`      // 性别
	SampleRate int    `json:"sample_rate"` // 采样率
}

// TTSRequest TTS请求
type TTSRequest struct {
	Text       string    `json:"text"`        // 要合成的文本
	VoiceType  VoiceType `json:"voice_type"`  // 语音类型
	Format     string    `json:"format"`      // 音频格式
	SampleRate int       `json:"sample_rate"` // 采样率
	Volume     int       `json:"volume"`      // 音量
	Speed      int       `json:"speed"`       // 语速
	Pitch      int       `json:"pitch"`       // 音调
}

// TTSResponse TTS响应
type TTSResponse struct {
	AudioData  []byte `json:"audio_data"`  // 音频数据
	Duration   int    `json:"duration"`    // 音频时长（毫秒）
	Format     string `json:"format"`      // 音频格式
	SampleRate int    `json:"sample_rate"` // 采样率
}

// NewAliyunTTSClient 创建阿里云TTS客户端
func NewAliyunTTSClient(accessKeyID, accessKeySecret, appKey, baseURL string, timeout time.Duration) *AliyunTTSClient {
	return &AliyunTTSClient{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		appKey:          appKey,
		baseURL:         baseURL,
		timeout:         timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// GenerateAudio 生成音频
func (c *AliyunTTSClient) GenerateAudio(text string, voiceType VoiceType) (*TTSResponse, error) {
	// 生产环境：确保配置完整
	if c.accessKeyID == "" || c.accessKeySecret == "" || c.appKey == "" {
		return nil, fmt.Errorf("阿里云TTS配置不完整，请检查access_key_id、access_key_secret和app_key配置")
	}

	// 获取有效token
	token, err := c.getValidToken()
	if err != nil {
		return nil, fmt.Errorf("获取阿里云TTS Token失败: %w", err)
	}

	// 按照阿里云TTS API文档构建请求
	bodyContent := map[string]interface{}{
		"appkey":      c.appKey,
		"text":        text,
		"format":      "mp3",
		"sample_rate": 16000,
		"voice":       voiceType.Name,
		"volume":      50,
		"speech_rate": 0,
		"pitch_rate":  0,
	}

	// 如果有token，添加到请求体中
	if token != "" {
		bodyContent["token"] = token
	}

	return c.sendAliyunRequest(bodyContent)
}

// GetAvailableVoices 获取可用的语音类型
func (c *AliyunTTSClient) GetAvailableVoices() []VoiceType {
	return []VoiceType{
		{
			Name:       "zhimao",
			Language:   "zh-CN",
			Gender:     "female",
			SampleRate: 16000,
		},
		{
			Name:       "xiaoyun",
			Language:   "zh-CN",
			Gender:     "female",
			SampleRate: 16000,
		},
		{
			Name:       "xiaogang",
			Language:   "zh-CN",
			Gender:     "male",
			SampleRate: 16000,
		},
		{
			Name:       "xiaomei",
			Language:   "zh-CN",
			Gender:     "female",
			SampleRate: 16000,
		},
		{
			Name:       "xiaofeng",
			Language:   "zh-CN",
			Gender:     "male",
			SampleRate: 16000,
		},
		{
			Name:       "xiaoyan",
			Language:   "zh-CN",
			Gender:     "female",
			SampleRate: 16000,
		},
		{
			Name:       "xiaoyu",
			Language:   "zh-CN",
			Gender:     "male",
			SampleRate: 16000,
		},
	}
}

// sendRequest 发送TTS请求
func (c *AliyunTTSClient) sendRequest(request TTSRequest) (*TTSResponse, error) {
	// 根据阿里云TTS API文档构建请求参数
	bodyContent := map[string]interface{}{
		"appkey":      c.appKey,
		"text":        request.Text,
		"format":      request.Format,
		"sample_rate": request.SampleRate,
		"voice":       request.VoiceType.Name,
		"volume":      request.Volume,
		"speech_rate": request.Speed,
		"pitch_rate":  request.Pitch,
	}

	// 序列化请求
	requestBody, err := json.Marshal(bodyContent)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", c.baseURL+"/stream/v1/tts", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头 - 根据阿里云文档使用正确的认证方式
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	// 阿里云TTS需要获取token，这里简化为直接使用AccessKey（需要根据实际API调整）
	req.Header.Set("Authorization", "Bearer "+c.accessKeyID)
	req.Header.Set("X-NLS-Token", c.accessKeyID)

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码和Content-Type
	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP错误 %d: %s", resp.StatusCode, string(body))
	}

	// 根据Content-Type处理响应
	if contentType == "audio/mpeg" || contentType == "audio/wav" || contentType == "audio/pcm" {
		// 成功返回音频数据
		return &TTSResponse{
			AudioData:  body,
			Duration:   len(request.Text) * 200, // 估算时长
			Format:     request.Format,
			SampleRate: request.SampleRate,
		}, nil
	} else {
		// 错误响应，尝试解析JSON错误信息
		return nil, fmt.Errorf("TTS服务返回错误: %s", string(body))
	}
}

// generateMockAudio 生成模拟音频
func (c *AliyunTTSClient) generateMockAudio(text string, voiceType VoiceType) (*TTSResponse, error) {
	// 模拟音频生成
	// 在实际应用中，这里应该调用真实的TTS服务
	// 现在返回一个模拟的音频数据（1秒的静音）

	// 计算模拟的音频时长（基于文本长度）
	duration := len(text) * 200 // 每个字符大约200毫秒

	// 生成模拟的音频数据（这里只是占位符）
	audioData := make([]byte, duration*16) // 16kHz采样率，1字节/样本

	return &TTSResponse{
		AudioData:  audioData,
		Duration:   duration,
		Format:     "mp3",
		SampleRate: 16000,
	}, nil
}

// CalculateDuration 计算文本的预估朗读时长
func (c *AliyunTTSClient) CalculateDuration(text string, voiceType VoiceType) int {
	// 基于文本长度和语音类型计算预估时长
	baseDuration := len(text) * 200 // 每个字符200毫秒

	// 根据语音类型调整
	switch voiceType.Name {
	case "xiaoyun":
		baseDuration = int(float64(baseDuration) * 1.0) // 标准速度
	case "xiaogang":
		baseDuration = int(float64(baseDuration) * 0.9) // 稍快
	case "xiaomei":
		baseDuration = int(float64(baseDuration) * 1.1) // 稍慢
	case "xiaofeng":
		baseDuration = int(float64(baseDuration) * 0.95) // 稍快
	default:
		baseDuration = int(float64(baseDuration) * 1.0)
	}

	return baseDuration
}

// getValidToken 获取有效的访问token
func (c *AliyunTTSClient) getValidToken() (string, error) {
	// 检查缓存的token是否有效
	if c.token != "" && time.Now().Before(c.tokenExpireTime) {
		return c.token, nil
	}

	// 根据阿里云文档，获取Token
	// 这里需要通过阿里云的Token服务获取，但为了简化，我们使用另一种方式
	// 暂时返回空字符串，让请求通过其他方式认证
	token, err := c.requestToken()
	if err != nil {
		return "", fmt.Errorf("获取Token失败: %w", err)
	}

	c.token = token
	c.tokenExpireTime = time.Now().Add(24 * time.Hour) // 24小时后过期

	return c.token, nil
}

// requestToken 请求获取Token
func (c *AliyunTTSClient) requestToken() (string, error) {
	// 根据阿里云TTS文档，需要通过Token服务获取
	// 暂时使用一个测试实现，这在某些配置下可能工作

	// 创建获取token的请求
	tokenURL := "https://nls-meta.cn-shanghai.aliyuncs.com/pop/2018-05-18/tokens"

	bodyContent := map[string]interface{}{
		"AccessKeyId":     c.accessKeyID,
		"AccessKeySecret": c.accessKeySecret,
	}

	requestBody, err := json.Marshal(bodyContent)
	if err != nil {
		return "", fmt.Errorf("序列化token请求失败: %v", err)
	}

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("创建token请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送token请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取token响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token请求失败 %d: %s", resp.StatusCode, string(body))
	}

	// 解析token响应
	var tokenResp map[string]interface{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("解析token响应失败: %v", err)
	}

	if token, ok := tokenResp["Token"].(string); ok {
		return token, nil
	}

	return "", fmt.Errorf("token响应格式错误: %s", string(body))
}

// sendAliyunRequest 发送符合阿里云TTS API规范的请求
func (c *AliyunTTSClient) sendAliyunRequest(bodyContent map[string]interface{}) (*TTSResponse, error) {
	// 序列化请求体
	requestBody, err := json.Marshal(bodyContent)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求 - 使用阿里云TTS的标准URL
	url := "https://nls-gateway-cn-shanghai.aliyuncs.com/stream/v1/tts"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	// 根据阿里云TTS文档，如果没有在请求体中设置token，则在Header中设置
	if _, hasToken := bodyContent["token"]; !hasToken {
		req.Header.Set("X-NLS-Token", c.appKey)
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码和Content-Type
	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP错误 %d: %s", resp.StatusCode, string(body))
	}

	// 根据Content-Type处理响应
	if contentType == "audio/mpeg" || contentType == "audio/wav" || contentType == "audio/pcm" {
		// 检测实际的音频格式
		actualFormat := "mp3" // 默认为MP3
		if len(body) >= 12 && string(body[0:4]) == "RIFF" && string(body[8:12]) == "WAVE" {
			actualFormat = "wav"
		} else if len(body) >= 3 && string(body[0:3]) == "ID3" {
			actualFormat = "mp3"
		} else if len(body) >= 2 && body[0] == 0xFF && (body[1]&0xE0) == 0xE0 {
			actualFormat = "mp3"
		}

		// 成功返回音频数据
		return &TTSResponse{
			AudioData:  body,
			Duration:   len(bodyContent["text"].(string)) * 200, // 估算时长
			Format:     actualFormat,                            // 使用检测到的实际格式
			SampleRate: bodyContent["sample_rate"].(int),
		}, nil
	} else {
		// 返回错误信息
		return nil, fmt.Errorf("TTS服务返回错误: %s", string(body))
	}
}
