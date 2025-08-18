package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"ai-classroom/internal/models"
	"ai-classroom/internal/repositories"
	"ai-classroom/pkg/storage"
	"ai-classroom/pkg/tts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AudioHandler 音频生成处理器
type AudioHandler struct {
	ttsClient  tts.TTSClient
	storage    storage.StorageService
	db         *gorm.DB
	courseRepo repositories.CourseRepository
	userRepo   repositories.UserRepository

	// 音频生成状态管理
	progressMutex sync.RWMutex
	progressMap   map[uint]*AudioGenerationProgress
}

// AudioGenerationProgress 音频生成进度
type AudioGenerationProgress struct {
	CourseID        uint       `json:"course_id"`
	Status          string     `json:"status"` // processing, completed, failed
	TotalSlides     int        `json:"total_slides"`
	CompletedSlides int        `json:"completed_slides"`
	Progress        float64    `json:"progress"` // 0-100
	CurrentSlide    string     `json:"current_slide"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	StartTime       time.Time  `json:"start_time"`
	EstimatedEnd    time.Time  `json:"estimated_end"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

// GenerateAudioRequest 生成音频请求
type GenerateAudioRequest struct {
	CourseID   uint   `json:"course_id"`
	VoiceType  string `json:"voice_type,omitempty"`
	Speed      int    `json:"speed,omitempty"`
	Volume     int    `json:"volume,omitempty"`
	Regenerate bool   `json:"regenerate,omitempty"` // 是否重新生成已有音频
}

// AudioStatusResponse 音频状态响应
type AudioStatusResponse struct {
	CourseID uint                     `json:"course_id"`
	Status   string                   `json:"status"`
	Progress *AudioGenerationProgress `json:"progress,omitempty"`
	Slides   []SlideAudioInfo         `json:"slides,omitempty"`
	Message  string                   `json:"message"`
}

// SlideAudioInfo 幻灯片音频信息
type SlideAudioInfo struct {
	SlideID  uint   `json:"slide_id"`
	Title    string `json:"title"`
	AudioURL string `json:"audio_url"`
	Duration int    `json:"duration"`
	Status   string `json:"status"` // pending, generating, completed, failed
}

// NewAudioHandler 创建音频处理器
func NewAudioHandler(
	ttsClient tts.TTSClient,
	storage storage.StorageService,
	db *gorm.DB,
	courseRepo repositories.CourseRepository,
	userRepo repositories.UserRepository,
) *AudioHandler {
	return &AudioHandler{
		ttsClient:   ttsClient,
		storage:     storage,
		db:          db,
		courseRepo:  courseRepo,
		userRepo:    userRepo,
		progressMap: make(map[uint]*AudioGenerationProgress),
	}
}

// GenerateCourseAudio 生成课程音频
func (h *AudioHandler) GenerateCourseAudio(c *gin.Context) {
	// 获取用户ID（从JWT中间件）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 获取课程ID
	courseIDStr := c.Param("courseId")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的课程ID",
		})
		return
	}

	// 验证课程权限
	course, err := h.courseRepo.GetByID(uint(courseID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "课程不存在",
		})
		return
	}

	if course.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "无权限访问此课程",
		})
		return
	}

	// 解析请求参数
	var request GenerateAudioRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		// 如果没有请求体，使用默认参数
		request = GenerateAudioRequest{
			CourseID:   uint(courseID),
			VoiceType:  "zhixiaobai",
			Speed:      0,
			Volume:     70,
			Regenerate: false,
		}
	}
	request.CourseID = uint(courseID)

	// 检查是否已在生成中
	h.progressMutex.RLock()
	if progress, exists := h.progressMap[uint(courseID)]; exists && progress.Status == "processing" {
		h.progressMutex.RUnlock()
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "音频生成已在进行中",
			"data":    progress,
		})
		return
	}
	h.progressMutex.RUnlock()

	// 异步生成音频
	go h.generateAudioAsync(uint(courseID), request)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "音频生成已开始",
		"data": gin.H{
			"course_id": courseID,
			"status":    "processing",
		},
	})
}

// GetAudioStatus 获取音频生成状态
func (h *AudioHandler) GetAudioStatus(c *gin.Context) {
	// 获取课程ID
	courseIDStr := c.Param("courseId")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的课程ID",
		})
		return
	}

	// 获取进度信息
	h.progressMutex.RLock()
	progress, exists := h.progressMap[uint(courseID)]
	h.progressMutex.RUnlock()

	// 获取幻灯片音频信息
	slides, err := h.getSlideAudioInfo(uint(courseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取音频信息失败",
			"error":   err.Error(),
		})
		return
	}

	response := AudioStatusResponse{
		CourseID: uint(courseID),
		Slides:   slides,
	}

	if exists {
		response.Status = progress.Status
		response.Progress = progress
		response.Message = fmt.Sprintf("音频生成进度: %.1f%%", progress.Progress)
	} else {
		// 检查是否已有完整音频
		hasCompleteAudio := true
		for _, slide := range slides {
			if slide.AudioURL == "" {
				hasCompleteAudio = false
				break
			}
		}

		if hasCompleteAudio {
			response.Status = "completed"
			response.Message = "所有音频已生成完成"
		} else {
			response.Status = "pending"
			response.Message = "尚未开始音频生成"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取状态成功",
		"data":    response,
	})
}

// generateAudioAsync 异步生成音频
func (h *AudioHandler) generateAudioAsync(courseID uint, request GenerateAudioRequest) {
	ctx := context.Background()

	// 初始化进度
	progress := &AudioGenerationProgress{
		CourseID:        courseID,
		Status:          "processing",
		TotalSlides:     0,
		CompletedSlides: 0,
		Progress:        0,
		StartTime:       time.Now(),
	}

	h.progressMutex.Lock()
	h.progressMap[courseID] = progress
	h.progressMutex.Unlock()

	// 获取幻灯片列表
	var slides []models.Slide
	if err := h.db.Where("course_id = ?", courseID).Order("slide_number ASC").Find(&slides).Error; err != nil {
		h.updateProgress(courseID, func(p *AudioGenerationProgress) {
			p.Status = "failed"
			p.ErrorMessage = "获取幻灯片列表失败: " + err.Error()
		})
		return
	}

	if len(slides) == 0 {
		h.updateProgress(courseID, func(p *AudioGenerationProgress) {
			p.Status = "failed"
			p.ErrorMessage = "课程没有幻灯片"
		})
		return
	}

	// 更新总数和预估时间
	h.updateProgress(courseID, func(p *AudioGenerationProgress) {
		p.TotalSlides = len(slides)
		p.EstimatedEnd = time.Now().Add(time.Duration(len(slides)*30) * time.Second)
	})

	// 生成音频
	successCount := 0
	for i, slide := range slides {
		// 检查是否需要生成音频
		if slide.AudioURL != "" && !request.Regenerate {
			successCount++
			h.updateProgress(courseID, func(p *AudioGenerationProgress) {
				p.CompletedSlides = i + 1
				p.Progress = float64(p.CompletedSlides) / float64(p.TotalSlides) * 100
				p.CurrentSlide = slide.Title
			})
			continue
		}

		// 生成单张幻灯片音频
		err := h.generateSlideAudio(ctx, &slide, request)
		if err != nil {
			fmt.Printf("生成幻灯片 %d 音频失败: %v\n", slide.ID, err)
		} else {
			successCount++
		}

		// 更新进度
		h.updateProgress(courseID, func(p *AudioGenerationProgress) {
			p.CompletedSlides = i + 1
			p.Progress = float64(p.CompletedSlides) / float64(p.TotalSlides) * 100
			p.CurrentSlide = slide.Title
		})

		// 频率限制（每秒1个请求）
		time.Sleep(time.Second)
	}

	// 完成生成
	now := time.Now()
	h.updateProgress(courseID, func(p *AudioGenerationProgress) {
		if successCount == len(slides) {
			p.Status = "completed"
		} else {
			p.Status = "partial_success"
			p.ErrorMessage = fmt.Sprintf("成功生成 %d/%d 个音频", successCount, len(slides))
		}
		p.CompletedAt = &now
	})

	// 记录操作日志
	h.logAudioGeneration(courseID, len(slides), successCount)
}

// generateSlideAudio 生成单张幻灯片音频
func (h *AudioHandler) generateSlideAudio(ctx context.Context, slide *models.Slide, request GenerateAudioRequest) error {
	// 1. 预处理文本
	fullText := h.combineSlideContent(slide)
	if fullText == "" {
		return fmt.Errorf("幻灯片内容为空")
	}

	// 2. 选择音色
	voices := h.ttsClient.GetAvailableVoices()
	voiceType := voices[0] // 默认使用第一个音色
	if request.VoiceType != "" {
		for _, voice := range voices {
			if voice.Name == request.VoiceType {
				voiceType = voice
				break
			}
		}
	}

	// 添加详细日志
	fmt.Printf("🎤 音频生成音色选择:")
	fmt.Printf("  请求的音色: %s", request.VoiceType)
	fmt.Printf("  选择的音色: %s", voiceType.Name)
	fmt.Printf("  可用音色列表: %v", func() []string {
		var names []string
		for _, v := range voices {
			names = append(names, v.Name)
		}
		return names
	}())

	// 3. 生成音频
	ttsResponse, err := h.ttsClient.GenerateAudio(fullText, voiceType)
	if err != nil {
		return fmt.Errorf("TTS生成失败: %w", err)
	}

	// 4. 保存音频文件（生产环境固定使用MP3格式）
	fileExt := "mp3" // 生产环境固定使用MP3格式，不管实际音频数据是什么格式
	fmt.Printf("🎵 保存音频文件为MP3格式，TTS返回格式: %s, 数据大小: %d bytes", ttsResponse.Format, len(ttsResponse.AudioData))
	fileName := fmt.Sprintf("slide_%d_%d.%s", slide.CourseID, slide.SlideNumber, fileExt)
	filePath := fmt.Sprintf("audio/slides/%s", fileName)
	audioURL, err := h.storage.SaveAudio(filePath, ttsResponse.AudioData)
	if err != nil {
		return fmt.Errorf("保存音频文件失败: %w", err)
	}

	// 5. 更新数据库
	err = h.db.Model(slide).Updates(map[string]interface{}{
		"audio_url": audioURL,
		"duration":  ttsResponse.Duration,
	}).Error
	if err != nil {
		return fmt.Errorf("更新数据库失败: %w", err)
	}

	return nil
}

// combineSlideContent 组合幻灯片内容
func (h *AudioHandler) combineSlideContent(slide *models.Slide) string {
	var parts []string

	if slide.Title != "" {
		parts = append(parts, slide.Title)
	}

	if slide.Content != "" {
		parts = append(parts, slide.Content)
	}

	if slide.SpeakerNotes != "" {
		parts = append(parts, slide.SpeakerNotes)
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("%s。", parts[0]) + fmt.Sprintf("具体内容如下：%s", parts[1])
}

// getSlideAudioInfo 获取幻灯片音频信息
func (h *AudioHandler) getSlideAudioInfo(courseID uint) ([]SlideAudioInfo, error) {
	var slides []models.Slide
	err := h.db.Where("course_id = ?", courseID).Order("slide_number ASC").Find(&slides).Error
	if err != nil {
		return nil, err
	}

	var slideInfos []SlideAudioInfo
	for _, slide := range slides {
		status := "pending"
		if slide.AudioURL != "" {
			status = "completed"
		}

		slideInfos = append(slideInfos, SlideAudioInfo{
			SlideID:  slide.ID,
			Title:    slide.Title,
			AudioURL: slide.AudioURL,
			Duration: slide.Duration,
			Status:   status,
		})
	}

	return slideInfos, nil
}

// updateProgress 更新进度
func (h *AudioHandler) updateProgress(courseID uint, updateFunc func(*AudioGenerationProgress)) {
	h.progressMutex.Lock()
	defer h.progressMutex.Unlock()

	if progress, exists := h.progressMap[courseID]; exists {
		updateFunc(progress)
	}
}

// logAudioGeneration 记录音频生成日志
func (h *AudioHandler) logAudioGeneration(courseID uint, totalSlides, successCount int) {
	logData := map[string]interface{}{
		"course_id":     courseID,
		"total_slides":  totalSlides,
		"success_count": successCount,
		"error_count":   totalSlides - successCount,
	}

	operationLog := &models.OperationLog{
		OperationType: "batch_audio_generation",
		OperationDesc: fmt.Sprintf("课程 %d 批量音频生成完成", courseID),
		ResourceType:  "course",
		ResourceID:    &courseID,
		RequestData:   models.RequestData(logData),
		Status:        "success",
		CreatedAt:     time.Now(),
	}

	if successCount < totalSlides {
		operationLog.Status = "partial_success"
		operationLog.ErrorMessage = fmt.Sprintf("%d张幻灯片生成失败", totalSlides-successCount)
	}

	h.db.Create(operationLog)
}

// PreviewVoice 语音预览
func (h *AudioHandler) PreviewVoice(c *gin.Context) {
	var request struct {
		Text      string `json:"text" binding:"required"`
		VoiceType string `json:"voice_type"`
		Speed     int    `json:"speed"`
		Volume    int    `json:"volume"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 选择音色
	voices := h.ttsClient.GetAvailableVoices()
	voiceType := voices[0]
	if request.VoiceType != "" {
		for _, voice := range voices {
			if voice.Name == request.VoiceType {
				voiceType = voice
				break
			}
		}
	}

	// 生成预览音频
	ttsResponse, err := h.ttsClient.GenerateAudio(request.Text, voiceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "语音生成失败",
			"error":   err.Error(),
		})
		return
	}

	// 保存临时音频文件
	fileName := fmt.Sprintf("preview_%d.mp3", time.Now().Unix())
	audioURL, err := h.storage.UploadAudio(fileName, ttsResponse.AudioData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "保存音频失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "预览音频生成成功",
		"data": gin.H{
			"audio_url": audioURL,
			"duration":  ttsResponse.Duration,
		},
	})
}
