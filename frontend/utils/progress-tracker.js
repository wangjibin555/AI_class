// utils/progress-tracker.js
// 学习进度追踪器

class LearningProgressTracker {
  constructor(courseId, learningRecordId = null) {
    this.courseId = courseId
    this.learningRecordId = learningRecordId
    
    // 学习数据
    this.learningData = {
      startTime: null,
      endTime: null,
      totalTime: 0,
      activeTime: 0,
      pauseTime: 0,
      currentSlide: 0,
      totalSlides: 0,
      completedSlides: new Set(),
      visitedSlides: new Set(),
      progress: 0,
      completionRate: 0,
      averageSlideTime: 0,
      fastestSlide: null,
      slowestSlide: null,
      retryCount: 0,
      pauseCount: 0,
      seekCount: 0
    }
    
    // 滑片级别数据
    this.slideData = new Map()
    
    // 行为数据
    this.behaviorLog = []
    
    // 定时器
    this.trackingTimer = null
    this.saveTimer = null
    
    // 状态
    this.isTracking = false
    this.isPaused = false
    this.lastActiveTime = Date.now()
    
    // 配置
    this.config = {
      saveInterval: 30000,      // 30秒保存一次
      inactiveThreshold: 30000, // 30秒未活动算作暂停
      autoSave: true,
      trackBehavior: true,
      trackDetailed: true
    }
    
    this.init()
  }
  
  init() {
    this.loadExistingProgress()
    this.bindEvents()
    console.log('LearningProgressTracker initialized for course:', this.courseId)
  }
  
  // 加载已有进度
  async loadExistingProgress() {
    try {
      const storageKey = `learning_progress_${this.courseId}`
      const savedProgress = wx.getStorageSync(storageKey)
      
      if (savedProgress) {
        this.learningData = { ...this.learningData, ...savedProgress }
        this.learningData.completedSlides = new Set(savedProgress.completedSlides || [])
        this.learningData.visitedSlides = new Set(savedProgress.visitedSlides || [])
        
        console.log('Loaded existing progress:', this.learningData)
      }
      
      // 如果有学习记录ID，从服务器同步
      if (this.learningRecordId) {
        await this.syncFromServer()
      }
    } catch (error) {
      console.warn('Failed to load existing progress:', error)
    }
  }
  
  // 从服务器同步进度
  async syncFromServer() {
    try {
      const { getBaseURLSync } = require('./config')
      const baseURL = getBaseURLSync()
      
      const response = await new Promise((resolve, reject) => {
        wx.request({
          url: `${baseURL}/learning/record/${this.learningRecordId}`,
          method: 'GET',
          header: {
            'Authorization': `Bearer ${wx.getStorageSync('access_token')}`
          },
          success: resolve,
          fail: reject
        })
      })
      
      if (response.statusCode === 200 && response.data && response.data.code === 200) {
        const serverData = response.data.data
        this.learningData.currentSlide = serverData.current_slide || 0
        this.learningData.progress = serverData.progress || 0
        this.learningData.totalTime = serverData.study_duration || 0
        this.learningData.completionRate = serverData.complete_rate || 0
        
        console.log('Synced progress from server:', serverData)
      } else {
        console.warn('Failed to sync from server: Invalid response', response)
      }
    } catch (error) {
      console.warn('Failed to sync from server:', error)
    }
  }
  
  // 绑定事件监听
  bindEvents() {
    // 页面可见性变化
    if (typeof wx !== 'undefined') {
      wx.onAppShow(() => {
        this.handleAppShow()
      })
      
      wx.onAppHide(() => {
        this.handleAppHide()
      })
    }
  }
  
  // 开始学习追踪
  startTracking(totalSlides = 0) {
    if (this.isTracking) return
    
    this.isTracking = true
    this.learningData.startTime = Date.now()
    this.learningData.totalSlides = totalSlides
    this.lastActiveTime = Date.now()
    
    // 记录开始行为
    this.recordBehavior('start_learning', {
      course_id: this.courseId,
      start_time: this.learningData.startTime,
      total_slides: totalSlides
    })
    
    // 开始定时追踪
    this.startTimerTracking()
    
    // 开始自动保存
    if (this.config.autoSave) {
      this.startAutoSave()
    }
    
    console.log('Started learning tracking for course:', this.courseId)
  }
  
  // 停止学习追踪
  stopTracking() {
    if (!this.isTracking) return
    
    this.isTracking = false
    this.learningData.endTime = Date.now()
    
    // 计算总学习时间
    if (this.learningData.startTime) {
      this.learningData.totalTime = this.learningData.endTime - this.learningData.startTime
    }
    
    // 记录结束行为
    this.recordBehavior('end_learning', {
      course_id: this.courseId,
      end_time: this.learningData.endTime,
      total_time: this.learningData.totalTime,
      completion_rate: this.learningData.completionRate
    })
    
    // 停止定时器
    this.stopTimerTracking()
    this.stopAutoSave()
    
    // 最终保存
    this.saveProgress()
    
    console.log('Stopped learning tracking, total time:', this.learningData.totalTime)
  }
  
  // 暂停追踪
  pauseTracking() {
    if (this.isPaused) return
    
    this.isPaused = true
    this.learningData.pauseCount++
    
    this.recordBehavior('pause_learning', {
      current_slide: this.learningData.currentSlide,
      pause_count: this.learningData.pauseCount
    })
    
    console.log('Learning tracking paused')
  }
  
  // 恢复追踪
  resumeTracking() {
    if (!this.isPaused) return
    
    this.isPaused = false
    this.lastActiveTime = Date.now()
    
    this.recordBehavior('resume_learning', {
      current_slide: this.learningData.currentSlide
    })
    
    console.log('Learning tracking resumed')
  }
  
  // 更新当前幻灯片
  updateCurrentSlide(slideIndex, slideData = {}) {
    const previousSlide = this.learningData.currentSlide
    const slideStartTime = Date.now()
    
    // 🔥 改进：如果跳过了幻灯片，标记所有跳过的幻灯片为完成
    if (slideIndex > previousSlide && previousSlide >= 0) {
      // 标记从previousSlide到slideIndex-1的所有幻灯片为完成
      for (let i = previousSlide; i < slideIndex; i++) {
        if (!this.learningData.completedSlides.has(i)) {
          this.completeSlide(i)
          console.log(`📝 [追踪器] 自动完成跳过的幻灯片 ${i}`)
        }
      }
      
      // 如果当前幻灯片还没有完成，也自动标记为完成
      if (!this.learningData.completedSlides.has(slideIndex)) {
        this.completeSlide(slideIndex)
        console.log(`📝 [追踪器] 自动完成当前幻灯片 ${slideIndex}`)
      }
    }
    
    // 更新上一张幻灯片的学习时间
    if (previousSlide !== slideIndex && this.slideData.has(previousSlide)) {
      const prevSlideData = this.slideData.get(previousSlide)
      if (prevSlideData && prevSlideData.startTime) {
        prevSlideData.studyTime += slideStartTime - prevSlideData.lastUpdateTime
        prevSlideData.lastUpdateTime = slideStartTime
      }
    }
    
    // 更新当前幻灯片
    this.learningData.currentSlide = slideIndex
    this.learningData.visitedSlides.add(slideIndex)
    
    // 初始化或更新幻灯片数据
    if (!this.slideData.has(slideIndex)) {
      this.slideData.set(slideIndex, {
        slideIndex,
        title: slideData.title || '',
        visitCount: 0,
        studyTime: 0,
        startTime: slideStartTime,
        lastUpdateTime: slideStartTime,
        completed: false,
        interactions: []
      })
    }
    
    const currentSlideData = this.slideData.get(slideIndex)
    currentSlideData.visitCount++
    currentSlideData.lastUpdateTime = slideStartTime
    
    // 记录幻灯片切换行为
    this.recordBehavior('slide_change', {
      from_slide: previousSlide,
      to_slide: slideIndex,
      slide_title: slideData.title,
      timestamp: slideStartTime
    })
    
    // 更新进度
    this.updateProgress()
    
    console.log(`Slide changed from ${previousSlide} to ${slideIndex}`)
  }
  
  // 标记幻灯片完成
  completeSlide(slideIndex) {
    this.learningData.completedSlides.add(slideIndex)
    
    // 更新幻灯片数据
    if (this.slideData.has(slideIndex)) {
      const slideData = this.slideData.get(slideIndex)
      slideData.completed = true
      slideData.completedAt = Date.now()
    }
    
    // 记录完成行为
    this.recordBehavior('complete_slide', {
      slide_index: slideIndex,
      completed_count: this.learningData.completedSlides.size
    })
    
    // 更新完成率
    this.updateProgress()
    
    console.log(`Slide ${slideIndex} completed, total completed: ${this.learningData.completedSlides.size}`)
  }

  // 🔥 新增：批量完成所有幻灯片
  completeAllSlides() {
    for (let i = 0; i < this.learningData.totalSlides; i++) {
      this.learningData.completedSlides.add(i)
      
      // 更新幻灯片数据
      if (this.slideData.has(i)) {
        const slideData = this.slideData.get(i)
        slideData.completed = true
        slideData.completedAt = Date.now()
      }
    }
    
    // 记录批量完成行为
    this.recordBehavior('complete_all_slides', {
      total_slides: this.learningData.totalSlides,
      completion_method: 'auto'
    })
    
    this.updateProgress()
    this.saveProgress()
    console.log(`All ${this.learningData.totalSlides} slides completed automatically`)
  }
  
  // 记录拖拽/跳转行为
  recordSeek(fromTime, toTime, slideIndex) {
    this.learningData.seekCount++
    
    this.recordBehavior('seek', {
      from_time: fromTime,
      to_time: toTime,
      slide_index: slideIndex,
      seek_count: this.learningData.seekCount
    })
    
    // 更新幻灯片交互数据
    if (this.slideData.has(slideIndex)) {
      const slideData = this.slideData.get(slideIndex)
      slideData.interactions.push({
        type: 'seek',
        from_time: fromTime,
        to_time: toTime,
        timestamp: Date.now()
      })
    }
  }
  
  // 记录重试行为
  recordRetry(slideIndex, reason = '') {
    this.learningData.retryCount++
    
    this.recordBehavior('retry', {
      slide_index: slideIndex,
      reason: reason,
      retry_count: this.learningData.retryCount
    })
    
    console.log(`Retry recorded for slide ${slideIndex}, reason: ${reason}`)
  }
  
  // 更新学习进度
  updateProgress() {
    const totalSlides = this.learningData.totalSlides
    if (totalSlides === 0) return
    
    // 计算进度百分比
    this.learningData.progress = (this.learningData.currentSlide + 1) / totalSlides * 100
    
    // 计算完成率
    this.learningData.completionRate = this.learningData.completedSlides.size / totalSlides * 100
    
    // 计算平均每张幻灯片学习时间
    if (this.learningData.visitedSlides.size > 0) {
      const totalStudyTime = Array.from(this.slideData.values())
        .reduce((total, slide) => total + slide.studyTime, 0)
      this.learningData.averageSlideTime = totalStudyTime / this.learningData.visitedSlides.size
    }
    
    // 找出最快和最慢的幻灯片
    this.updateSlideSpeedStats()
    
    // 触发进度更新回调
    if (this.onProgressUpdate && typeof this.onProgressUpdate === 'function') {
      this.onProgressUpdate(this.learningData)
    }
    
    console.log(`Progress updated: ${this.learningData.progress.toFixed(1)}%, completion: ${this.learningData.completionRate.toFixed(1)}%`)
  }
  
  // 更新幻灯片速度统计
  updateSlideSpeedStats() {
    const slideTimes = Array.from(this.slideData.values())
      .filter(slide => slide.studyTime > 0)
      .map(slide => ({
        index: slide.slideIndex,
        time: slide.studyTime,
        title: slide.title
      }))
    
    if (slideTimes.length === 0) return
    
    // 找出最快的幻灯片
    this.learningData.fastestSlide = slideTimes.reduce((fastest, current) => 
      current.time < fastest.time ? current : fastest
    )
    
    // 找出最慢的幻灯片
    this.learningData.slowestSlide = slideTimes.reduce((slowest, current) => 
      current.time > slowest.time ? current : slowest
    )
  }
  
  // 记录行为日志
  recordBehavior(action, data = {}) {
    if (!this.config.trackBehavior) return
    
    const behaviorEntry = {
      action,
      timestamp: Date.now(),
      course_id: this.courseId,
      ...data
    }
    
    this.behaviorLog.push(behaviorEntry)
    
    // 限制日志长度，避免内存过大
    if (this.behaviorLog.length > 1000) {
      this.behaviorLog = this.behaviorLog.slice(-500)
    }
  }
  
  // 开始定时追踪
  startTimerTracking() {
    this.stopTimerTracking()
    
    this.trackingTimer = setInterval(() => {
      this.updateActiveTime()
    }, 1000) // 每秒更新一次
  }
  
  // 停止定时追踪
  stopTimerTracking() {
    if (this.trackingTimer) {
      clearInterval(this.trackingTimer)
      this.trackingTimer = null
    }
  }
  
  // 更新活跃时间
  updateActiveTime() {
    if (!this.isTracking || this.isPaused) return
    
    const now = Date.now()
    const timeDiff = now - this.lastActiveTime
    
    // 检查是否超过非活跃阈值
    if (timeDiff > this.config.inactiveThreshold) {
      this.pauseTracking()
      return
    }
    
    // 更新活跃时间
    this.learningData.activeTime += 1000 // 1秒
    
    // 更新当前幻灯片学习时间
    const currentSlideData = this.slideData.get(this.learningData.currentSlide)
    if (currentSlideData) {
      currentSlideData.studyTime += 1000
      currentSlideData.lastUpdateTime = now
      
      // 🔥 新增：基于学习时长自动标记完成（30秒规则）
      if (currentSlideData.studyTime > 30000 && !currentSlideData.completed) {
        if (!this.learningData.completedSlides.has(this.learningData.currentSlide)) {
          this.completeSlide(this.learningData.currentSlide)
          console.log(`Auto-completed slide ${this.learningData.currentSlide} after 30 seconds of study`)
        }
      }
    }
    
    this.lastActiveTime = now
  }
  
  // 开始自动保存
  startAutoSave() {
    this.stopAutoSave()
    
    this.saveTimer = setInterval(() => {
      this.saveProgress()
    }, this.config.saveInterval)
  }
  
  // 停止自动保存
  stopAutoSave() {
    if (this.saveTimer) {
      clearInterval(this.saveTimer)
      this.saveTimer = null
    }
  }
  
  // 保存进度到本地
  saveProgress() {
    try {
      const progressData = {
        ...this.learningData,
        completedSlides: Array.from(this.learningData.completedSlides),
        visitedSlides: Array.from(this.learningData.visitedSlides),
        slideData: Array.from(this.slideData.entries()),
        lastSaveTime: Date.now()
      }
      
      const storageKey = `learning_progress_${this.courseId}`
      wx.setStorageSync(storageKey, progressData)
      
      console.log('Progress saved to local storage')
      
      // 同步到服务器
      if (this.learningRecordId) {
        this.syncToServer()
      }
    } catch (error) {
      console.error('Failed to save progress:', error)
    }
  }
  
  // 同步到服务器
  async syncToServer() {
    try {
      const { getBaseURLSync } = require('./config')
      const baseURL = getBaseURLSync()
      
      const syncData = {
        learning_record_id: this.learningRecordId,
        current_slide: this.learningData.currentSlide,
        progress: this.learningData.progress / 100, // 转换为0-1
        study_duration: this.learningData.activeTime / 1000, // 转换为秒
        complete_rate: this.learningData.completionRate / 100,
        behavior_data: this.behaviorLog.slice(-100), // 只发送最近100条行为
        slide_data: this.getSlideDataForSync()
      }
      
      const response = await new Promise((resolve, reject) => {
        wx.request({
          url: `${baseURL}/learning/progress`,
          method: 'POST',
          header: {
            'Authorization': `Bearer ${wx.getStorageSync('access_token')}`,
            'Content-Type': 'application/json'
          },
          data: syncData,
          success: resolve,
          fail: reject
        })
      })
      
      if (response.statusCode === 200 && response.data && response.data.code === 200) {
        console.log('Progress synced to server successfully')
      } else {
        console.warn('Failed to sync to server: Invalid response', response)
      }
    } catch (error) {
      console.warn('Failed to sync to server:', error)
    }
  }
  
  // 获取用于同步的幻灯片数据
  getSlideDataForSync() {
    return Array.from(this.slideData.values()).map(slide => ({
      slide_index: slide.slideIndex,
      visit_count: slide.visitCount,
      study_time: slide.studyTime,
      completed: slide.completed,
      interaction_count: slide.interactions.length
    }))
  }
  
  // 处理应用显示
  handleAppShow() {
    if (this.isTracking && this.isPaused) {
      this.resumeTracking()
    }
  }
  
  // 处理应用隐藏
  handleAppHide() {
    if (this.isTracking && !this.isPaused) {
      this.pauseTracking()
    }
    
    // 立即保存进度
    this.saveProgress()
  }
  
  // 获取学习统计
  getStatistics() {
    const totalSlides = this.learningData.totalSlides
    const completedSlides = this.learningData.completedSlides.size
    const visitedSlides = this.learningData.visitedSlides.size
    
    return {
      // 基础统计
      total_slides: totalSlides,
      completed_slides: completedSlides,
      visited_slides: visitedSlides,
      current_slide: this.learningData.currentSlide,
      
      // 进度统计
      progress_percentage: this.learningData.progress,
      completion_rate: this.learningData.completionRate,
      
      // 时间统计
      total_time: this.learningData.totalTime,
      active_time: this.learningData.activeTime,
      average_slide_time: this.learningData.averageSlideTime,
      
      // 行为统计
      retry_count: this.learningData.retryCount,
      pause_count: this.learningData.pauseCount,
      seek_count: this.learningData.seekCount,
      
      // 效率指标
      efficiency_score: this.calculateEfficiencyScore(),
      engagement_score: this.calculateEngagementScore(),
      
      // 速度统计
      fastest_slide: this.learningData.fastestSlide,
      slowest_slide: this.learningData.slowestSlide,
      
      // 详细数据
      slide_data: Array.from(this.slideData.values()),
      behavior_summary: this.getBehaviorSummary()
    }
  }
  
  // 计算效率分数
  calculateEfficiencyScore() {
    if (this.learningData.totalSlides === 0) return 0
    
    const progressRate = this.learningData.progress / 100
    const timeEfficiency = this.learningData.activeTime / (this.learningData.totalTime || 1)
    const retryPenalty = Math.max(0, 1 - (this.learningData.retryCount * 0.1))
    
    return Math.min(100, (progressRate * timeEfficiency * retryPenalty) * 100)
  }
  
  // 计算参与度分数
  calculateEngagementScore() {
    if (this.learningData.totalSlides === 0) return 0
    
    const visitRate = this.learningData.visitedSlides.size / this.learningData.totalSlides
    const timeEngagement = Math.min(1, this.learningData.activeTime / (this.learningData.totalSlides * 60000)) // 假设每张1分钟
    const interactionScore = Math.min(1, this.behaviorLog.length / (this.learningData.totalSlides * 3)) // 假设每张3次交互
    
    return (visitRate * 0.4 + timeEngagement * 0.4 + interactionScore * 0.2) * 100
  }
  
  // 获取行为摘要
  getBehaviorSummary() {
    const behaviorCounts = {}
    
    this.behaviorLog.forEach(entry => {
      behaviorCounts[entry.action] = (behaviorCounts[entry.action] || 0) + 1
    })
    
    return behaviorCounts
  }
  
  // 重置进度数据
  resetProgress() {
    this.learningData = {
      startTime: null,
      endTime: null,
      totalTime: 0,
      activeTime: 0,
      pauseTime: 0,
      currentSlide: 0,
      totalSlides: 0,
      completedSlides: new Set(),
      visitedSlides: new Set(),
      progress: 0,
      completionRate: 0,
      averageSlideTime: 0,
      fastestSlide: null,
      slowestSlide: null,
      retryCount: 0,
      pauseCount: 0,
      seekCount: 0
    }
    
    this.slideData.clear()
    this.behaviorLog = []
    
    // 清除本地存储
    const storageKey = `learning_progress_${this.courseId}`
    wx.removeStorageSync(storageKey)
    
    console.log('Learning progress reset')
  }
  
  // 销毁追踪器
  destroy() {
    this.stopTracking()
    this.stopTimerTracking()
    this.stopAutoSave()
    
    // 最终保存
    this.saveProgress()
    
    console.log('LearningProgressTracker destroyed')
  }
}

module.exports = LearningProgressTracker 