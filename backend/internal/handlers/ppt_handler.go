package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"ai-classroom/internal/models"
	"ai-classroom/internal/repositories"
	"ai-classroom/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// PPTHandler PPT处理器
type PPTHandler struct {
	db         *gorm.DB
	courseRepo repositories.CourseRepository
	storage    storage.StorageService
}

// NewPPTHandler 创建PPT处理器
func NewPPTHandler(db *gorm.DB, courseRepo repositories.CourseRepository, storage storage.StorageService) *PPTHandler {
	return &PPTHandler{
		db:         db,
		courseRepo: courseRepo,
		storage:    storage,
	}
}

// PreviewPPT 预览PPT
func (h *PPTHandler) PreviewPPT(c *gin.Context) {
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

	// 获取课程信息
	course, err := h.courseRepo.GetByID(uint(courseID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "课程不存在",
		})
		return
	}

	// 检查权限
	if course.UserID != userID.(uint) && !course.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "无权限访问此课程",
		})
		return
	}

	// 检查PPT文件是否存在
	if course.PPTFilePath == "" {
		// 尝试生成PPT文件
		pptURL, err := h.generatePPTFile(course)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "PPT文件不存在，生成失败: " + err.Error(),
			})
			return
		}

		// 更新课程信息
		course.PPTFilePath = pptURL
		h.courseRepo.Update(course)
	}

	// 构建PPT预览URL
	previewURL := h.buildPPTPreviewURL(course.PPTFilePath)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"course_id":    course.ID,
			"title":        course.Title,
			"ppt_url":      previewURL,
			"file_path":    course.PPTFilePath,
			"slides_count": course.SlidesCount,
		},
		"message": "PPT预览地址获取成功",
	})
}

// DownloadPPT 下载PPT
func (h *PPTHandler) DownloadPPT(c *gin.Context) {
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

	// 获取课程信息
	course, err := h.courseRepo.GetByID(uint(courseID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "课程不存在",
		})
		return
	}

	// 检查权限
	if course.UserID != userID.(uint) && !course.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "无权限访问此课程",
		})
		return
	}

	// 检查PPT文件是否存在
	if course.PPTFilePath == "" {
		// 尝试生成PPT文件
		pptURL, err := h.generatePPTFile(course)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "PPT文件不存在，生成失败: " + err.Error(),
			})
			return
		}

		// 更新课程信息
		course.PPTFilePath = pptURL
		h.courseRepo.Update(course)
	}

	// 构建下载URL
	downloadURL := h.buildPPTDownloadURL(course.PPTFilePath)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"course_id":    course.ID,
			"title":        course.Title,
			"download_url": downloadURL,
			"file_path":    course.PPTFilePath,
			"file_size":    h.getPPTFileSize(course.PPTFilePath),
		},
		"message": "PPT下载地址获取成功",
	})
}

// GeneratePPTFile 生成PPT文件
func (h *PPTHandler) GeneratePPTFile(c *gin.Context) {
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

	// 获取课程信息
	course, err := h.courseRepo.GetByID(uint(courseID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "课程不存在",
		})
		return
	}

	// 检查权限
	if course.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "无权限操作此课程",
		})
		return
	}

	// 异步生成PPT文件
	go func() {
		pptURL, err := h.generatePPTFile(course)
		if err != nil {
			fmt.Printf("生成PPT文件失败: %v\n", err)
			return
		}

		// 更新课程信息
		course.PPTFilePath = pptURL
		h.courseRepo.Update(course)
		fmt.Printf("PPT文件生成成功: %s\n", pptURL)
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "PPT文件生成任务已开始",
		"data": gin.H{
			"course_id": course.ID,
			"status":    "generating",
		},
	})
}

// GetPPTStatus 获取PPT生成状态
func (h *PPTHandler) GetPPTStatus(c *gin.Context) {
	courseIDStr := c.Param("courseId")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的课程ID",
		})
		return
	}

	// 获取课程信息
	course, err := h.courseRepo.GetByID(uint(courseID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "课程不存在",
		})
		return
	}

	status := "not_generated"
	if course.PPTFilePath != "" {
		if h.checkPPTFileExists(course.PPTFilePath) {
			status = "completed"
		} else {
			status = "failed"
		}
	} else if course.Status == models.CourseStatusGenerating {
		status = "generating"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"course_id":    course.ID,
			"status":       status,
			"ppt_path":     course.PPTFilePath,
			"slides_count": course.SlidesCount,
		},
	})
}

// 辅助方法

// generatePPTFile 生成PPT文件
func (h *PPTHandler) generatePPTFile(course *models.Course) (string, error) {
	// 这里可以调用PPT生成服务
	// 目前简化实现，生成HTML格式的PPT

	// 获取课程的幻灯片
	var slides []models.Slide
	if err := h.db.Where("course_id = ?", course.ID).Order("slide_number ASC").Find(&slides).Error; err != nil {
		return "", fmt.Errorf("获取幻灯片失败: %w", err)
	}

	if len(slides) == 0 {
		return "", fmt.Errorf("课程没有幻灯片数据")
	}

	// 生成HTML PPT内容
	htmlContent := h.generateHTMLPPT(course, slides)

	// 保存PPT文件
	filename := fmt.Sprintf("ppt_%d_%d.html", course.ID, course.UpdatedAt.Unix())

	// 保存到存储服务
	pptURL, err := h.storage.UploadFile(filename, []byte(htmlContent))
	if err != nil {
		return "", fmt.Errorf("保存PPT文件失败: %w", err)
	}

	return pptURL, nil
}

// generateHTMLPPT 生成HTML格式的PPT
func (h *PPTHandler) generateHTMLPPT(course *models.Course, slides []models.Slide) string {
	var slidesHTML strings.Builder

	for i, slide := range slides {
		slidesHTML.WriteString(fmt.Sprintf(`
		<div class="slide" id="slide_%d">
			<div class="slide-header">
				<h1>%s</h1>
				<div class="slide-number">%d / %d</div>
			</div>
			<div class="slide-content">
				<div class="slide-body">%s</div>
				%s
			</div>
		</div>
		`, i+1, slide.Title, slide.SlideNumber, len(slides), h.formatSlideContent(slide.Content), h.formatSpeakerNotes(slide.SpeakerNotes)))
	}

	htmlTemplate := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body { font-family: 'Microsoft YaHei', Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
        .presentation { max-width: 1200px; margin: 0 auto; background: white; box-shadow: 0 0 20px rgba(0,0,0,0.1); }
        .slide { min-height: 80vh; padding: 40px; border-bottom: 1px solid #eee; display: flex; flex-direction: column; }
        .slide-header { margin-bottom: 30px; }
        .slide-header h1 { color: #2c3e50; margin: 0; font-size: 2.5em; font-weight: bold; }
        .slide-number { color: #7f8c8d; font-size: 0.9em; margin-top: 10px; }
        .slide-content { flex: 1; display: flex; flex-direction: column; }
        .slide-body { flex: 1; font-size: 1.2em; line-height: 1.8; color: #34495e; }
        .slide-body ul, .slide-body ol { padding-left: 30px; }
        .slide-body li { margin-bottom: 10px; }
        .speaker-notes { margin-top: 30px; padding: 20px; background: #ecf0f1; border-radius: 8px; font-style: italic; color: #7f8c8d; }
        .speaker-notes:before { content: "💡 演讲者备注："; font-weight: bold; color: #3498db; }
        @media print { .slide { page-break-after: always; } }
        @media (max-width: 768px) { 
            .slide { padding: 20px; min-height: 70vh; }
            .slide-header h1 { font-size: 2em; }
            .slide-body { font-size: 1.1em; }
        }
    </style>
</head>
<body>
    <div class="presentation">
        <div class="slide" id="title-slide">
            <div class="slide-header">
                <h1>%s</h1>
                <div class="slide-number">封面</div>
            </div>
            <div class="slide-content">
                <div class="slide-body">
                    <p><strong>课程描述：</strong>%s</p>
                    <p><strong>幻灯片数量：</strong>%d 张</p>
                    <p><strong>生成时间：</strong>%s</p>
                </div>
            </div>
        </div>
        %s
    </div>
</body>
</html>`, course.Title, course.Title, course.Description, len(slides), course.CreatedAt.Format("2006-01-02 15:04:05"), slidesHTML.String())

	return htmlTemplate
}

// formatSlideContent 格式化幻灯片内容
func (h *PPTHandler) formatSlideContent(content string) string {
	// 简单的格式化：将换行转换为段落
	lines := strings.Split(content, "\n")
	var formatted strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			// 列表项
			formatted.WriteString(fmt.Sprintf("<li>%s</li>", line[2:]))
		} else {
			// 普通段落
			formatted.WriteString(fmt.Sprintf("<p>%s</p>", line))
		}
	}

	result := formatted.String()
	if strings.Contains(result, "<li>") {
		result = "<ul>" + result + "</ul>"
	}

	return result
}

// formatSpeakerNotes 格式化演讲者备注
func (h *PPTHandler) formatSpeakerNotes(notes string) string {
	if notes == "" {
		return ""
	}
	return fmt.Sprintf(`<div class="speaker-notes">%s</div>`, notes)
}

// buildPPTPreviewURL 构建PPT预览URL
func (h *PPTHandler) buildPPTPreviewURL(filePath string) string {
	// 从配置中获取文件服务器地址
	baseURL := viper.GetString("server.file_base_url")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%s",
			viper.GetString("server.external_host"),
			viper.GetString("server.external_port"))
	}
	return fmt.Sprintf("%s/static/%s", baseURL, filePath)
}

// buildPPTDownloadURL 构建PPT下载URL
func (h *PPTHandler) buildPPTDownloadURL(filePath string) string {
	// 从配置中获取文件服务器地址
	baseURL := viper.GetString("server.file_base_url")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%s",
			viper.GetString("server.external_host"),
			viper.GetString("server.external_port"))
	}
	return fmt.Sprintf("%s/api/v1/files/download?file=%s", baseURL, filePath)
}

// checkPPTFileExists 检查PPT文件是否存在
func (h *PPTHandler) checkPPTFileExists(filePath string) bool {
	// 这里应该检查文件是否在存储服务中存在
	// 简化实现，假设文件存在
	return filePath != ""
}

// getPPTFileSize 获取PPT文件大小
func (h *PPTHandler) getPPTFileSize(filePath string) int64 {
	// 这里应该获取实际文件大小
	// 简化实现，返回估算大小
	return 1024 * 1024 // 1MB
}
