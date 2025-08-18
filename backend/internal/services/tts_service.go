package services

import (
	"ai-classroom/internal/models"
	"ai-classroom/pkg/tts"
	"fmt"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

// TTSService TTS服务接口
type TTSService interface {
	GenerateSlideAudio(slide *models.Slide, voiceType string) error
	GenerateCourseAudio(course *models.Course) error
	GetAvailableVoices() []tts.VoiceType
	CalculateDuration(text string, voiceType string) int
}

// ttsService TTS服务实现
type ttsService struct {
	ttsClient tts.TTSClient
	storage   StorageService
	db        *gorm.DB
}

// NewTTSService 创建TTS服务实例
func NewTTSService(ttsClient tts.TTSClient, storage StorageService, db *gorm.DB) TTSService {
	return &ttsService{
		ttsClient: ttsClient,
		storage:   storage,
		db:        db,
	}
}

// GenerateSlideAudio 为幻灯片生成音频
func (s *ttsService) GenerateSlideAudio(slide *models.Slide, voiceType string) error {
	// 获取语音类型配置
	voiceConfig := s.getVoiceConfig(voiceType)

	// 准备要合成的文本（标题 + 内容 + 演讲备注）
	text := slide.Title + "。" + slide.Content
	if slide.SpeakerNotes != "" {
		text += "。" + slide.SpeakerNotes
	}

	// 生成音频
	response, err := s.ttsClient.GenerateAudio(text, voiceConfig)
	if err != nil {
		return fmt.Errorf("生成音频失败: %w", err)
	}

	// 生成文件名
	// 生产环境固定使用MP3格式，不管TTS返回什么格式
	fileExt := "mp3"
	fmt.Printf("🎵 TTS返回格式: %s, 强制使用MP3扩展名", response.Format)
	filename := fmt.Sprintf("slide_%d_%d.%s", slide.CourseID, slide.SlideNumber, fileExt)
	filepath := filepath.Join("audio", "slides", filename)

	// 保存音频文件
	audioURL, err := s.storage.SaveAudio(filepath, response.AudioData)
	if err != nil {
		return fmt.Errorf("保存音频文件失败: %w", err)
	}

	// 更新幻灯片信息
	slide.AudioURL = audioURL
	slide.Duration = response.Duration / 1000 // 转换为秒

	return nil
}

// GenerateCourseAudio 为整个课程生成音频
func (s *ttsService) GenerateCourseAudio(course *models.Course) error {
	// 获取课程的所有幻灯片
	slides, err := s.getCourseSlides(course.ID)
	if err != nil {
		return fmt.Errorf("获取课程幻灯片失败: %w", err)
	}

	// 为每张幻灯片生成音频
	for _, slide := range slides {
		if err := s.GenerateSlideAudio(slide, course.VoiceType); err != nil {
			return fmt.Errorf("为幻灯片 %d 生成音频失败: %w", slide.SlideNumber, err)
		}

		// 更新幻灯片信息
		if err := s.updateSlide(slide); err != nil {
			return fmt.Errorf("更新幻灯片失败: %w", err)
		}

		// 添加延迟避免API限制
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// GetAvailableVoices 获取可用的语音类型
func (s *ttsService) GetAvailableVoices() []tts.VoiceType {
	return s.ttsClient.GetAvailableVoices()
}

// CalculateDuration 计算文本的预估朗读时长
func (s *ttsService) CalculateDuration(text string, voiceType string) int {
	voiceConfig := s.getVoiceConfig(voiceType)
	return s.ttsClient.CalculateDuration(text, voiceConfig)
}

// getVoiceConfig 获取语音类型配置
func (s *ttsService) getVoiceConfig(voiceType string) tts.VoiceType {
	// 默认语音配置
	defaultVoice := tts.VoiceType{
		Name:       "zhixiaobai",
		Language:   "zh-CN",
		Gender:     "female",
		SampleRate: 16000,
	}

	// 根据语音类型设置配置
	switch voiceType {
	case "zhixiaobai":
		defaultVoice.Name = "zhixiaobai"
		defaultVoice.Gender = "female"
	case "zhimao":
		defaultVoice.Name = "zhimao"
		defaultVoice.Gender = "female"
	case "xiaoyun":
		defaultVoice.Name = "xiaoyun"
		defaultVoice.Gender = "female"
	case "xiaogang":
		defaultVoice.Name = "xiaogang"
		defaultVoice.Gender = "male"
	case "xiaomei":
		defaultVoice.Name = "xiaomei"
		defaultVoice.Gender = "female"
	case "xiaofeng":
		defaultVoice.Name = "xiaofeng"
		defaultVoice.Gender = "male"
	case "xiaoyan":
		defaultVoice.Name = "xiaoyan"
		defaultVoice.Gender = "female"
	case "xiaoyu":
		defaultVoice.Name = "xiaoyu"
		defaultVoice.Gender = "male"
	}

	return defaultVoice
}

// getCourseSlides 获取课程的所有幻灯片
func (s *ttsService) getCourseSlides(courseID uint) ([]*models.Slide, error) {
	var slides []*models.Slide
	if err := s.db.Where("course_id = ?", courseID).Order("slide_number ASC").Find(&slides).Error; err != nil {
		return nil, fmt.Errorf("获取课程幻灯片失败: %w", err)
	}
	return slides, nil
}

// updateSlide 更新幻灯片信息
func (s *ttsService) updateSlide(slide *models.Slide) error {
	if err := s.db.Save(slide).Error; err != nil {
		return fmt.Errorf("更新幻灯片失败: %w", err)
	}
	return nil
}

// StorageService 存储服务接口
type StorageService interface {
	SaveAudio(filepath string, data []byte) (string, error)
}
