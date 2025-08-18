package services

import (
	"ai-classroom/internal/models"
	"ai-classroom/pkg/ai"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// QuizService 练习服务接口
type QuizService interface {
	GenerateQuiz(courseID uint, userID uint, req GenerateQuizRequest) (*models.Quiz, error)
	GetQuizByID(quizID uint, userID uint) (*models.Quiz, error)
	StartQuiz(quizID uint, userID uint) (*models.QuizAttempt, error)
	SubmitAnswer(attemptID uint, userID uint, req SubmitAnswerRequest) (map[string]interface{}, error)
	SubmitQuiz(attemptID uint, userID uint) (*QuizResult, error)
	GetAttemptByID(attemptID uint, userID uint) (*models.QuizAttempt, error)
	GetQuizStats(quizID uint, userID uint) (map[string]interface{}, error)
}

// GenerateQuizRequest 生成练习请求
type GenerateQuizRequest struct {
	QuestionCount int      `json:"question_count" binding:"min=1,max=50"` // 题目数量
	Difficulty    string   `json:"difficulty"`                            // 难度等级
	QuestionTypes []string `json:"question_types"`                        // 题目类型
	TimeLimit     int      `json:"time_limit"`                            // 时间限制（分钟）
}

// SubmitAnswerRequest 提交答案请求
type SubmitAnswerRequest struct {
	QuestionID uint     `json:"question_id" binding:"required"` // 题目ID
	Answer     string   `json:"answer"`                         // 答案（单选/填空）
	Answers    []string `json:"answers"`                        // 答案（多选）
}

// QuizResult 练习结果
type QuizResult struct {
	AttemptID       uint             `json:"attempt_id"`       // 答题记录ID
	Score           int              `json:"score"`            // 得分
	TotalScore      int              `json:"total_score"`      // 总分
	Percentage      float64          `json:"percentage"`       // 正确率
	CorrectCount    int              `json:"correct_count"`    // 正确题数
	TotalCount      int              `json:"total_count"`      // 总题数
	TimeSpent       int              `json:"time_spent"`       // 用时（秒）
	CompletedAt     *time.Time       `json:"completed_at"`     // 完成时间
	QuestionResults []QuestionResult `json:"question_results"` // 每题结果
}

// QuestionResult 题目结果
type QuestionResult struct {
	QuestionID    uint   `json:"question_id"`    // 题目ID
	IsCorrect     bool   `json:"is_correct"`     // 是否正确
	UserAnswer    string `json:"user_answer"`    // 用户答案
	CorrectAnswer string `json:"correct_answer"` // 正确答案
	Explanation   string `json:"explanation"`    // 解释
}

// quizService 练习服务实现
type quizService struct {
	db       *gorm.DB
	aiClient *ai.DashScopeClient
}

// NewQuizService 创建练习服务实例
func NewQuizService(db *gorm.DB, aiClient *ai.DashScopeClient) QuizService {
	return &quizService{
		db:       db,
		aiClient: aiClient,
	}
}

// GenerateQuiz 生成练习
func (s *quizService) GenerateQuiz(courseID uint, userID uint, req GenerateQuizRequest) (*models.Quiz, error) {
	// 获取课程信息
	var course models.Course
	if err := s.db.First(&course, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("课程不存在")
		}
		return nil, fmt.Errorf("获取课程失败: %w", err)
	}

	// 检查权限
	if course.UserID != userID && !course.IsPublic {
		return nil, errors.New("无权限访问该课程")
	}

	// 使用AI生成题目
	questions, err := s.generateQuestionsWithAI(course.SourceContent, req)
	if err != nil {
		return nil, fmt.Errorf("AI生成题目失败: %w", err)
	}

	// 创建练习
	quiz := &models.Quiz{
		CourseID:       courseID,
		UserID:         userID,
		Title:          fmt.Sprintf("%s - 练习", course.Title),
		Description:    fmt.Sprintf("基于课程《%s》生成的练习", course.Title),
		QuestionCount:  len(questions),
		TotalQuestions: len(questions),
		TimeLimit:      req.TimeLimit,
		Difficulty:     models.Difficulty(req.Difficulty),
		Status:         models.QuizStatusActive,
	}

	// 保存练习
	if err := s.db.Create(quiz).Error; err != nil {
		return nil, fmt.Errorf("创建练习失败: %w", err)
	}

	// 保存题目
	for i, question := range questions {
		question.QuizID = quiz.ID
		question.QuestionNumber = i + 1
		question.OrderNum = i + 1
		if err := s.db.Create(&question).Error; err != nil {
			return nil, fmt.Errorf("保存题目失败: %w", err)
		}
	}

	return quiz, nil
}

// GetQuizByID 根据ID获取练习
func (s *quizService) GetQuizByID(quizID uint, userID uint) (*models.Quiz, error) {
	var quiz models.Quiz
	if err := s.db.Preload("Questions").First(&quiz, quizID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("练习不存在")
		}
		return nil, fmt.Errorf("获取练习失败: %w", err)
	}

	// 检查权限
	if quiz.UserID != userID {
		return nil, errors.New("无权限访问该练习")
	}

	return &quiz, nil
}

// StartQuiz 开始练习
func (s *quizService) StartQuiz(quizID uint, userID uint) (*models.QuizAttempt, error) {
	// 检查练习是否存在
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("练习不存在")
		}
		return nil, fmt.Errorf("获取练习失败: %w", err)
	}

	// 检查是否有进行中的答题记录
	var existingAttempt models.QuizAttempt
	if err := s.db.Where("quiz_id = ? AND user_id = ? AND status = ?",
		quizID, userID, models.AttemptStatusInProgress).First(&existingAttempt).Error; err == nil {
		// 返回现有的答题记录
		return &existingAttempt, nil
	}

	// 创建新的答题记录
	now := time.Now()
	attempt := &models.QuizAttempt{
		QuizID:        quizID,
		UserID:        userID,
		CourseID:      quiz.CourseID,
		Status:        models.AttemptStatusInProgress,
		StartTime:     now,
		QuestionCount: quiz.QuestionCount,
	}

	if err := s.db.Create(attempt).Error; err != nil {
		return nil, fmt.Errorf("创建答题记录失败: %w", err)
	}

	return attempt, nil
}

// SubmitAnswer 提交答案
func (s *quizService) SubmitAnswer(attemptID uint, userID uint, req SubmitAnswerRequest) (map[string]interface{}, error) {
	// 获取答题记录
	var attempt models.QuizAttempt
	if err := s.db.First(&attempt, attemptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("答题记录不存在")
		}
		return nil, fmt.Errorf("获取答题记录失败: %w", err)
	}

	// 检查权限
	if attempt.UserID != userID {
		return nil, errors.New("无权限访问该答题记录")
	}

	// 检查答题状态
	if attempt.Status != models.AttemptStatusInProgress {
		return nil, errors.New("答题已结束，无法提交答案")
	}

	// 获取题目信息
	var question models.Question
	if err := s.db.First(&question, req.QuestionID).Error; err != nil {
		return nil, errors.New("题目不存在")
	}

	// 检查答案是否正确
	isCorrect := s.checkAnswer(question, req)

	// 保存用户答案
	userAnswer := &models.UserAnswer{
		UserID:     userID,
		AttemptID:  attemptID,
		QuestionID: req.QuestionID,
		UserAnswer: req.Answer, // 使用UserAnswer字段名
		IsCorrect:  isCorrect,
		AnsweredAt: time.Now(),
	}

	if err := s.db.Create(userAnswer).Error; err != nil {
		return nil, fmt.Errorf("保存答案失败: %w", err)
	}

	return map[string]interface{}{
		"is_correct":     isCorrect,
		"correct_answer": question.CorrectAnswer,
		"explanation":    question.Explanation,
	}, nil
}

// SubmitQuiz 提交练习
func (s *quizService) SubmitQuiz(attemptID uint, userID uint) (*QuizResult, error) {
	// 获取答题记录
	var attempt models.QuizAttempt
	if err := s.db.Preload("UserAnswers").First(&attempt, attemptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("答题记录不存在")
		}
		return nil, fmt.Errorf("获取答题记录失败: %w", err)
	}

	// 检查权限
	if attempt.UserID != userID {
		return nil, errors.New("无权限访问该答题记录")
	}

	// 检查答题状态
	if attempt.Status != models.AttemptStatusInProgress {
		return nil, errors.New("答题已结束")
	}

	// 计算得分
	score, correctCount, questionResults := s.calculateScore(attempt)

	// 更新答题记录
	now := time.Now()
	attempt.Status = models.AttemptStatusCompleted
	attempt.UserScore = score
	attempt.CorrectCount = correctCount
	attempt.EndTime = &now

	if err := s.db.Save(&attempt).Error; err != nil {
		return nil, fmt.Errorf("更新答题记录失败: %w", err)
	}

	// 构建结果
	result := &QuizResult{
		AttemptID:       attemptID,
		Score:           score,
		TotalScore:      attempt.QuestionCount * 10, // 每题10分
		Percentage:      float64(correctCount) / float64(attempt.QuestionCount) * 100,
		CorrectCount:    correctCount,
		TotalCount:      attempt.QuestionCount,
		TimeSpent:       int(now.Sub(attempt.StartTime).Seconds()),
		CompletedAt:     &now,
		QuestionResults: questionResults,
	}

	return result, nil
}

// GetAttemptByID 根据ID获取答题记录
func (s *quizService) GetAttemptByID(attemptID uint, userID uint) (*models.QuizAttempt, error) {
	var attempt models.QuizAttempt
	if err := s.db.Preload("UserAnswers").First(&attempt, attemptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("答题记录不存在")
		}
		return nil, fmt.Errorf("获取答题记录失败: %w", err)
	}

	// 检查权限
	if attempt.UserID != userID {
		return nil, errors.New("无权限访问该答题记录")
	}

	return &attempt, nil
}

// GetQuizStats 获取练习统计
func (s *quizService) GetQuizStats(quizID uint, userID uint) (map[string]interface{}, error) {
	// 检查练习是否存在
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("练习不存在")
		}
		return nil, fmt.Errorf("获取练习失败: %w", err)
	}

	// 检查权限
	if quiz.UserID != userID {
		return nil, errors.New("无权限访问该练习")
	}

	// 获取统计信息
	var stats struct {
		TotalAttempts     int64   `json:"total_attempts"`
		CompletedAttempts int64   `json:"completed_attempts"`
		AverageScore      float64 `json:"average_score"`
		BestScore         int     `json:"best_score"`
		AverageTime       float64 `json:"average_time"`
	}

	// 总答题次数
	s.db.Model(&models.QuizAttempt{}).Where("quiz_id = ?", quizID).Count(&stats.TotalAttempts)

	// 已完成答题次数
	s.db.Model(&models.QuizAttempt{}).Where("quiz_id = ? AND status = ?",
		quizID, models.AttemptStatusCompleted).Count(&stats.CompletedAttempts)

	// 平均分
	s.db.Model(&models.QuizAttempt{}).Where("quiz_id = ? AND status = ?",
		quizID, models.AttemptStatusCompleted).Select("AVG(score)").Scan(&stats.AverageScore)

	// 最高分
	s.db.Model(&models.QuizAttempt{}).Where("quiz_id = ? AND status = ?",
		quizID, models.AttemptStatusCompleted).Select("MAX(score)").Scan(&stats.BestScore)

	// 平均用时
	s.db.Model(&models.QuizAttempt{}).Where("quiz_id = ? AND status = ?",
		quizID, models.AttemptStatusCompleted).Select("AVG(TIMESTAMPDIFF(SECOND, start_time, end_time))").Scan(&stats.AverageTime)

	return map[string]interface{}{
		"quiz_id": quizID,
		"stats":   stats,
	}, nil
}

// generateQuestionsWithAI 使用AI生成题目
func (s *quizService) generateQuestionsWithAI(content string, req GenerateQuizRequest) ([]models.Question, error) {
	// 构建AI提示
	prompt := fmt.Sprintf(`请基于以下内容生成%d道练习题：

内容：
%s

要求：
1. 题目类型：%s
2. 难度等级：%s
3. 每题都要有正确答案和详细解释
4. 严格按照JSON格式输出

输出格式：
{
  "questions": [
    {
      "question_text": "题目内容",
      "question_type": "single_choice/multiple_choice/fill_blank",
      "options": ["选项A", "选项B", "选项C", "选项D"],
      "correct_answer": "正确答案",
      "explanation": "详细解释"
    }
  ]
}`, req.QuestionCount, content, req.QuestionTypes, req.Difficulty)

	// 调用AI生成题目
	response, err := s.aiClient.GenerateContent(prompt)
	if err != nil {
		return nil, err
	}

	// 解析AI响应
	var aiResult struct {
		Questions []struct {
			QuestionText  string   `json:"question_text"`
			QuestionType  string   `json:"question_type"`
			Options       []string `json:"options"`
			CorrectAnswer string   `json:"correct_answer"`
			Explanation   string   `json:"explanation"`
		} `json:"questions"`
	}

	if err := parseJSONResponse(response, &aiResult); err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	// 转换为题目模型
	var questions []models.Question
	for _, q := range aiResult.Questions {
		question := models.Question{
			Question:      q.QuestionText,
			Type:          models.QuestionType(q.QuestionType),
			Options:       q.Options,
			CorrectAnswer: q.CorrectAnswer,
			Explanation:   q.Explanation,
		}
		questions = append(questions, question)
	}

	return questions, nil
}

// checkAnswer 检查答案是否正确
func (s *quizService) checkAnswer(question models.Question, req SubmitAnswerRequest) bool {
	switch question.Type {
	case models.QuestionTypeSingleChoice, models.QuestionTypeFillBlank:
		return req.Answer == question.CorrectAnswer
	case models.QuestionTypeMultipleChoice:
		return s.checkMultipleChoiceAnswer(req.Answer, question.CorrectAnswer)
	default:
		return false
	}
}

// checkMultipleChoiceAnswer 检查多选题答案
func (s *quizService) checkMultipleChoiceAnswer(userAnswer, correctAnswer string) bool {
	if userAnswer == "" || correctAnswer == "" {
		return false
	}

	// 解析用户答案 "A,C" -> ["A", "C"]
	userAnswers := strings.Split(userAnswer, ",")
	for i := range userAnswers {
		userAnswers[i] = strings.TrimSpace(userAnswers[i])
	}

	// 解析正确答案 "A,C" -> ["A", "C"]
	correctAnswers := strings.Split(correctAnswer, ",")
	for i := range correctAnswers {
		correctAnswers[i] = strings.TrimSpace(correctAnswers[i])
	}

	// 排序并比较
	sort.Strings(userAnswers)
	sort.Strings(correctAnswers)

	if len(userAnswers) != len(correctAnswers) {
		return false
	}

	for i := range userAnswers {
		if userAnswers[i] != correctAnswers[i] {
			return false
		}
	}

	return true
}

// calculateScore 计算得分
func (s *quizService) calculateScore(attempt models.QuizAttempt) (int, int, []QuestionResult) {
	score := 0
	correctCount := 0
	var questionResults []QuestionResult

	// 查询用户答案
	var userAnswers []models.UserAnswer
	if err := s.db.Where("attempt_id = ?", attempt.ID).Find(&userAnswers).Error; err != nil {
		return score, correctCount, questionResults
	}

	for _, userAnswer := range userAnswers {
		var question models.Question
		if err := s.db.First(&question, userAnswer.QuestionID).Error; err != nil {
			continue
		}

		isCorrect := userAnswer.IsCorrect
		if isCorrect {
			score += 10 // 每题10分
			correctCount++
		}

		questionResults = append(questionResults, QuestionResult{
			QuestionID:    userAnswer.QuestionID,
			IsCorrect:     isCorrect,
			UserAnswer:    userAnswer.UserAnswer,
			CorrectAnswer: question.CorrectAnswer,
			Explanation:   question.Explanation,
		})
	}

	return score, correctCount, questionResults
}

// parseJSONResponse 解析JSON响应
func parseJSONResponse(response string, target interface{}) error {
	// 清理响应文本，提取JSON部分
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return fmt.Errorf("无法找到有效的JSON内容")
	}

	jsonContent := response[jsonStart : jsonEnd+1]

	// 使用标准库解析JSON
	return json.Unmarshal([]byte(jsonContent), target)
}
