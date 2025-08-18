package services

import (
	"ai-classroom/pkg/storage"
	"ai-classroom/pkg/tts"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// EnhancedTTSService 增强TTS服务接口
type EnhancedTTSService interface {
	GenerateAudioBatch(texts []string, options *AudioOptions) (*BatchAudioResult, error)
	GetGenerationProgress(taskID string) (*AudioProgress, error)
	PreviewVoice(text string, voiceType string) (*AudioPreview, error)
	ClearCache() error
}

// EnhancedTTSServiceImpl 增强TTS服务实现
type EnhancedTTSServiceImpl struct {
	ttsClient *tts.EnhancedAliyunTTSClient
	storage   storage.StorageService
	db        *gorm.DB

	// 进度追踪
	progressMutex sync.RWMutex
	progressMap   map[string]*AudioProgress

	// 缓存管理
	cacheMap map[string]*AudioCache
	cacheMux sync.RWMutex
}

// AudioOptions 音频生成选项
type AudioOptions struct {
	VoiceType  string  `json:"voice_type"`
	Speed      float32 `json:"speed"`
	Volume     float32 `json:"volume"`
	PitchRate  float32 `json:"pitch_rate"`
	SampleRate int     `json:"sample_rate"`
	Format     string  `json:"format"`
}

// BatchAudioResult 批量音频生成结果
type BatchAudioResult struct {
	TaskID     string        `json:"task_id"`
	Status     string        `json:"status"`
	TotalFiles int           `json:"total_files"`
	AudioFiles []AudioFile   `json:"audio_files"`
	StartTime  time.Time     `json:"start_time"`
	Duration   time.Duration `json:"duration"`
}

// AudioFile 音频文件信息
type AudioFile struct {
	Index    int    `json:"index"`
	Text     string `json:"text"`
	URL      string `json:"url"`
	Duration int    `json:"duration"`
	Size     int64  `json:"size"`
}

// AudioProgress 音频生成进度
type AudioProgress struct {
	TaskID      string    `json:"task_id"`
	Status      string    `json:"status"` // processing, completed, failed
	Total       int       `json:"total"`
	Completed   int       `json:"completed"`
	Progress    float64   `json:"progress"`
	CurrentText string    `json:"current_text"`
	Error       string    `json:"error,omitempty"`
	StartTime   time.Time `json:"start_time"`
	UpdateTime  time.Time `json:"update_time"`
}

// AudioPreview 音频预览结果
type AudioPreview struct {
	URL      string `json:"url"`
	Duration int    `json:"duration"`
	Size     int64  `json:"size"`
}

// AudioCache 音频缓存
type AudioCache struct {
	Text      string    `json:"text"`
	VoiceType string    `json:"voice_type"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewEnhancedTTSService 创建增强TTS服务
func NewEnhancedTTSService(ttsClient *tts.EnhancedAliyunTTSClient, storage storage.StorageService, db *gorm.DB) EnhancedTTSService {
	return &EnhancedTTSServiceImpl{
		ttsClient:   ttsClient,
		storage:     storage,
		db:          db,
		progressMap: make(map[string]*AudioProgress),
		cacheMap:    make(map[string]*AudioCache),
	}
}

// GenerateAudioBatch 批量生成音频
func (s *EnhancedTTSServiceImpl) GenerateAudioBatch(texts []string, options *AudioOptions) (*BatchAudioResult, error) {
	taskID := s.generateTaskID()
	startTime := time.Now()

	// 初始化进度
	progress := &AudioProgress{
		TaskID:     taskID,
		Status:     "processing",
		Total:      len(texts),
		Completed:  0,
		Progress:   0.0,
		StartTime:  startTime,
		UpdateTime: startTime,
	}

	s.progressMutex.Lock()
	s.progressMap[taskID] = progress
	s.progressMutex.Unlock()

	result := &BatchAudioResult{
		TaskID:     taskID,
		Status:     "processing",
		TotalFiles: len(texts),
		AudioFiles: make([]AudioFile, 0, len(texts)),
		StartTime:  startTime,
	}

	// 异步处理音频生成
	go s.processAudioBatch(taskID, texts, options, result)

	return result, nil
}

// processAudioBatch 处理批量音频生成
func (s *EnhancedTTSServiceImpl) processAudioBatch(taskID string, texts []string, options *AudioOptions, result *BatchAudioResult) {
	defer func() {
		if r := recover(); r != nil {
			s.updateProgress(taskID, "failed", -1, fmt.Sprintf("处理异常: %v", r))
		}
	}()

	for i, text := range texts {
		// 更新进度
		s.updateProgress(taskID, "processing", i, text)

		// 检查缓存
		cacheKey := s.getCacheKey(text, options.VoiceType)
		if cached, exists := s.getFromCache(cacheKey); exists {
			audioFile := AudioFile{
				Index:    i,
				Text:     text,
				URL:      cached.URL,
				Duration: 0, // TODO: 从缓存获取
				Size:     0, // TODO: 从缓存获取
			}
			result.AudioFiles = append(result.AudioFiles, audioFile)
			continue
		}

		// 生成音频
		audioData, err := s.generateSingleAudio(text, options)
		if err != nil {
			s.updateProgress(taskID, "failed", i, fmt.Sprintf("生成音频失败: %v", err))
			return
		}

		// 保存音频文件
		// 确保使用MP3格式
		fileExt := "mp3"
		filename := fmt.Sprintf("audio_%s_%d.%s", taskID, i, fileExt)
		audioURL, err := s.storage.UploadAudio(filename, audioData)
		if err != nil {
			s.updateProgress(taskID, "failed", i, fmt.Sprintf("保存音频失败: %v", err))
			return
		}

		// 添加到缓存
		s.addToCache(cacheKey, &AudioCache{
			Text:      text,
			VoiceType: options.VoiceType,
			URL:       audioURL,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour), // 24小时过期
		})

		audioFile := AudioFile{
			Index:    i,
			Text:     text,
			URL:      audioURL,
			Duration: s.estimateDuration(text),
			Size:     int64(len(audioData)),
		}
		result.AudioFiles = append(result.AudioFiles, audioFile)
	}

	// 更新最终状态
	result.Status = "completed"
	result.Duration = time.Since(result.StartTime)
	s.updateProgress(taskID, "completed", len(texts), "")
}

// generateSingleAudio 生成单个音频
func (s *EnhancedTTSServiceImpl) generateSingleAudio(text string, options *AudioOptions) ([]byte, error) {
	// 使用 TTS 客户端生成音频
	voiceType := tts.VoiceType{
		Name:       options.VoiceType,
		Language:   "zh-CN",
		Gender:     "female",
		SampleRate: 16000,
	}

	response, err := s.ttsClient.GenerateAudio(text, voiceType)
	if err != nil {
		return nil, fmt.Errorf("TTS生成失败: %w", err)
	}

	return response.AudioData, nil
}

// GetGenerationProgress 获取生成进度
func (s *EnhancedTTSServiceImpl) GetGenerationProgress(taskID string) (*AudioProgress, error) {
	s.progressMutex.RLock()
	defer s.progressMutex.RUnlock()

	progress, exists := s.progressMap[taskID]
	if !exists {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	return progress, nil
}

// PreviewVoice 预览语音
func (s *EnhancedTTSServiceImpl) PreviewVoice(text string, voiceType string) (*AudioPreview, error) {
	// 检查缓存
	cacheKey := s.getCacheKey(text, voiceType)
	if cached, exists := s.getFromCache(cacheKey); exists {
		return &AudioPreview{
			URL:      cached.URL,
			Duration: s.estimateDuration(text),
			Size:     0, // TODO: 从缓存获取
		}, nil
	}

	// 生成预览音频
	options := &AudioOptions{
		VoiceType:  voiceType,
		Speed:      1.0,
		Volume:     1.0,
		PitchRate:  1.0,
		SampleRate: 16000,
		Format:     "mp3",
	}

	audioData, err := s.generateSingleAudio(text, options)
	if err != nil {
		return nil, fmt.Errorf("生成预览音频失败: %v", err)
	}

	// 保存预览文件
	// 确保使用MP3格式
	fileExt := "mp3"
	filename := fmt.Sprintf("preview_%s_%d.%s", voiceType, time.Now().Unix(), fileExt)
	audioURL, err := s.storage.UploadAudio(filename, audioData)
	if err != nil {
		return nil, fmt.Errorf("保存预览音频失败: %v", err)
	}

	// 添加到缓存
	s.addToCache(cacheKey, &AudioCache{
		Text:      text,
		VoiceType: voiceType,
		URL:       audioURL,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour), // 预览缓存1小时
	})

	return &AudioPreview{
		URL:      audioURL,
		Duration: s.estimateDuration(text),
		Size:     int64(len(audioData)),
	}, nil
}

// ClearCache 清除缓存
func (s *EnhancedTTSServiceImpl) ClearCache() error {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()

	s.cacheMap = make(map[string]*AudioCache)
	return nil
}

// 辅助方法

// generateTaskID 生成任务ID
func (s *EnhancedTTSServiceImpl) generateTaskID() string {
	return fmt.Sprintf("tts_%d", time.Now().UnixNano())
}

// updateProgress 更新进度
func (s *EnhancedTTSServiceImpl) updateProgress(taskID, status string, completed int, currentText string) {
	s.progressMutex.Lock()
	defer s.progressMutex.Unlock()

	progress, exists := s.progressMap[taskID]
	if !exists {
		return
	}

	progress.Status = status
	if completed >= 0 {
		progress.Completed = completed
		if progress.Total > 0 {
			progress.Progress = float64(completed) / float64(progress.Total) * 100
		}
	}
	if currentText != "" {
		if status == "failed" {
			progress.Error = currentText
		} else {
			progress.CurrentText = currentText
		}
	}
	progress.UpdateTime = time.Now()
}

// getCacheKey 获取缓存键
func (s *EnhancedTTSServiceImpl) getCacheKey(text, voiceType string) string {
	return fmt.Sprintf("%s_%s", voiceType, text)
}

// getFromCache 从缓存获取
func (s *EnhancedTTSServiceImpl) getFromCache(key string) (*AudioCache, bool) {
	s.cacheMux.RLock()
	defer s.cacheMux.RUnlock()

	cached, exists := s.cacheMap[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(cached.ExpiresAt) {
		delete(s.cacheMap, key)
		return nil, false
	}

	return cached, true
}

// addToCache 添加到缓存
func (s *EnhancedTTSServiceImpl) addToCache(key string, cache *AudioCache) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()

	s.cacheMap[key] = cache
}

// estimateDuration 估算音频时长
func (s *EnhancedTTSServiceImpl) estimateDuration(text string) int {
	// 简单估算：中文约每分钟300字，英文约每分钟150词
	textLength := len([]rune(text))
	if textLength == 0 {
		return 1 // 默认1秒
	}

	// 假设平均语速为每分钟200字
	estimatedMinutes := float64(textLength) / 200.0
	estimatedSeconds := int(estimatedMinutes * 60)

	// 最少1秒，最多600秒（10分钟）
	if estimatedSeconds < 1 {
		estimatedSeconds = 1
	} else if estimatedSeconds > 600 {
		estimatedSeconds = 600
	}

	return estimatedSeconds
}
