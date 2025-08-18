package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-classroom/internal/models"
	"ai-classroom/internal/repositories"

	"math"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// convertProgressToPercentage 智能转换进度数据为百分比(0-100)
// 处理数据库中可能存在的两种格式：
// 1. 0-10000格式（旧数据）：需要除以100
// 2. 0-1格式（新数据）：需要乘以100
func convertProgressToPercentage(value float64) float64 {
	if value > 100 {
		// 认为是0-10000格式，转换为百分比
		return math.Min(math.Max(value/100, 0), 100)
	} else {
		// 认为是0-1格式，转换为百分比
		return math.Min(math.Max(value*100, 0), 100)
	}
}

// LearningRecordResponse 学习记录响应结构
type LearningRecordResponse struct {
	ID            uint                  `json:"id"`
	UserID        uint                  `json:"user_id"`
	CourseID      uint                  `json:"course_id"`
	CurrentSlide  int                   `json:"current_slide"`
	TotalSlides   int                   `json:"total_slides"`
	Progress      float64               `json:"progress"`
	StudyDuration int                   `json:"study_duration"`
	CompleteRate  float64               `json:"complete_rate"`
	LastPosition  int                   `json:"last_position"`
	QuizBestScore *float64              `json:"quiz_best_score"`
	QuizAttempts  int                   `json:"quiz_attempts"`
	IsCompleted   bool                  `json:"is_completed"`
	IsBookmarked  bool                  `json:"is_bookmarked"`
	Status        models.LearningStatus `json:"status"`
	StartTime     *time.Time            `json:"start_time"`
	EndTime       *time.Time            `json:"end_time"`
	LastStudyTime *time.Time            `json:"last_study_time"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`

	// 课程信息
	CourseTitle    string `json:"course_title"`
	CourseCategory string `json:"course_category"`
}

// LearningHandler 学习记录处理器
type LearningHandler struct {
	db         *gorm.DB
	courseRepo repositories.CourseRepository
}

// NewLearningHandler 创建学习记录处理器
func NewLearningHandler(db *gorm.DB, courseRepo repositories.CourseRepository) *LearningHandler {
	return &LearningHandler{
		db:         db,
		courseRepo: courseRepo,
	}
}

// StartLearningRequest 开始学习请求
type StartLearningRequest struct {
	CourseID     uint   `json:"course_id" binding:"required"`
	LearningType string `json:"learning_type"`
	StartTime    int64  `json:"start_time"`
}

// UpdateProgressRequest 更新进度请求
type UpdateProgressRequest struct {
	LearningRecordID uint    `json:"learning_record_id" binding:"required"`
	CurrentSlide     int     `json:"current_slide"`
	StudyDuration    int     `json:"study_duration"`
	CompleteRate     float64 `json:"complete_rate"`
	SlideProgress    string  `json:"slide_progress"`
}

// StartLearning 开始学习记录
func (h *LearningHandler) StartLearning(c *gin.Context) {
	var req StartLearningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 验证课程存在
	course, err := h.courseRepo.GetByID(req.CourseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "课程不存在",
		})
		return
	}

	// 检查是否已有学习记录
	var existingRecord models.LearningRecord
	result := h.db.Where("user_id = ? AND course_id = ?", userID, req.CourseID).First(&existingRecord)

	if result.Error == nil {
		// 更新现有记录
		now := time.Now()
		existingRecord.StartTime = &now
		existingRecord.Status = models.LearningStatusLearning
		existingRecord.UpdatedAt = time.Now()

		if err := h.db.Save(&existingRecord).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新学习记录失败",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"learning_record_id": existingRecord.ID,
				"course_id":          course.ID,
				"status":             "updated",
			},
			"message": "学习记录更新成功",
		})
		return
	}

	// 创建新的学习记录
	now := time.Now()
	learningRecord := models.LearningRecord{
		UserID:       userID.(uint),
		CourseID:     req.CourseID,
		StartTime:    &now,
		Status:       models.LearningStatusLearning,
		CurrentSlide: 0,
		TotalSlides:  course.SlidesCount,
	}

	if err := h.db.Create(&learningRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建学习记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"learning_record_id": learningRecord.ID,
			"course_id":          course.ID,
			"status":             "created",
		},
		"message": "学习记录创建成功",
	})
}

// UpdateProgress 更新学习进度
func (h *LearningHandler) UpdateProgress(c *gin.Context) {
	var req UpdateProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 获取学习记录
	var learningRecord models.LearningRecord
	if err := h.db.Where("id = ? AND user_id = ?", req.LearningRecordID, userID).First(&learningRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "学习记录不存在",
		})
		return
	}

	// 更新进度
	learningRecord.CurrentSlide = req.CurrentSlide
	learningRecord.StudyDuration = req.StudyDuration
	learningRecord.CompleteRate = req.CompleteRate // 前端已经转换为0-1小数，直接存储
	learningRecord.UpdatedAt = time.Now()

	// 如果完成率达到1.0（100%），标记为已完成
	if req.CompleteRate >= 1.0 {
		learningRecord.Status = models.LearningStatusCompleted
		learningRecord.EndTime = &learningRecord.UpdatedAt
	}

	if err := h.db.Save(&learningRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新学习进度失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "学习进度更新成功",
		"data": gin.H{
			"learning_record_id": learningRecord.ID,
			"current_slide":      learningRecord.CurrentSlide,
			"complete_rate":      learningRecord.CompleteRate,
			"status":             learningRecord.Status,
		},
	})
}

// GetLearningRecord 获取学习记录
func (h *LearningHandler) GetLearningRecord(c *gin.Context) {
	recordIDStr := c.Param("recordId")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的记录ID",
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 获取学习记录
	var learningRecord models.LearningRecord
	if err := h.db.Where("id = ? AND user_id = ?", recordID, userID).First(&learningRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "学习记录不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": learningRecord,
	})
}

// GetCourseLearningRecord 获取课程学习记录
func (h *LearningHandler) GetCourseLearningRecord(c *gin.Context) {
	courseIDStr := c.Param("courseId")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的课程ID",
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 获取学习记录
	var learningRecord models.LearningRecord
	if err := h.db.Where("course_id = ? AND user_id = ?", courseID, userID).First(&learningRecord).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "学习记录不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "查询学习记录失败",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": learningRecord,
	})
}

// GetLearningRecords 获取学习记录列表
func (h *LearningHandler) GetLearningRecords(c *gin.Context) {
	fmt.Println("DEBUG: GetLearningRecords 方法被调用")

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")

	offset := (page - 1) * pageSize

	// 构建查询
	query := h.db.Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	var total int64
	query.Model(&models.LearningRecord{}).Count(&total)

	// 获取记录
	var records []models.LearningRecord
	fmt.Printf("DEBUG: 正在查询学习记录，用户ID: %v\n", userID)
	if err := query.Offset(offset).Limit(pageSize).Order("updated_at DESC").Find(&records).Error; err != nil {
		fmt.Printf("DEBUG: 查询学习记录失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询学习记录失败",
		})
		return
	}

	fmt.Printf("DEBUG: 查询到 %d 条学习记录\n", len(records))

	// 手动查询课程信息
	var courseIDs []uint
	for _, record := range records {
		courseIDs = append(courseIDs, record.CourseID)
	}

	fmt.Printf("DEBUG: 需要查询的课程ID: %v\n", courseIDs)

	var courses []models.Course
	if len(courseIDs) > 0 {
		if err := h.db.Where("id IN ?", courseIDs).Find(&courses).Error; err != nil {
			fmt.Printf("DEBUG: 查询课程信息失败: %v\n", err)
		} else {
			fmt.Printf("DEBUG: 查询到 %d 门课程\n", len(courses))
		}
	}

	// 创建课程ID到课程信息的映射
	courseMap := make(map[uint]models.Course)
	for _, course := range courses {
		courseMap[course.ID] = course
	}

	// 构建响应数据
	var responseRecords []LearningRecordResponse
	for _, record := range records {
		responseRecord := LearningRecordResponse{
			ID:            record.ID,
			UserID:        record.UserID,
			CourseID:      record.CourseID,
			CurrentSlide:  record.CurrentSlide,
			TotalSlides:   record.TotalSlides,
			Progress:      convertProgressToPercentage(record.Progress),
			StudyDuration: record.StudyDuration,
			CompleteRate:  convertProgressToPercentage(record.CompleteRate),
			LastPosition:  record.LastPosition,
			QuizBestScore: record.QuizBestScore,
			QuizAttempts:  record.QuizAttempts,
			IsCompleted:   record.IsCompleted,
			IsBookmarked:  record.IsBookmarked,
			Status:        record.Status,
			StartTime:     record.StartTime,
			EndTime:       record.EndTime,
			LastStudyTime: record.LastStudyTime,
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
		}

		fmt.Printf("DEBUG: 处理进度修复 - 原始进度: %.1f, 修复后进度: %.1f\n", record.Progress, responseRecord.Progress)

		// 添加课程信息
		if course, exists := courseMap[record.CourseID]; exists {
			responseRecord.CourseTitle = course.Title
			responseRecord.CourseCategory = string(course.Category)
		}

		responseRecords = append(responseRecords, responseRecord)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"records":   responseRecords,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// EndLearning 结束学习
func (h *LearningHandler) EndLearning(c *gin.Context) {
	var req struct {
		LearningRecordID uint  `json:"learning_record_id" binding:"required"`
		EndTime          int64 `json:"end_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 获取学习记录
	var learningRecord models.LearningRecord
	if err := h.db.Where("id = ? AND user_id = ?", req.LearningRecordID, userID).First(&learningRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "学习记录不存在",
		})
		return
	}

	// 更新结束时间
	endTime := time.Now()
	learningRecord.EndTime = &endTime
	learningRecord.Status = models.LearningStatusCompleted
	learningRecord.UpdatedAt = time.Now()

	if err := h.db.Save(&learningRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "结束学习记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "学习记录已结束",
		"data": gin.H{
			"learning_record_id": learningRecord.ID,
			"status":             learningRecord.Status,
			"end_time":           learningRecord.EndTime,
		},
	})
}

// DeleteLearningRecord 删除学习记录
func (h *LearningHandler) DeleteLearningRecord(c *gin.Context) {
	recordIDStr := c.Param("recordId")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的记录ID",
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 删除学习记录
	result := h.db.Where("id = ? AND user_id = ?", recordID, userID).Delete(&models.LearningRecord{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除学习记录失败",
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "学习记录不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "学习记录删除成功",
	})
}

// GetLearningStats 获取学习统计
func (h *LearningHandler) GetLearningStats(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 统计用户学习数据
	var stats struct {
		TotalCourses     int64   `json:"total_courses"`
		CompletedCourses int64   `json:"completed_courses"`
		TotalStudyTime   int64   `json:"total_study_time"`
		AverageProgress  float64 `json:"average_progress"`
	}

	// 查询总课程数
	h.db.Model(&models.LearningRecord{}).Where("user_id = ?", userID).Count(&stats.TotalCourses)

	// 查询已完成课程数
	h.db.Model(&models.LearningRecord{}).Where("user_id = ? AND status = ?", userID, models.LearningStatusCompleted).Count(&stats.CompletedCourses)

	// 查询总学习时间
	h.db.Model(&models.LearningRecord{}).Where("user_id = ?", userID).Select("SUM(study_duration)").Scan(&stats.TotalStudyTime)

	// 计算平均进度
	var avgProgress float64
	h.db.Model(&models.LearningRecord{}).Where("user_id = ?", userID).Select("AVG(complete_rate)").Scan(&avgProgress)
	stats.AverageProgress = convertProgressToPercentage(avgProgress) // 智能转换为百分比

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"data":    stats,
		"message": "获取学习统计成功",
	})
}
