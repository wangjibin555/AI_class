package tts

// TTSClient TTS客户端通用接口
type TTSClient interface {
	// GenerateAudio 生成音频
	GenerateAudio(text string, voiceType VoiceType) (*TTSResponse, error)

	// GetAvailableVoices 获取可用音色
	GetAvailableVoices() []VoiceType

	// CalculateDuration 计算音频时长
	CalculateDuration(text string, voiceType VoiceType) int
}

// 确保所有TTS客户端都实现了接口
var _ TTSClient = (*AliyunTTSClient)(nil)
var _ TTSClient = (*EnhancedAliyunTTSClient)(nil)
