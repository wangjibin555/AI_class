// pages/quiz/result/result.js
const quizAPI = require('../../../apis/quiz.js')

Page({
  /**
   * 页面的初始数据
   */
  data: {
    attemptId: null,
    quizResult: null,
    questionResults: [],
    
    // 统计信息
    score: 0,
    totalScore: 0,
    percentage: 0,
    correctCount: 0,
    totalCount: 0,
    timeSpent: 0,
    completedAt: null,
    
    // 界面状态
    loading: false,
    currentTab: 'summary', // summary, details
    showAnswer: false
  },

  /**
   * 生命周期函数--监听页面加载
   */
  onLoad(options) {
    console.log('练习结果页面加载', options)
    
    if (!options.attemptId) {
      wx.showToast({
        title: '缺少答题记录ID',
        icon: 'none'
      })
      setTimeout(() => {
        wx.navigateBack()
      }, 1500)
      return
    }
    
    this.setData({ 
      attemptId: options.attemptId
    })
    
    this.loadQuizResult()
  },

  /**
   * 加载练习结果
   */
  async loadQuizResult() {
    if (this.data.loading) return
    
    this.setData({ loading: true })
    
    try {
      const response = await quizAPI.getAttemptDetail(this.data.attemptId)
      const attempt = response.data || response
      
      this.setData({
        quizResult: attempt,
        score: attempt.score || 0,
        totalScore: attempt.question_count * 10, // 每题10分
        correctCount: attempt.correct_count || 0,
        totalCount: attempt.question_count || 0,
        timeSpent: attempt.time_spent || 0,
        completedAt: attempt.end_time,
        questionResults: attempt.question_results || []
      })
      
      // 计算正确率
      const percentage = this.data.totalCount > 0 ? 
        (this.data.correctCount / this.data.totalCount) * 100 : 0
      
      this.setData({ percentage })
      
      // 设置页面标题
      wx.setNavigationBarTitle({
        title: '练习结果'
      })
      
    } catch (error) {
      console.error('加载练习结果失败:', error)
      wx.showToast({
        title: '加载失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  /**
   * 切换标签页
   */
  switchTab(e) {
    const tab = e.currentTarget.dataset.tab
    this.setData({ currentTab: tab })
  },

  /**
   * 切换答案显示
   */
  toggleAnswer() {
    this.setData({
      showAnswer: !this.data.showAnswer
    })
  },

  /**
   * 重新练习
   */
  retryQuiz() {
    wx.showModal({
      title: '重新练习',
      content: '确定要重新开始练习吗？',
      success: (res) => {
        if (res.confirm) {
          // 返回上一页并重新开始
          wx.navigateBack()
        }
      }
    })
  },

  /**
   * 分享结果
   */
  shareResult() {
    const { score, totalScore, percentage, correctCount, totalCount } = this.data
    
    wx.showShareMenu({
      withShareTicket: true,
      menus: ['shareAppMessage', 'shareTimeline']
    })
  },

  /**
   * 返回课程
   */
  backToCourse() {
    wx.navigateBack({
      delta: 2 // 返回两层，跳过练习页面
    })
  },

  /**
   * 格式化时间
   */
  formatTime(seconds) {
    if (!seconds) return '0:00'
    
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins}:${secs.toString().padStart(2, '0')}`
  },

  /**
   * 格式化日期
   */
  formatDate(dateString) {
    if (!dateString) return ''
    
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  },

  /**
   * 获取成绩等级
   */
  getGrade() {
    const percentage = this.data.percentage
    
    if (percentage >= 90) return { grade: 'A', text: '优秀', color: '#4caf50' }
    if (percentage >= 80) return { grade: 'B', text: '良好', color: '#2196f3' }
    if (percentage >= 70) return { grade: 'C', text: '中等', color: '#ff9800' }
    if (percentage >= 60) return { grade: 'D', text: '及格', color: '#ff5722' }
    return { grade: 'F', text: '不及格', color: '#f44336' }
  },

  /**
   * 页面分享
   */
  onShareAppMessage() {
    const { score, totalScore, percentage, correctCount, totalCount } = this.data
    const grade = this.getGrade()
    
    return {
      title: `我在练习中获得了${score}分，正确率${percentage.toFixed(1)}%，等级${grade.grade}！`,
      desc: `答对了${correctCount}/${totalCount}题，来挑战一下吧！`,
      path: '/pages/index/index'
    }
  }
})