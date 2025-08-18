// pages/profile/learning/learning.js
const courseAPI = require('../../../apis/course.js')
const authAPI = require('../../../apis/auth.js')

Page({
  /**
   * 页面的初始数据
   */
  data: {
    // 学习记录列表
    learningRecords: [],
    
    // 筛选条件
    filterType: 'all', // all, in_progress, completed, expired
    filterCategory: 'all',
    
    // Picker索引
    statusIndex: 0,
    categoryIndex: 0,
    
    // 当前显示标签
    currentStatusLabel: '全部状态',
    currentCategoryLabel: '全部分类',
    
    // 分页
    page: 1,
    pageSize: 50,
    hasMore: true,
    loading: false,
    
    // 统计信息
    stats: {
      totalCourses: 0,
      completedCourses: 0,
      inProgressCourses: 0,
      totalStudyTime: 0,
      totalQuizScore: 0
    },
    
    // 分类选项
    categoryOptions: [
      { value: 'all', label: '全部分类' },
      { value: 'general', label: '通用' },
      { value: 'programming', label: '编程' },
      { value: 'business', label: '商业' },
      { value: 'education', label: '教育' },
      { value: 'science', label: '科学' },
      { value: 'literature', label: '文学' },
      { value: 'history', label: '历史' },
      { value: 'art', label: '艺术' }
    ],
    
    // 状态选项
    statusOptions: [
      { value: 'all', label: '全部状态' },
      { value: 'in_progress', label: '学习中' },
      { value: 'completed', label: '已完成' },
      { value: 'expired', label: '已过期' }
    ]
  },

  /**
   * 生命周期函数--监听页面加载
   */
  onLoad(options) {
    this.loadLearningRecords()
    this.loadStats()
  },

  /**
   * 生命周期函数--监听页面初次渲染完成
   */
  onReady() {

  },

  /**
   * 生命周期函数--监听页面显示
   */
  onShow() {
    // 每次显示页面时重新加载数据
    this.refreshData()
  },

  /**
   * 生命周期函数--监听页面隐藏
   */
  onHide() {

  },

  /**
   * 生命周期函数--监听页面卸载
   */
  onUnload() {

  },

  /**
   * 加载学习记录
   */
  async loadLearningRecords(refresh = false) {
    if (this.data.loading) return
    
    this.setData({ loading: true })
    
    try {
      const params = {
        page: refresh ? 1 : this.data.page,
        page_size: this.data.pageSize,
        status: this.data.filterType === 'all' ? '' : this.data.filterType,
        category: this.data.filterCategory === 'all' ? '' : this.data.filterCategory
      }
      
      const result = await courseAPI.getLearningRecords(params)
      
      // 处理数据，限制进度值并映射状态
      const processedRecords = (result.records || []).map(record => ({
        ...record,
        // 使用数据库同步的complete_rate作为进度显示
        progress: Math.min(Math.max(record.progress || 0, 0), 100),
        // 映射状态：learning -> in_progress，保持其他状态不变
        status: record.status === 'learning' ? 'in_progress' : record.status
      }))
      
      if (refresh) {
        this.setData({
          learningRecords: processedRecords,
          page: 1,
          hasMore: processedRecords.length >= this.data.pageSize
        })
      } else {
        this.setData({
          learningRecords: [...this.data.learningRecords, ...processedRecords],
          page: this.data.page + 1,
          hasMore: processedRecords.length >= this.data.pageSize
        })
      }
    } catch (error) {
      console.error('加载学习记录失败:', error)
      wx.showToast({
        title: '加载失败',
        icon: 'none'
      })
    } finally {
      this.setData({ loading: false })
      wx.stopPullDownRefresh()
    }
  },

  /**
   * 加载统计信息
   */
  async loadStats() {
    try {
      const result = await courseAPI.getLearningStats()
      this.setData({
        stats: result || this.data.stats
      })
    } catch (error) {
      console.error('加载统计信息失败:', error)
    }
  },

  /**
   * 刷新数据
   */
  refreshData() {
    this.loadLearningRecords(true)
    this.loadStats()
  },

  /**
   * 筛选状态变化
   */
  onStatusFilterChange(e) {
    const index = e.detail.value
    const filterType = this.data.statusOptions[index].value
    const currentStatusLabel = this.data.statusOptions[index].label
    
    this.setData({
      filterType: filterType,
      statusIndex: index,
      currentStatusLabel: currentStatusLabel,
      learningRecords: [],
      page: 1,
      hasMore: true
    })
    this.loadLearningRecords(true)
  },

  /**
   * 筛选分类变化
   */
  onCategoryFilterChange(e) {
    const index = e.detail.value
    const filterCategory = this.data.categoryOptions[index].value
    const currentCategoryLabel = this.data.categoryOptions[index].label
    
    this.setData({
      filterCategory: filterCategory,
      categoryIndex: index,
      currentCategoryLabel: currentCategoryLabel,
      learningRecords: [],
      page: 1,
      hasMore: true
    })
    this.loadLearningRecords(true)
  },

  /**
   * 继续学习
   */
  continueLearning(e) {
    const courseId = e.currentTarget.dataset.id
    wx.navigateTo({
      url: `/pages/player/player?courseId=${courseId}`
    })
  },

  /**
   * 查看课程详情
   */
  viewCourseDetail(e) {
    const courseId = e.currentTarget.dataset.id
    wx.navigateTo({
      url: `/pages/course/detail/detail?id=${courseId}`
    })
  },

  /**
   * 重新开始学习
   */
  restartLearning(e) {
    const courseId = e.currentTarget.dataset.id
    wx.showModal({
      title: '确认重新开始',
      content: '重新开始将清空当前学习进度，确定要继续吗？',
      success: (res) => {
        if (res.confirm) {
          this.resetLearningProgress(courseId)
        }
      }
    })
  },

  /**
   * 重置学习进度
   */
  async resetLearningProgress(courseId) {
    try {
      await courseAPI.resetLearningProgress(courseId)
      wx.showToast({
        title: '重置成功',
        icon: 'success'
      })
      this.refreshData()
    } catch (error) {
      console.error('重置学习进度失败:', error)
      wx.showToast({
        title: '重置失败',
        icon: 'none'
      })
    }
  },

  /**
   * 删除学习记录
   */
  deleteRecord(e) {
    const recordId = e.currentTarget.dataset.id
    wx.showModal({
      title: '确认删除',
      content: '删除后将无法恢复，确定要删除吗？',
      success: (res) => {
        if (res.confirm) {
          this.confirmDeleteRecord(recordId)
        }
      }
    })
  },

  /**
   * 确认删除记录
   */
  async confirmDeleteRecord(recordId) {
    try {
      await courseAPI.deleteLearningRecord(recordId)
      wx.showToast({
        title: '删除成功',
        icon: 'success'
      })
      this.refreshData()
    } catch (error) {
      console.error('删除学习记录失败:', error)
      wx.showToast({
        title: '删除失败',
        icon: 'none'
      })
    }
  },

  /**
   * 页面相关事件处理函数--监听用户下拉动作
   */
  onPullDownRefresh() {
    this.refreshData()
  },

  /**
   * 页面上拉触底事件的处理函数
   */
  onReachBottom() {
    if (this.data.hasMore && !this.data.loading) {
      this.loadLearningRecords()
    }
  },

  /**
   * 格式化学习时长
   */
  formatStudyTime(minutes) {
    if (minutes < 60) {
      return `${minutes}分钟`
    } else {
      const hours = Math.floor(minutes / 60)
      const mins = minutes % 60
      return mins > 0 ? `${hours}小时${mins}分钟` : `${hours}小时`
    }
  },

  /**
   * 格式化日期
   */
  formatDate(dateString) {
    const date = new Date(dateString)
    const now = new Date()
    const diff = now - date
    
    if (diff < 24 * 60 * 60 * 1000) {
      return '今天'
    } else if (diff < 2 * 24 * 60 * 60 * 1000) {
      return '昨天'
    } else if (diff < 7 * 24 * 60 * 60 * 1000) {
      return `${Math.floor(diff / (24 * 60 * 60 * 1000))}天前`
    } else {
      return `${date.getMonth() + 1}月${date.getDate()}日`
    }
  },

  /**
   * 用户点击右上角分享
   */
  onShareAppMessage() {
    return {
      title: '我的学习记录',
      path: '/pages/profile/learning/learning'
    }
  },

  /**
   * 跳转到课程列表
   */
  goToCourseList() {
    wx.navigateTo({
      url: '/pages/course/list/list'
    })
  },



})