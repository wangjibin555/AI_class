package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ai-classroom/internal/coze"
	"ai-classroom/internal/models"

	"gorm.io/gorm"
)

// ExerciseGenerationService 练习生成服务
type ExerciseGenerationService struct {
	db                     *gorm.DB
	exerciseWorkflowClient *coze.ExerciseWorkflowClient
}

// NewExerciseGenerationService 创建练习生成服务实例
func NewExerciseGenerationService(db *gorm.DB, exerciseWorkflowClient *coze.ExerciseWorkflowClient) *ExerciseGenerationService {
	return &ExerciseGenerationService{
		db:                     db,
		exerciseWorkflowClient: exerciseWorkflowClient,
	}
}

// GenerateExerciseForCourse 为课程生成练习
func (s *ExerciseGenerationService) GenerateExerciseForCourse(ctx context.Context, userID, courseID uint, req models.GenerationRequest) (*models.GenerationResponse, error) {
	// 1. 验证课程和用户权限
	course, err := s.validateCourseAccess(userID, courseID)
	if err != nil {
		return nil, err
	}

	// 2. 检查是否已存在AI生成的练习
	existingQuiz, err := s.checkExistingAIQuiz(courseID, userID)
	if err != nil {
		return nil, err
	}
	if existingQuiz != nil {
		// 检查参数是否匹配
		if s.quizParametersMatch(existingQuiz, req) {
			// 参数匹配，返回现有练习
			return &models.GenerationResponse{
				QuizID:         existingQuiz.ID,
				TotalQuestions: existingQuiz.TotalQuestions,
				Status:         "already_exists",
			}, nil
		} else {
			// 参数不匹配，删除旧练习并重新生成
			log.Printf("📝 [练习参数变更] 删除旧练习并重新生成，课程ID: %d, 旧参数: 题数=%d 难度=%s, 新参数: 题数=%d 难度=%s",
				courseID, existingQuiz.QuestionCount, existingQuiz.Difficulty, req.QuestionCount, req.Difficulty)

			if err := s.deleteExistingQuiz(existingQuiz.ID); err != nil {
				return nil, fmt.Errorf("删除旧练习失败: %v", err)
			}
		}
	}

	// 3. 创建生成记录
	record, err := s.createGenerationRecord(userID, courseID, course.SourceURL, req)
	if err != nil {
		return nil, err
	}

	// 4. 调用工作流生成练习
	result, err := s.generateWithWorkflow(ctx, record, course.SourceURL, req)
	if err != nil {
		// 更新记录为失败状态
		s.updateGenerationRecord(record.ID, models.GenerationStatusFailed, err.Error(), nil)
		return nil, err
	}

	return result, nil
}

// validateCourseAccess 验证课程访问权限
func (s *ExerciseGenerationService) validateCourseAccess(userID, courseID uint) (*models.Course, error) {
	var course models.Course
	err := s.db.Where("id = ? AND user_id = ?", courseID, userID).First(&course).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("课程不存在或无权访问")
		}
		return nil, fmt.Errorf("查询课程失败: %v", err)
	}

	if course.SourceURL == "" {
		return nil, fmt.Errorf("课程缺少源URL，无法生成练习")
	}

	return &course, nil
}

// CheckExistingAIQuiz 检查是否已存在AI生成的练习（公开方法）
func (s *ExerciseGenerationService) CheckExistingAIQuiz(courseID, userID uint) (*models.Quiz, error) {
	// 首先检查是否有已完成的练习
	quiz, err := s.checkExistingAIQuiz(courseID, userID)
	if err != nil {
		return nil, err
	}
	if quiz != nil {
		return quiz, nil
	}

	// 如果没有找到练习，检查是否有正在生成中的记录
	var record models.ExerciseGenerationRecord
	err = s.db.Where("course_id = ? AND user_id = ? AND generation_status IN (?, ?)",
		courseID, userID, models.GenerationStatusPending, models.GenerationStatusProcessing).
		Order("created_at DESC").First(&record).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 既没有现有练习，也没有生成中的记录
		}
		return nil, fmt.Errorf("查询生成记录失败: %v", err)
	}

	// 如果有正在生成中的记录，返回一个特殊的Quiz对象表示正在生成
	return &models.Quiz{
		ID:            0, // 特殊ID表示正在生成
		CourseID:      courseID,
		UserID:        userID,
		QuestionCount: record.TotalQuestions,
		Title:         "正在生成中...",
		Status:        "generating", // 特殊状态
	}, nil
}

// checkExistingAIQuiz 检查是否已存在AI生成的练习
func (s *ExerciseGenerationService) checkExistingAIQuiz(courseID, userID uint) (*models.Quiz, error) {
	var quiz models.Quiz
	err := s.db.Where("course_id = ? AND user_id = ? AND generation_source = ?",
		courseID, userID, "ai_workflow").First(&quiz).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 不存在
		}
		return nil, fmt.Errorf("查询现有练习失败: %v", err)
	}

	return &quiz, nil
}

// createGenerationRecord 创建生成记录
func (s *ExerciseGenerationService) createGenerationRecord(userID, courseID uint, sourceURL string, req models.GenerationRequest) (*models.ExerciseGenerationRecord, error) {
	requestDataBytes, _ := json.Marshal(req)
	requestDataStr := string(requestDataBytes)

	record := &models.ExerciseGenerationRecord{
		UserID:           userID,
		CourseID:         courseID,
		WorkflowID:       "7533813173200338996", // 练习生成工作流ID
		SourceURL:        sourceURL,
		GenerationStatus: models.GenerationStatusPending,
		RequestData:      &requestDataStr,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("创建生成记录失败: %v", err)
	}

	return record, nil
}

// generateWithWorkflow 使用工作流生成练习
func (s *ExerciseGenerationService) generateWithWorkflow(ctx context.Context, record *models.ExerciseGenerationRecord, sourceURL string, req models.GenerationRequest) (*models.GenerationResponse, error) {
	startTime := time.Now()

	// 更新状态为处理中
	s.updateGenerationRecord(record.ID, models.GenerationStatusProcessing, "", nil)

	// 准备工作流请求
	workflowReq := &coze.ExerciseWorkflowRequest{
		SourceURL:     sourceURL,
		UserID:        fmt.Sprintf("%d", record.UserID),
		CourseID:      fmt.Sprintf("%d", record.CourseID),
		QuestionCount: req.QuestionCount,
		Difficulty:    req.Difficulty,
	}

	// 调用专用的练习生成工作流客户端
	workflowResult, err := s.exerciseWorkflowClient.GenerateExercise(ctx, workflowReq)
	if err != nil {
		return nil, fmt.Errorf("工作流调用失败: %v", err)
	}

	// 计算处理时长
	processingDuration := int(time.Since(startTime).Seconds())

	// 创建测验和题目
	result, err := s.createQuizFromWorkflowResult(record, workflowResult, processingDuration)
	if err != nil {
		return nil, err
	}

	// 更新生成记录为成功状态
	responseDataBytes, _ := json.Marshal(workflowResult)
	responseDataStr := string(responseDataBytes)
	difficultyDistBytes, _ := json.Marshal(workflowResult.DifficultyDistribution)
	difficultyDistStr := string(difficultyDistBytes)

	s.updateGenerationRecord(record.ID, models.GenerationStatusCompleted, "", map[string]interface{}{
		"quiz_id":                 result.QuizID,
		"total_questions":         result.TotalQuestions,
		"processing_duration":     processingDuration,
		"response_data":           &responseDataStr,
		"difficulty_distribution": &difficultyDistStr,
	})

	return result, nil
}

// createQuizFromWorkflowResult 从工作流结果创建测验
func (s *ExerciseGenerationService) createQuizFromWorkflowResult(record *models.ExerciseGenerationRecord, workflowResult *coze.ExerciseWorkflowResult, processingDuration int) (*models.GenerationResponse, error) {
	// 开启事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 解析用户请求的参数
	var userRequest models.GenerationRequest
	if record.RequestData != nil {
		json.Unmarshal([]byte(*record.RequestData), &userRequest)
	}

	// 使用用户选择的参数，而不是AI推断的参数
	questionCount := workflowResult.TotalQuestions
	if userRequest.QuestionCount > 0 {
		questionCount = userRequest.QuestionCount
	}

	difficulty := s.determineDifficultyFromDist(workflowResult.DifficultyDistribution)
	if userRequest.Difficulty != "" {
		difficulty = models.Difficulty(userRequest.Difficulty)
	}

	// 创建Quiz记录
	quiz := &models.Quiz{
		CourseID:         record.CourseID,
		UserID:           record.UserID,
		Title:            workflowResult.CourseName + " - 智能练习",
		Description:      "基于课程内容智能生成的练习题",
		TotalQuestions:   questionCount,
		QuestionCount:    questionCount,
		Difficulty:       difficulty,
		Status:           "active",
		GenerationSource: "ai_workflow",
		WorkflowID:       &record.WorkflowID,
		SourceURL:        &record.SourceURL,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 设置生成参数
	generationParams := map[string]interface{}{
		"workflow_id":             record.WorkflowID,
		"total_questions":         workflowResult.TotalQuestions,
		"difficulty_distribution": workflowResult.DifficultyDistribution,
		"processing_time":         workflowResult.ProcessingTime,
	}
	quiz.SetGenerationInfo(record.WorkflowID, record.SourceURL, generationParams)

	if err := tx.Create(quiz).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建测验失败: %v", err)
	}

	// 批量创建Question记录
	questions := make([]*models.Question, 0, len(workflowResult.Questions))
	for i, q := range workflowResult.Questions {
		// 将选项转换为字符串数组
		var options []string
		for key, value := range q.Options {
			options = append(options, fmt.Sprintf("%s: %v", key, value))
		}

		question := &models.Question{
			QuizID:           quiz.ID,
			CourseID:         record.CourseID,
			Type:             "single_choice",
			Question:         q.Question,
			QuestionNumber:   q.ID,
			Options:          options,
			CorrectAnswer:    q.CorrectAnswer,
			Difficulty:       s.mapDifficultyToDatabase(q.Difficulty),
			Points:           1,
			OrderNum:         i + 1,
			SourceQuestionID: &q.ID,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		questions = append(questions, question)
	}

	if err := tx.CreateInBatches(questions, 10).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建题目失败: %v", err)
	}

	// 更新生成记录的quiz_id
	if err := tx.Model(record).Update("quiz_id", quiz.ID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新生成记录失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("保存数据失败: %v", err)
	}

	return &models.GenerationResponse{
		QuizID:         quiz.ID,
		TotalQuestions: len(questions),
		GenerationID:   record.ID,
		Status:         "completed",
	}, nil
}

// determineDifficultyFromDist 根据难度分布确定整体难度
func (s *ExerciseGenerationService) determineDifficultyFromDist(distribution coze.DifficultyDist) models.Difficulty {
	if distribution.Hard > distribution.Easy && distribution.Hard > distribution.Medium {
		return models.DifficultyHard
	}
	if distribution.Medium > distribution.Easy {
		return models.DifficultyNormal
	}
	return models.DifficultyEasy
}

// mapDifficultyToDatabase 将Coze返回的难度值映射到数据库枚举值
func (s *ExerciseGenerationService) mapDifficultyToDatabase(cozeDifficulty string) models.Difficulty {
	switch cozeDifficulty {
	case "easy":
		return models.DifficultyEasy
	case "medium":
		return models.DifficultyNormal // 将 medium 映射为 normal
	case "hard":
		return models.DifficultyHard
	default:
		// 默认返回 normal 难度
		return models.DifficultyNormal
	}
}

// updateGenerationRecord 更新生成记录
func (s *ExerciseGenerationService) updateGenerationRecord(recordID uint, status, errorMsg string, additionalData map[string]interface{}) error {
	updates := map[string]interface{}{
		"generation_status": status,
		"updated_at":        time.Now(),
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	if additionalData != nil {
		for key, value := range additionalData {
			updates[key] = value
		}
	}

	return s.db.Model(&models.ExerciseGenerationRecord{}).Where("id = ?", recordID).Updates(updates).Error
}

// GetGenerationRecord 获取生成记录
func (s *ExerciseGenerationService) GetGenerationRecord(recordID uint, userID uint) (*models.ExerciseGenerationRecord, error) {
	var record models.ExerciseGenerationRecord
	err := s.db.Where("id = ? AND user_id = ?", recordID, userID).
		Preload("User").
		Preload("Course").
		Preload("Quiz").
		First(&record).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("生成记录不存在")
		}
		return nil, fmt.Errorf("查询生成记录失败: %v", err)
	}

	return &record, nil
}

// GetUserGenerationRecords 获取用户的生成记录列表
func (s *ExerciseGenerationService) GetUserGenerationRecords(userID uint, page, pageSize int) ([]*models.ExerciseGenerationRecord, error) {
	var records []*models.ExerciseGenerationRecord

	offset := (page - 1) * pageSize
	err := s.db.Where("user_id = ?", userID).
		Preload("Course").
		Preload("Quiz").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&records).Error

	if err != nil {
		return nil, fmt.Errorf("查询生成记录失败: %v", err)
	}

	return records, nil
}

// quizParametersMatch 检查练习参数是否匹配
func (s *ExerciseGenerationService) quizParametersMatch(existingQuiz *models.Quiz, req models.GenerationRequest) bool {
	// 检查题目数量
	if existingQuiz.QuestionCount != req.QuestionCount {
		return false
	}

	// 检查难度等级
	if string(existingQuiz.Difficulty) != req.Difficulty {
		return false
	}

	return true
}

// deleteExistingQuiz 删除现有练习及相关数据
func (s *ExerciseGenerationService) deleteExistingQuiz(quizID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除用户答案记录
		if err := tx.Where("quiz_id IN (SELECT id FROM quiz_attempts WHERE quiz_id = ?)", quizID).Delete(&models.UserAnswer{}).Error; err != nil {
			return fmt.Errorf("删除用户答案失败: %v", err)
		}

		// 2. 删除答题记录
		if err := tx.Where("quiz_id = ?", quizID).Delete(&models.QuizAttempt{}).Error; err != nil {
			return fmt.Errorf("删除答题记录失败: %v", err)
		}

		// 3. 删除题目
		if err := tx.Where("quiz_id = ?", quizID).Delete(&models.Question{}).Error; err != nil {
			return fmt.Errorf("删除题目失败: %v", err)
		}

		// 4. 删除练习
		if err := tx.Delete(&models.Quiz{}, quizID).Error; err != nil {
			return fmt.Errorf("删除练习失败: %v", err)
		}

		// 5. 更新相关的生成记录（将quiz_id设为NULL）
		if err := tx.Model(&models.ExerciseGenerationRecord{}).Where("quiz_id = ?", quizID).Update("quiz_id", nil).Error; err != nil {
			return fmt.Errorf("更新生成记录失败: %v", err)
		}

		log.Printf("✅ [练习删除] 练习及相关数据删除成功，练习ID: %d", quizID)
		return nil
	})
}
