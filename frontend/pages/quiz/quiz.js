// pages/quiz/quiz.js
const quizAPI = require('../../apis/quiz.js')
const courseAPI = require('../../apis/course.js')

Page({
  /**
   * 页面的初始数据
   */
  data: {
    courseId: null,
    courseInfo: null,
    quizId: null,
    attemptId: null,
    
    // 练习信息
    quiz: null,
    questions: [],
    currentQuestionIndex: 0,
    totalQuestions: 0,
    
    // 用户选择的参数
    userSelectedDifficulty: 'normal',
    userSelectedQuestionCount: 8,
    difficultyDisplay: '普通',
    
    // 答题状态
    isStarted: false,
    isCompleted: false,
    timeLimit: 0,
    timeRemaining: 0,
    startTime: null,
    timer: null,
    
    // 当前题目
    currentQuestion: null,
    userAnswer: '',
    userAnswers: [],
    selectedOption: '',  // 新增：用于显示选中状态的完整选项文本
    selectedOptions: [],
    
    // 答题结果
    score: 0,
    totalScore: 0,
    correctCount: 0,
    percentage: 0,
    questionResults: [],
    
    // 界面状态
    loading: false,
    submitting: false,
    showResult: false,
    showExplanation: false
  },

  /**
   * 生命周期函数--监听页面加载
   */
  onLoad(options) {
    console.log('练习页面加载', options)
    
    if (!options.courseId) {
      wx.showToast({
        title: '缺少课程ID',
        icon: 'none'
      })
      setTimeout(() => {
        wx.navigateBack()
      }, 1500)
      return
    }
    
    const selectedDifficulty = options.difficulty || 'normal'
    const selectedQuestionCount = parseInt(options.questionCount) || 8
    
    console.log('🔍 接收到的参数:', {
      difficulty: options.difficulty,
      questionCount: options.questionCount,
      selectedDifficulty,
      selectedQuestionCount
    })
    
    this.setData({ 
      courseId: options.courseId,
      quizId: options.quizId,
      // 接收用户选择的参数
      userSelectedDifficulty: selectedDifficulty,
      userSelectedQuestionCount: selectedQuestionCount,
      // 预计算难度显示
      difficultyDisplay: this.formatDifficulty(selectedDifficulty)
    })
    
    this.loadCourseInfo()
    this.loadQuizData()
  },

  /**
   * 生命周期函数--监听页面卸载
   */
  onUnload() {
    // 清理定时器
    if (this.data.timer) {
      clearInterval(this.data.timer)
    }
  },

  /**
   * 加载课程信息
   */
  async loadCourseInfo() {
    try {
      const response = await courseAPI.getCourseDetail(this.data.courseId)
      const courseInfo = response.data || response
      
      this.setData({ courseInfo })
      
      // 设置页面标题
      wx.setNavigationBarTitle({
        title: `${courseInfo.title} - 练习`
      })
    } catch (error) {
      console.error('加载课程信息失败:', error)
    }
  },

  /**
   * 加载练习数据
   */
  async loadQuizData() {
    if (this.data.loading) return
    
    this.setData({ loading: true })
    
    try {
      if (this.data.quizId) {
        // 加载现有练习
        const response = await quizAPI.getQuiz(this.data.quizId)
        const quiz = response.data || response
        
        console.log('🔍 加载的quiz数据:', quiz)
        console.log('🔍 questions数据:', quiz.questions)
        
        // 确保所有 ID 都是数字类型
        const questions = (quiz.questions || []).map(question => ({
          ...question,
          id: parseInt(question.id) // 强制转换ID为数字
        }))
        
        console.log('🔍 处理后questions数据:', questions)
        
        this.setData({
          quiz,
          questions: questions,
          // 优先显示用户选择的参数，如果没有则使用实际练习数据
          totalQuestions: this.data.userSelectedQuestionCount || quiz.question_count || 0,
          timeLimit: quiz.time_limit || 0
        })
      } else {
        // 生成新练习
        await this.generateNewQuiz()
      }
      
      // 设置当前题目
      this.setCurrentQuestion()
      
    } catch (error) {
      console.error('加载练习数据失败:', error)
      wx.showToast({
        title: '加载失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  /**
   * 生成新练习
   */
  async generateNewQuiz() {
    try {
      const quizRequest = {
        question_count: this.data.userSelectedQuestionCount,
        difficulty: this.data.userSelectedDifficulty,
        question_types: ['single_choice', 'multiple_choice', 'fill_blank'],
        time_limit: 30
      }
      
      console.log('🧠 使用用户选择的参数生成练习:', quizRequest)
      
      const response = await quizAPI.generateQuiz(this.data.courseId, quizRequest)
      const quiz = response.data || response
      
      // 确保所有 ID 都是数字类型
      const questions = (quiz.questions || []).map(question => ({
        ...question,
        id: parseInt(question.id) // 强制转换ID为数字
      }))
      
      this.setData({
        quizId: quiz.id,
        quiz,
        questions: questions,
        totalQuestions: quiz.question_count || 0,
        timeLimit: quiz.time_limit || 0
      })
      
    } catch (error) {
      console.error('生成练习失败:', error)
      throw error
    }
  },

  /**
   * 开始练习
   */
  async startQuiz() {
    if (this.data.submitting) return
    
    this.setData({ submitting: true })
    
    try {
      const response = await quizAPI.startQuiz(this.data.quizId)
      const attempt = response.data || response
      
      this.setData({
        attemptId: attempt.id,
        isStarted: true,
        startTime: Date.now(),
        timeRemaining: this.data.timeLimit * 60 // 转换为秒
      })
      
      // 只有设置了时间限制才开始计时
      if (this.data.timeLimit > 0) {
        this.startTimer()
      }
      
    } catch (error) {
      console.error('开始练习失败:', error)
      wx.showToast({
        title: '开始失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ submitting: false })
    }
  },

  /**
   * 开始计时
   */
  startTimer() {
    const timer = setInterval(() => {
      const timeRemaining = this.data.timeRemaining - 1
      
      if (timeRemaining <= 0) {
        // 时间到，自动提交
        clearInterval(timer)
        this.submitQuiz()
      } else {
        this.setData({ timeRemaining })
      }
    }, 1000)
    
    this.setData({ timer })
  },

  /**
   * 设置当前题目
   */
  setCurrentQuestion() {
    if (this.data.questions.length === 0) {
      console.log('❌ questions数组为空')
      return
    }
    
    const currentQuestion = this.data.questions[this.data.currentQuestionIndex]
    console.log('🔍 当前题目数据:', currentQuestion)
    console.log('🔍 题目索引:', this.data.currentQuestionIndex)
    console.log('🔍 总题目数:', this.data.questions.length)
    
    const userAnswer = this.data.userAnswers[this.data.currentQuestionIndex] || ''
    const selectedOptions = this.data.selectedOptions[this.data.currentQuestionIndex] || []
    
    // 恢复选中状态：根据userAnswer找到对应的完整选项文本
    let selectedOption = ''
    if (userAnswer && currentQuestion.options) {
      selectedOption = currentQuestion.options.find(option => 
        this.extractAnswerKey(option) === userAnswer
      ) || ''
    }
    
    this.setData({
      currentQuestion,
      userAnswer,
      selectedOption,  // 设置选中的完整选项文本
      selectedOptions
    })
  },

  /**
   * 统一的选项点击处理
   */
  handleOptionTap(e) {
    if (!this.data.currentQuestion) return
    
    const option = e.currentTarget.dataset.option
    console.log('🔍 点击选项:', option)
    console.log('🔍 题目类型:', this.data.currentQuestion.type)
    
    if (this.data.currentQuestion.type === 'single_choice') {
      this.selectOption(e)
    } else if (this.data.currentQuestion.type === 'multiple_choice') {
      this.selectMultipleOptions(e)
    }
  },

  /**
   * 提取答案选项的字母部分 (A/B/C/D)
   */
  extractAnswerKey(option) {
    if (!option) return ''
    
    // 从 "A: 文本内容" 中提取 "A"
    const colonIndex = option.indexOf(':')
    if (colonIndex > 0) {
      return option.substring(0, colonIndex).trim()
    }
    
    // 如果没有冒号，检查是否以字母开头
    const match = option.match(/^[A-Z]/i)
    return match ? match[0].toUpperCase() : option
  },

  /**
   * 选择选项（单选）
   */
  selectOption(e) {
    const option = e.currentTarget.dataset.option
    console.log('✅ 单选题选择:', option)
    
    // 提取答案字母 (A/B/C/D)
    const answerKey = this.extractAnswerKey(option)
    console.log('🔑 提取的答案字母:', answerKey)
    
    // 保存到当前题目的答案
    const userAnswers = [...this.data.userAnswers]
    userAnswers[this.data.currentQuestionIndex] = answerKey  // 存储字母而非完整文本
    
    this.setData({
      userAnswer: answerKey,  // 修改为字母
      selectedOption: option,  // 保存完整文本用于选中状态显示
      selectedOptions: [option],  // 保持完整文本用于显示
      userAnswers: userAnswers
    })
    
    // 强制更新视图
    this.setData({
      currentQuestion: this.data.currentQuestion
    })
    
    console.log('✅ 选择后状态:', {
      userAnswer: answerKey,
      selectedOptions: [option],
      questionIndex: this.data.currentQuestionIndex
    })
  },

  /**
   * 选择选项（多选）
   */
  selectMultipleOptions(e) {
    const option = e.currentTarget.dataset.option
    console.log('✅ 多选题选择:', option)
    
    const selectedOptions = [...this.data.selectedOptions]
    
    const index = selectedOptions.indexOf(option)
    if (index > -1) {
      selectedOptions.splice(index, 1)
      console.log('❌ 取消选择:', option)
    } else {
      selectedOptions.push(option)
      console.log('✅ 添加选择:', option)
    }
    
    // 提取每个选项的字母部分
    const answerKeys = selectedOptions.map(opt => this.extractAnswerKey(opt))
    const userAnswer = answerKeys.join(',')  // "A,C" 格式
    
    console.log('🔑 提取的答案字母数组:', answerKeys)
    console.log('🔑 最终答案字符串:', userAnswer)
    
    // 保存到当前题目的答案
    const userAnswers = [...this.data.userAnswers]
    userAnswers[this.data.currentQuestionIndex] = userAnswer
    
    this.setData({
      selectedOptions,
      userAnswer: userAnswer,
      userAnswers: userAnswers
    })
    
    // 强制更新视图
    this.setData({
      currentQuestion: this.data.currentQuestion
    })
    
    console.log('✅ 多选选择后状态:', {
      selectedOptions,
      userAnswer,
      answerKeys,
      questionIndex: this.data.currentQuestionIndex
    })
  },

  /**
   * 输入答案（填空题）
   */
  onAnswerInput(e) {
    const answer = e.detail.value
    console.log('✅ 填空题输入:', answer)
    
    // 保存到当前题目的答案
    const userAnswers = [...this.data.userAnswers]
    userAnswers[this.data.currentQuestionIndex] = answer
    
    this.setData({
      userAnswer: answer,
      userAnswers: userAnswers
    })
  },

  /**
   * 提交当前答案
   */
  async submitAnswer() {
    console.log('🔍 提交前验证 - userAnswer:', this.data.userAnswer)
    console.log('🔍 提交前验证 - selectedOptions:', this.data.selectedOptions)
    
    if (!this.data.userAnswer || this.data.userAnswer.trim() === '') {
      wx.showToast({
        title: '请先选择答案',
        icon: 'none'
      })
      return
    }
    
    if (this.data.submitting) return
    
    this.setData({ submitting: true })
    
    try {
      console.log('🔍 当前题目数据:', this.data.currentQuestion)
      console.log('🔍 原始question_id:', this.data.currentQuestion.id, typeof this.data.currentQuestion.id)
      
      // 确保question_id是数字类型
      const questionId = parseInt(this.data.currentQuestion.id)
      
      console.log('🔍 转换后question_id:', questionId, typeof questionId)
      
      const request = {
        question_id: questionId,
        answer: this.data.userAnswer,
        answers: this.data.selectedOptions || []
      }
      
      console.log('🔍 最终请求数据:', request)
      
      const response = await quizAPI.submitAnswer(this.data.attemptId, request)
      const result = response.data || response
      
      // 保存用户答案
      const userAnswers = [...this.data.userAnswers]
      userAnswers[this.data.currentQuestionIndex] = this.data.userAnswer
      
      this.setData({
        userAnswers,
        showExplanation: true
      })
      
      // 显示答案结果
      wx.showToast({
        title: result.is_correct ? '回答正确！' : '回答错误',
        icon: result.is_correct ? 'success' : 'error'
      })
      
    } catch (error) {
      console.error('提交答案失败:', error)
      wx.showToast({
        title: '提交失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ submitting: false })
    }
  },

  /**
   * 下一题
   */
  nextQuestion() {
    if (this.data.currentQuestionIndex < this.data.totalQuestions - 1) {
      const newIndex = this.data.currentQuestionIndex + 1
      this.setData({
        currentQuestionIndex: newIndex,
        showExplanation: false
      })
      this.setCurrentQuestion()
    } else {
      // 最后一题，显示提交按钮
      this.setData({
        showResult: true
      })
    }
  },

  /**
   * 上一题
   */
  prevQuestion() {
    if (this.data.currentQuestionIndex > 0) {
      const newIndex = this.data.currentQuestionIndex - 1
      this.setData({
        currentQuestionIndex: newIndex,
        showExplanation: false
      })
      this.setCurrentQuestion()
    }
  },

  /**
   * 提交练习
   */
  async submitQuiz() {
    if (this.data.submitting) return
    
    this.setData({ submitting: true })
    
    try {
      const response = await quizAPI.submitQuiz(this.data.attemptId)
      const result = response.data || response
      
      this.setData({
        isCompleted: true,
        score: result.score,
        totalScore: result.total_score,
        correctCount: result.correct_count,
        percentage: result.percentage,
        questionResults: result.question_results || []
      })
      
      // 停止计时
      if (this.data.timer) {
        clearInterval(this.data.timer)
      }
      
      // 显示结果
      this.showQuizResult()
      
    } catch (error) {
      console.error('提交练习失败:', error)
      wx.showToast({
        title: '提交失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ submitting: false })
    }
  },

  /**
   * 显示练习结果
   */
  showQuizResult() {
    const { score, totalScore, percentage, correctCount, totalQuestions } = this.data
    
    wx.showModal({
      title: '练习完成',
      content: `得分：${score}/${totalScore}\n正确率：${percentage.toFixed(1)}%\n正确题数：${correctCount}/${totalQuestions}`,
      showCancel: false,
      confirmText: '查看详情',
      success: (res) => {
        if (res.confirm) {
          // 跳转到结果页面
          wx.navigateTo({
            url: `/pages/quiz/result/result?attemptId=${this.data.attemptId}`
          })
        }
      }
    })
  },

  /**
   * 格式化时间
   */
  formatTime(seconds) {
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins}:${secs.toString().padStart(2, '0')}`
  },

  /**
   * 格式化进度
   */
  formatProgress() {
    return `${this.data.currentQuestionIndex + 1} / ${this.data.totalQuestions}`
  },

  /**
   * 转换难度显示
   */
  formatDifficulty(difficulty) {
    const difficultyMap = {
      'easy': '简单',
      'normal': '普通', 
      'hard': '困难'
    }
    return difficultyMap[difficulty] || '普通'
  },

  /**
   * 页面分享
   */
  onShareAppMessage() {
    const { courseInfo } = this.data
    if (!courseInfo) return {}
    
    return {
      title: `${courseInfo.title} - 练习`,
      desc: '来测试一下你的学习成果吧！',
      path: `/pages/quiz/quiz?courseId=${this.data.courseId}`
    }
  }
})