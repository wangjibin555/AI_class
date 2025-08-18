// pages/player/player.js
const courseAPI = require('../../apis/course.js')
const LearningProgressTracker = require('../../utils/progress-tracker.js')
const AudioSyncManager = require('../../utils/audio-sync.js')
// 🔥 新增：引入字体适配器
const { calculateFontSize, SlideStyler, ContentAnalyzer } = require('../../utils/font-adapter.js')

Page({
  /**
   * 页面的初始数据
   */
  data: {
    courseId: null,
    courseInfo: null,
    slides: [],
    currentSlideIndex: 0,
    totalSlides: 0,
    
    // 播放模式 - 新增
    mode: 'text',              // 播放模式: audio, text
    audioMode: false,          // 音频模式开关
    startTime: null,           // 学习开始时间
    learningRecordId: null,    // 学习记录ID
    
    // 播放状态
    isPlaying: false,
    isAudioPlaying: false,
    audioDuration: 0,
    audioCurrentTime: 0,
    playbackRate: 1.0,         // 播放速度
    
    // 学习进度 - 增强
    learningProgress: {
      totalSlides: 0,
      currentSlide: 0,
      completedSlides: [],
      totalTime: 0,
      audioTime: 0,
      completionRate: 0
    },
    studyDuration: 0,
    
    // 🔥 新增：字体自适应相关数据
    currentSlideCharCount: 0,     // 当前幻灯片字符数
    currentFontSize: 36,          // 当前字体大小（rpx）
    currentContentClass: 'content-medium', // 当前内容样式类
    contentAnalysis: null,        // 内容分析结果
    
    // 界面状态
    showSlideList: false,
    showNotes: false,
    showProgress: false,
    showAudioSettings: false,  // 新增：音频设置面板
    showDropdownMenu: false,   // 下拉菜单显示状态
    
    // 音频控制 - 增强
    audioContext: null,
    audioSrc: '',
    audioController: null,     // 音频控制器
    syncManager: null,         // 音画同步管理器
    progressTracker: null,     // 进度追踪器
    isPaused: false,           // 暂停状态
    syncConfig: {              // 同步配置
      enabled: true,
      tolerance: 200,          // 200ms同步容差
      autoPlay: true,          // 启用自动播放
      preloadNext: true
    },
    
    // 音频设置
    audioSettings: {
		voiceType: 'zhixiaobai',     // 音色类型 - 默认使用知猫
      volume: 80,              // 音量
      autoPlayNext: true       // 自动播放下一张
    },
    
    // 加载状态
    loading: false,
    audioGenerating: false,    // 音频生成中
    audioCheckInterval: null,  // 音频检查定时器
    
    // 音频生成进度
    showAudioProgressModal: false,
    audioGenerationProgress: {
      status: 'idle',          // idle, generating, completed, failed
      currentSlide: 0,         // 当前正在生成的幻灯片
      totalSlides: 0,          // 总幻灯片数
      percentage: 0,           // 完成百分比
      message: '',             // 状态消息
      completedSlides: [],     // 已完成的幻灯片
      failedSlides: [],        // 失败的幻灯片
      errorMessage: '',        // 错误信息
      slideProgress: 0,        // 当前幻灯片进度
      slideProgressText: ''    // 当前幻灯片进度文本
    }
  },

  /**
   * 生命周期函数--监听页面加载
   */
  onLoad(options) {
    console.log('播放器页面加载', options)
    
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
    
    const slideIndex = parseInt(options.slideIndex) || 0
    const mode = options.mode || 'text'
    const startTime = options.startTime ? parseInt(options.startTime) : Date.now()
    
    this.setData({ 
      courseId: options.courseId,
      currentSlideIndex: slideIndex,
      mode: mode,
      audioMode: mode === 'audio',
      startTime: startTime
    })
    
    // 获取学习记录ID
    const learningRecordId = wx.getStorageSync(`learning_record_${options.courseId}`)
    if (learningRecordId) {
      this.setData({ learningRecordId })
    }
    
    this.loadCourseData()
  },

  /**
   * 生命周期函数--监听页面初次渲染完成
   */
  onReady() {
    // 创建音频上下文
    if (this.data.audioMode) {
      this.setData({
        audioContext: wx.createInnerAudioContext()
      })
      this.initAudioEventListeners()
      
      // 初始化音画同步管理器
      this.initSyncManager()
      
      // 初始化进度追踪器
      this.initProgressTracker()
    } else {
      // 对于非音频模式，也使用 InnerAudioContext 以确保兼容性
      this.setData({
        audioContext: wx.createInnerAudioContext()
      })
      this.initAudioEventListeners()
    }
  },

  /**
   * 初始化音画同步管理器
   */
  initSyncManager() {
    console.log('🔧 [同步] 初始化AudioSyncManager，autoPlayNext设置:', this.data.audioSettings.autoPlayNext)
    this.data.syncManager = new AudioSyncManager({
      tolerance: this.data.syncConfig.tolerance,
      checkInterval: 100,  // 100ms检查间隔
      autoCorrect: false,  // 🔥 禁用自动纠正，防止错误跳转
      preloadNext: this.data.syncConfig.preloadNext,
      autoPlayNext: this.data.audioSettings.autoPlayNext // 传递自动播放设置
    })
    
    // 设置音频上下文
    this.data.syncManager.setAudioContext(this.data.audioContext)
    
    // 设置幻灯片数据
    if (this.data.slides && this.data.slides.length > 0) {
      this.data.syncManager.setSlides(this.data.slides)
    }
    
    // 设置同步事件回调
    this.data.syncManager.on('syncError', (data) => {
      console.warn('音画同步错误:', data)
      // 可以在这里显示同步错误提示
    })
    
    this.data.syncManager.on('syncCorrected', (data) => {
      console.log('音画同步已纠正:', data)
    })
    
    // 🔴 移除slideChange事件监听，防止错误的幻灯片跳转
    // 幻灯片切换应该由用户操作或音频结束事件驱动
    /*
    this.data.syncManager.on('slideChange', (data) => {
      console.log('幻灯片自动切换:', data)
      // 自动切换到对应的幻灯片
      if (data.currentIndex !== this.data.currentSlideIndex) {
        this.setData({ 
          currentSlideIndex: data.currentIndex,
          'learningProgress.currentSlide': data.currentIndex
        })
        
        // 记录幻灯片访问
        this.recordSlideAccess(data.currentIndex)
      }
    })
    */
    
    this.data.syncManager.on('playStateChange', (data) => {
      console.log('播放状态变化:', data)
      this.setData({
        isPlaying: data.playing,
        isAudioPlaying: data.playing
      })
    })
  },

  /**
   * 初始化进度追踪器
   */
  initProgressTracker() {
    const learningRecordId = wx.getStorageSync(`learning_record_${this.data.courseId}`)
    this.data.progressTracker = new LearningProgressTracker(
      this.data.courseId,
      learningRecordId
    )
  },

  /**
   * 初始化音频事件监听器 - 新增
   */
  initAudioEventListeners() {
    const audioContext = this.data.audioContext
    if (!audioContext) return
    
    // 音频可以播放
    audioContext.onCanplay(() => {
      console.log('🎵 [播放器] 音频可以播放，时长:', audioContext.duration)
      // 更新音频时长
      this.setData({
        audioDuration: audioContext.duration || 0
      })
      // 预加载下一张幻灯片的音频
      if (this.data.syncManager && this.data.audioMode) {
        this.data.syncManager.preloadNextSlideAudio(this.data.currentSlideIndex)
      }
    })
    
    // 音频时间更新
    audioContext.onTimeUpdate(() => {
    this.setData({
        audioCurrentTime: audioContext.currentTime,
        audioDuration: audioContext.duration
      })
      this.updateLearningProgress()
    })
    
    // 音频播放结束
    audioContext.onEnded(() => {
      console.log('🎵 [播放器] 音频播放结束事件触发')
      this.setData({ 
        isAudioPlaying: false,
        audioCurrentTime: 0  // 🔥 重置播放进度为0
      })
      this.onAudioEnded()
    })
    
    // 音频播放开始
    audioContext.onPlay(() => {
      console.log('🎵 [播放器] 音频播放开始事件触发')
      this.setData({ 
        isAudioPlaying: true,
        isPlaying: true,
        isPaused: false
      })
    })
    
    // 音频暂停
    audioContext.onPause(() => {
      console.log('🎵 [播放器] 音频暂停事件触发')
      this.setData({ 
        isAudioPlaying: false,
        isPlaying: false,
        isPaused: true
      })
    })
    
    // 音频错误
    audioContext.onError((error) => {
      console.error('🎵 [播放器] 音频播放错误:', error)
      this.handleAudioError(error)
    })
  },

  /**
   * 生命周期函数--监听页面显示
   */
  onShow() {
    // 页面显示时恢复播放状态（仅在非暂停状态下）
    if (this.data.isPlaying && !this.data.isAudioPlaying && !this.data.isPaused) {
      console.log('🎵 [播放器] 页面显示，恢复播放')
      this.playAudio()
    }
    
    // 关闭下拉菜单
    if (this.data.showDropdownMenu) {
      this.setData({
        showDropdownMenu: false
      })
    }
  },

  /**
   * 生命周期函数--监听页面隐藏
   */
  onHide() {
    // 页面隐藏时暂停音频
    if (this.data.isAudioPlaying) {
      this.pauseAudio()
    }
    
    // 保存当前学习进度
    this.saveLearningProgress()
  },

  /**
   * 生命周期函数--监听页面卸载
   */
  onUnload() {
    // 保存最终学习进度
    this.saveLearningProgress()
    
    // 清理音频资源
    if (this.data.audioContext) {
      if (this.data.audioMode) {
      this.data.audioContext.destroy()
      }
    }
    
    // 清理定时器
    if (this.data.audioCheckInterval) {
      clearInterval(this.data.audioCheckInterval)
    }
  },

  /**
   * 加载课程数据 - 增强音频模式支持
   */
  async loadCourseData() {
    if (this.data.loading) return
    
    this.setData({ loading: true })
    
    try {
      console.log('播放器开始加载课程数据，courseId:', this.data.courseId)
      const response = await courseAPI.getCourseDetail(this.data.courseId)
      console.log('播放器课程API响应:', response)
      
      const courseInfo = response.data || response
      console.log('播放器解析的课程信息:', courseInfo)
      
      if (!courseInfo) {
        throw new Error('课程数据为空')
      }
      
      const slides = courseInfo.slides || []
      console.log('幻灯片数据:', slides, '数量:', slides.length)
      
      this.setData({
        courseInfo,
        slides: slides,
        totalSlides: slides.length,
        'learningProgress.totalSlides': slides.length
      })
      
      // 🔥 新增：初始化字体自适应
      this.initFontAdapter()
      
      // 设置页面标题
      wx.setNavigationBarTitle({
        title: courseInfo.title || '课程播放'
      })
      
      // 如果没有幻灯片数据，但有课程信息，创建一个默认幻灯片
      if (slides.length === 0 && courseInfo.title) {
        const defaultSlide = {
          id: 1,
          title: courseInfo.title,
          content: courseInfo.description || '开始学习本课程内容',
          slide_number: 1,
          audio_url: null,
          duration: 0
        }
        
        this.setData({
          slides: [defaultSlide],
          totalSlides: 1,
          'learningProgress.totalSlides': 1
        })
        
        console.log('创建了默认幻灯片:', defaultSlide)
      }
      
      // 加载学习进度
      this.loadLearningProgress()
      
      // 如果是音频模式，初始化音频功能
      if (this.data.audioMode) {
        this.initAudioMode()
      } else {
        // 设置当前幻灯片的音频（兼容原有功能）
      this.setCurrentSlideAudio()
      }
      
    } catch (error) {
      console.error('加载课程数据失败:', error)
      
      // 显示更友好的错误信息
      wx.showModal({
        title: '加载失败',
        content: `课程数据加载失败：${error.message}。是否返回上一页？`,
        confirmText: '返回',
        cancelText: '重试',
        success: (res) => {
          if (res.confirm) {
            wx.navigateBack()
          } else {
            // 重试加载
            this.loadCourseData()
          }
        }
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  /**
   * 加载学习进度
   */
  async loadLearningProgress() {
    try {
      const learningRecordId = wx.getStorageSync(`learning_record_${this.data.courseId}`)
      if (!learningRecordId) {
        console.log('没有找到学习记录ID，跳过加载学习进度')
        return
      }

      const learningAPI = require('../../apis/learning.js')
      const response = await learningAPI.getLearningRecord(learningRecordId)
      
      if (response && response.data) {
        const record = response.data
      this.setData({
          'learningProgress.currentSlide': record.current_slide || 0,
          'learningProgress.totalTime': record.study_duration || 0,
          'learningProgress.completionRate': record.complete_rate || 0,
          currentSlideIndex: record.current_slide || 0
        })
        
        console.log('学习进度加载成功:', record)
      }
    } catch (error) {
      console.warn('加载学习进度失败:', error)
      // 不影响主流程，只是记录错误
    }
  },

  /**
   * 初始化音频模式 - 新增功能
   */
  initAudioMode() {
    console.log('初始化音频模式')
    
    // 1. 检查音频文件是否存在
    this.checkAudioFiles()
    
    // 2. 初始化学习进度追踪
    this.initProgressTracking()
    
    // 3. 如果同步管理器已初始化，更新幻灯片数据
    if (this.data.syncManager && this.data.slides.length > 0) {
      this.data.syncManager.setSlides(this.data.slides)
    }
    
    // 4. 设置当前幻灯片音频
    this.setCurrentSlideAudio()
  },

  /**
   * 检查音频文件 - 新增功能
   */
  checkAudioFiles() {
    const slides = this.data.slides
    let hasAudio = false
    let missingAudioCount = 0
    
    if (!slides || slides.length === 0) {
      console.log('没有幻灯片数据，跳过音频检查')
      return
    }
    
    slides.forEach(slide => {
      if (slide.audio_url) {
        hasAudio = true
      } else {
        missingAudioCount++
      }
    })
    
    console.log(`音频检查结果: 总数${slides.length}, 有音频${slides.length - missingAudioCount}, 缺少音频${missingAudioCount}`)
    
    if (missingAudioCount > 0) {
      console.log(`有 ${missingAudioCount} 张幻灯片缺少音频，准备生成`)
      // 显示音频生成提示
      wx.showModal({
        title: '音频生成',
        content: `检测到${missingAudioCount}张幻灯片缺少音频，是否现在生成？`,
        confirmText: '生成',
        cancelText: '跳过',
        success: (res) => {
          if (res.confirm) {
            this.generateMissingAudio()
          } else {
            console.log('用户选择跳过音频生成')
          }
        }
      })
    } else if (hasAudio) {
      console.log('所有幻灯片都有音频文件')
      this.setCurrentSlideAudio()
    } else {
      console.log('所有幻灯片都没有音频，询问是否批量生成')
      wx.showModal({
        title: '生成课程音频',
        content: '该课程暂无音频讲解，是否为所有幻灯片生成音频？',
        confirmText: '生成',
        cancelText: '取消',
        success: (res) => {
          if (res.confirm) {
            this.generateCourseAudio()
          }
        }
      })
    }
  },

  /**
   * 生成缺失的音频 - 新增功能
   */
  generateMissingAudio() {
    // 初始化音频生成进度并显示模态框
    this.setData({ 
      audioGenerating: true,
      showAudioProgressModal: true, // 🆕 显示进度模态框
      audioGenerationProgress: {
        status: 'generating',
        currentSlide: 0,
        totalSlides: this.data.totalSlides,
        percentage: 0,
        message: '正在启动音频生成...',
        completedSlides: [],
        failedSlides: [],
        errorMessage: '',
        slideProgress: 0,
        slideProgressText: '正在初始化...'
      }
    })
    
    wx.showToast({
      title: '正在生成音频...',
      icon: 'loading',
      duration: 2000
    })
    
    // 获取有效token
    const token = this.getValidToken()
    
    // 获取API基础URL
    const { getBaseURLSync } = require('../../utils/config')
    const baseURL = getBaseURLSync()
    
    // 调用后端API生成音频
    wx.request({
      url: `${baseURL}/audio/generate/${this.data.courseId}`,
      method: 'POST',
      timeout: 300000, // 5分钟超时，适应音频生成的耗时操作
      header: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: {
        course_id: this.data.courseId,
        			voice_type: this.data.audioSettings.voiceType || 'zhixiaobai',
        regenerate: false
      },
      success: (res) => {
        console.log('音频生成API响应:', res)
        if (res.statusCode === 200 && res.data.code === 200) {
          console.log('音频生成启动成功')
          this.updateAudioGenerationProgress({
            message: '音频生成已启动，正在处理...'
          })
          this.startAudioGenerationCheck()
        } else {
          console.error('音频生成启动失败:', res.data)
          this.setData({ 
            audioGenerating: false,
            'audioGenerationProgress.status': 'failed',
            'audioGenerationProgress.message': '音频生成启动失败'
          })
          wx.showToast({
            title: '音频生成启动失败',
            icon: 'none'
          })
        }
      },
      fail: (error) => {
        console.error('音频生成API调用失败:', error)
        this.setData({ 
          audioGenerating: false,
          'audioGenerationProgress.status': 'failed',
          'audioGenerationProgress.message': '网络请求失败'
        })
        wx.showToast({
          title: '网络请求失败',
          icon: 'none'
        })
      }
    })
  },

  /**
   * 开始音频生成状态检查 - 新增功能
   */
  startAudioGenerationCheck() {
    const checkInterval = setInterval(() => {
      this.checkAudioGenerationStatus()
    }, 3000) // 每3秒检查一次
    
    this.setData({ audioCheckInterval: checkInterval })
  },

  /**
   * 检查音频生成状态 - 新增功能
   */
  checkAudioGenerationStatus() {
    const { getBaseURLSync } = require('../../utils/config')
    const baseURL = getBaseURLSync()
    
    wx.request({
      url: `${baseURL}/audio/status/${this.data.courseId}`,
      method: 'GET',
      header: {
        'Authorization': `Bearer ${this.getValidToken()}`
      },
      success: (res) => {
        console.log('音频状态检查响应:', res)
        if (res.statusCode === 200 && res.data.code === 200) {
          const status = res.data.data
          
          console.log('音频生成状态:', status)
          
          // 更新进度信息
          this.updateAudioGenerationProgressFromStatus(status)
          
          if (status.status === 'completed' || status.status === 'partial_success') {
            // 音频生成完成（包括部分成功）
            console.log('音频生成完成，状态:', status.status)
            this.onAudioGenerationComplete()
          } else if (status.status === 'failed') {
            // 音频生成失败
            this.onAudioGenerationFailed(status.error || '音频生成失败')
          } else if (status.status === 'processing') {
            // 继续等待
            console.log('音频生成进行中...')
          } else {
            // 其他未知状态
            console.log('未知音频生成状态:', status.status)
          }
        } else {
          console.error('音频状态检查失败:', res.data)
        }
      },
      fail: (error) => {
        console.error('音频状态检查网络失败:', error)
      }
    })
  },

  /**
   * 音频生成完成回调 - 新增功能
   */
  onAudioGenerationComplete() {
    console.log('音频生成完成')
    
    this.setData({ 
      audioGenerating: false,
      'audioGenerationProgress.status': 'completed',
      'audioGenerationProgress.message': '音频生成完成！',
      'audioGenerationProgress.percentage': 100
    })
    
    // 清理检查定时器
    if (this.data.audioCheckInterval) {
      clearInterval(this.data.audioCheckInterval)
      this.setData({ audioCheckInterval: null })
    }
    
    // 延迟2秒重新加载课程数据，确保数据库事务完成
    setTimeout(() => {
      console.log('延迟后重新加载课程数据')
      this.loadCourseData()
    }, 2000)
    
    wx.showToast({
      title: '音频生成完成',
      icon: 'success'
    })
  },

  /**
   * 音频生成失败回调 - 新增功能
   */
  onAudioGenerationFailed(error) {
    console.error('音频生成失败:', error)
    
    this.setData({ 
      audioGenerating: false,
      'audioGenerationProgress.status': 'failed',
      'audioGenerationProgress.message': `生成失败：${error}`
    })
    
    // 清理检查定时器
    if (this.data.audioCheckInterval) {
      clearInterval(this.data.audioCheckInterval)
      this.setData({ audioCheckInterval: null })
    }
    
    wx.showModal({
      title: '音频生成失败',
      content: `音频生成过程中出现错误：${error}。是否重试？`,
      confirmText: '重试',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          this.generateMissingAudio()
        }
      }
    })
  },

  /**
   * 更新音频生成进度
   */
  updateAudioGenerationProgress(updates) {
    const currentProgress = this.data.audioGenerationProgress
    const newProgress = { ...currentProgress, ...updates }
    
    console.log('🔧 更新音频生成进度:', { 
      current: currentProgress.percentage, 
      updates: updates.percentage, 
      new: newProgress.percentage,
      currentCompletedSlides: currentProgress.completedSlides?.length,
      updatesCompletedSlides: updates.completedSlides?.length,
      newCompletedSlides: newProgress.completedSlides?.length
    })
    
    // 🔧 修复：根据百分比重新计算completedSlides（如果不匹配）
    if (newProgress.percentage > 0 && newProgress.totalSlides > 0) {
      const currentCompleted = newProgress.completedSlides?.length || 0
      const expectedCompleted = Math.round((newProgress.percentage / 100) * newProgress.totalSlides)
      
      // 如果当前完成数量与期望的不匹配，重新计算
      if (currentCompleted !== expectedCompleted) {
        newProgress.completedSlides = Array(expectedCompleted).fill(0).map((_, i) => i + 1)
        console.log('🔧 根据百分比重新计算completedSlides:', {
          percentage: newProgress.percentage,
          totalSlides: newProgress.totalSlides,
          oldCompleted: currentCompleted,
          expectedCompleted: expectedCompleted,
          newCompletedSlides: newProgress.completedSlides
        })
      }
    }
    
    // 只有在没有明确传入percentage时才重新计算百分比
    if (updates.percentage === undefined && newProgress.totalSlides > 0 && newProgress.completedSlides) {
      const completedCount = Array.isArray(newProgress.completedSlides) ? 
        newProgress.completedSlides.length : newProgress.completedSlides
      newProgress.percentage = Math.round((completedCount / newProgress.totalSlides) * 100)
      console.log('🔧 重新计算的百分比:', newProgress.percentage)
    }
    
    this.setData({ audioGenerationProgress: newProgress })
  },

  /**
   * 从状态响应更新音频生成进度
   */
  updateAudioGenerationProgressFromStatus(status) {
    console.log('🔍 后端返回的完整状态数据:', status)
    
    const updates = {
      status: status.status
    }
    
    // 根据状态更新消息
    switch (status.status) {
      case 'processing':
        updates.message = `正在生成第 ${status.current_slide || 0} / ${status.total_slides || this.data.totalSlides} 张幻灯片的音频...`
        if (status.current_slide) {
          updates.currentSlide = status.current_slide
        }
        if (status.total_slides) {
          updates.totalSlides = status.total_slides
        }
        if (status.completed_slides) {
          updates.completedSlides = status.completed_slides
        }
        if (status.failed_slides) {
          updates.failedSlides = status.failed_slides
        }
        
        // 🔧 修复进度百分比提取
        if (status.progress && typeof status.progress === 'object') {
          // 从progress对象中提取百分比
          if (status.progress.percentage !== undefined) {
            updates.percentage = status.progress.percentage
          } else if (status.progress.progress !== undefined) {
            updates.percentage = status.progress.progress
          }
        }
        
        // 如果progress对象中没有百分比，尝试从message中解析
        if (updates.percentage === undefined && status.message) {
          const match = status.message.match(/(\d+\.?\d*)%/)
          if (match) {
            updates.percentage = parseFloat(match[1])
          }
        }
        
        // 如果还是没有百分比，根据完成的幻灯片数量计算
        if (updates.percentage === undefined && status.completed_slides && status.total_slides) {
          updates.percentage = Math.round((status.completed_slides.length || status.completed_slides) / status.total_slides * 100)
        }
        
        console.log('🔧 提取的进度百分比:', updates.percentage)
        
        // 🔧 根据百分比计算已完成的幻灯片数量（如果后端没有提供准确的数组）
        if (updates.percentage !== undefined && (updates.totalSlides || status.total_slides)) {
          const totalSlides = updates.totalSlides || status.total_slides
          const calculatedCompleted = Math.floor((updates.percentage / 100) * totalSlides)
          
          // 如果后端没有提供completedSlides或者是空数组，就根据百分比计算
          if (!updates.completedSlides || updates.completedSlides.length === 0) {
            updates.completedSlides = Array(calculatedCompleted).fill(0).map((_, i) => i + 1)
            console.log('🔧 根据百分比计算的已完成幻灯片:', calculatedCompleted, updates.completedSlides)
          }
        }
        
        break
      case 'completed':
        updates.message = '音频生成完成！'
        updates.percentage = 100
        break
      case 'partial_success':
        updates.message = '音频生成部分完成'
        updates.percentage = Math.round((status.completed_slides?.length || 0) / (status.total_slides || this.data.totalSlides) * 100)
        break
      case 'failed':
        updates.message = status.error || '音频生成失败'
        break
      default:
        updates.message = '状态未知'
    }
    
    this.updateAudioGenerationProgress(updates)
  },

  /**
   * 🆕 显示音频生成进度模态框
   */
  showAudioProgressModal() {
    this.setData({ showAudioProgressModal: true })
  },

  /**
   * 🆕 处理音频进度更新
   */
  onAudioProgressUpdate(e) {
    const { progress } = e.detail
    
    this.setData({
      audioGenerationProgress: {
        ...this.data.audioGenerationProgress,
        ...progress
      }
    })

    // 如果完成，刷新音频列表
    if (progress.status === 'completed') {
      this.setData({ audioGenerating: false })
      this.loadCourseData() // 使用正确的方法名
      
      wx.showToast({
        title: '音频生成完成！',
        icon: 'success'
      })
    } else if (progress.status === 'failed') {
      this.setData({ audioGenerating: false })
    }
  },

  /**
   * 🆕 处理进度模态框最小化
   */
  onAudioProgressMinimize() {
    this.setData({ showAudioProgressModal: false })
  },

  /**
   * 🆕 处理进度模态框取消
   */
  onAudioProgressCancel() {
    this.setData({ 
      showAudioProgressModal: false,
      audioGenerating: false
    })
  },

  /**
   * 🆕 处理进度重试
   */
  onAudioProgressRetry() {
    this.generateMissingAudio() // 重新开始生成
  },

  /**
   * 🆕 统一的音频生成入口方法
   */
  generateAudio() {
    this.generateMissingAudio()
  },

  /**
   * 🆕 处理进度完成
   */
  onAudioProgressComplete() {
    this.setData({ showAudioProgressModal: false })
    // 可以自动开始播放或其他操作
  },

  /**
   * 显示音频生成失败的模态框
   */
  showAudioGenerationFailedModal() {
    wx.showModal({
      title: '音频生成失败',
      content: '无法生成语音讲解，是否切换到文本模式？',
      success: (res) => {
        if (res.confirm) {
          this.setData({ 
            audioMode: false,
            mode: 'text'
          })
        }
      }
    })
  },

  /**
   * 初始化进度追踪 - 新增功能
   */
  initProgressTracking() {
    // 初始化进度追踪器
    this.initProgressTracker()
    
    // 开始学习追踪
    if (this.data.progressTracker) {
      this.data.progressTracker.startTracking(this.data.slides ? this.data.slides.length : 0)
      
      // 设置进度更新回调
      this.data.progressTracker.onProgressUpdate = (progressData) => {
        this.setData({
          'learningProgress.currentSlide': progressData.currentSlide,
          'learningProgress.progress': progressData.progress,
          'learningProgress.completionRate': progressData.completionRate,
          'learningProgress.totalTime': progressData.totalTime,
          'learningProgress.activeTime': progressData.activeTime
        })
      }
      
      console.log('进度追踪器已初始化并开始追踪')
    }
  },

  /**
   * 开始定期保存进度 - 新增功能
   */
  startProgressSaving() {
    // 每10秒保存一次进度
    setInterval(() => {
      this.saveLearningProgress()
    }, 10000)
  },

  /**
   * 音频播放控制 - 增强版
   */
  async playSlide(slideIndex, autoPlay = true) {
    if (slideIndex < 0 || slideIndex >= this.data.slides.length) {
      console.error('幻灯片索引超出范围:', slideIndex)
      return
    }

    const slide = this.data.slides[slideIndex]
    
    // 1. 切换幻灯片显示
    this.setData({ 
      currentSlideIndex: slideIndex,
      'learningProgress.currentSlide': slideIndex,
      // 🔥 重置音频播放进度为0
      audioCurrentTime: 0,
      audioDuration: 0
    })
    
    // 2. 记录幻灯片访问
    this.recordSlideAccess(slideIndex)
    
    // 3. 如果是音频模式且有音频文件
    if (this.data.audioMode && slide.audio_url) {
      this.setData({ audioSrc: slide.audio_url })
      
      if (autoPlay) {
        await this.playAudio()
      }
    }
    
    // 4. 预加载下一张音频
    if (this.data.audioMode && this.data.syncManager) {
      this.data.syncManager.preloadNextSlideAudio(this.data.currentSlideIndex)
    }
  },

  /**
   * 播放音频 - 增强版
   */
  async playAudio() {
    console.log('🎵 [播放器] 播放按钮被点击')
    console.log('🎵 [播放器] 当前音频源:', this.data.audioSrc)
    console.log('🎵 [播放器] 当前音频上下文:', this.data.audioContext)
    
    if (!this.data.audioSrc) {
      console.warn('🎵 [播放器] 当前幻灯片没有音频文件')
      if (this.data.audioMode) {
        // 音频模式下如果没有音频，尝试生成
        this.generateMissingAudio()
      }
      return
    }
    
    const audioContext = this.data.audioContext
    if (!audioContext) {
      console.warn('🎵 [播放器] 音频上下文不存在')
      return
    }
    
    try {
      // 设置音频源
      audioContext.src = this.data.audioSrc
      console.log('🎵 [播放器] 音频源已设置:', this.data.audioSrc)
      
      // 设置播放速度
      audioContext.playbackRate = this.data.playbackRate
      console.log('🎵 [播放器] 播放速度已设置:', this.data.playbackRate)
      
      // 更新同步管理器的播放状态
      if (this.data.syncManager) {
        this.data.syncManager.isPaused = false
        this.data.syncManager.isPlaying = true
        console.log('🎵 [播放器] 同步管理器状态已更新')
      }
      
      // 开始播放
      await audioContext.play()
      console.log('🎵 [播放器] play() 方法已调用')
    
      this.setData({
        isAudioPlaying: true,
        isPlaying: true,
        isPaused: false,
        // 🔥 确保音频时长在播放时更新
        audioDuration: audioContext.duration || 0
      })
      
      console.log('🎵 [播放器] 音频开始播放，状态已更新，时长:', audioContext.duration)
      
    } catch (error) {
      console.error('🎵 [播放器] 音频播放失败:', error)
      this.handleAudioError(error)
    }
  },

  /**
   * 暂停音频
   */
  pauseAudio() {
    console.log('🎵 [播放器] 暂停按钮被点击')
    console.log('🎵 [播放器] 当前音频上下文:', this.data.audioContext)
    console.log('🎵 [播放器] 当前音频源:', this.data.audioSrc)
    console.log('🎵 [播放器] 当前播放状态:', this.data.isAudioPlaying)
    
    const audioContext = this.data.audioContext
    if (audioContext) {
      try {
        audioContext.pause()
        console.log('🎵 [播放器] pause() 方法已调用')
        
        // 更新同步管理器的暂停状态
        if (this.data.syncManager) {
          this.data.syncManager.isPaused = true
          this.data.syncManager.stopSyncCheck()
          console.log('🎵 [播放器] 同步管理器已暂停')
        }
        
        this.setData({
          isAudioPlaying: false,
          isPlaying: false,
          isPaused: true
        })
        
        console.log('🎵 [播放器] 音频已暂停，状态已更新')
      } catch (error) {
        console.error('🎵 [播放器] 暂停音频时出错:', error)
      }
    } else {
      console.warn('🎵 [播放器] 音频上下文不存在')
    }
  },

  /**
   * 音频播放结束处理 - 简化版本
   */
  onAudioEnded() {
    console.log('🎵 [播放器] 音频播放结束，当前幻灯片:', this.data.currentSlideIndex)
    
    // 标记当前幻灯片为已完成
    this.markSlideCompleted(this.data.currentSlideIndex)
    
    // 🔥 重置音频播放进度为0
    this.setData({
      audioCurrentTime: 0,
      isAudioPlaying: false
    })
    
    // 检查是否被用户暂停
    if (this.data.isPaused) {
      console.log('🎵 [播放器] 检测到用户暂停，不自动播放下一张')
      return
    }
    
    // 🔥 简化逻辑：直接基于设置决定是否自动播放下一张
    if (this.data.audioSettings.autoPlayNext && this.hasNextSlide()) {
      console.log('🎵 [播放器] 自动播放下一张幻灯片')
      setTimeout(() => {
        this.nextSlide()
      }, 1000) // 1秒延迟，确保UI状态更新
    } else if (!this.hasNextSlide()) {
      console.log('🎵 [播放器] 已是最后一张，完成课程')
      this.completeCourse()
    } else {
      console.log('🎵 [播放器] 自动播放已禁用，等待用户手动操作')
    }
  },

  /**
   * 音频时间更新
   */
  onTimeUpdate(e) {
    const currentTime = e.detail.currentTime
    const duration = e.detail.duration
    
    this.setData({
      audioCurrentTime: currentTime,
      audioDuration: duration
    })
    
    console.log(`🎵 [播放器] 音频进度更新: ${currentTime.toFixed(1)}s / ${duration.toFixed(1)}s`)
  },

  /**
   * 上一张幻灯片
   */
  prevSlide() {
    if (this.data.currentSlideIndex > 0) {
      const newIndex = this.data.currentSlideIndex - 1
      console.log(`🎯 [切换] 切换到上一张幻灯片: ${newIndex}`)
      
      this.setData({
        currentSlideIndex: newIndex,
        // 🔥 重置音频播放进度为0
        audioCurrentTime: 0,
        audioDuration: 0
      })
      this.recordSlideAccess(newIndex) // 🔥 添加幻灯片访问记录
      this.setCurrentSlideAudio()
      this.saveLearningProgress()
      
      // 🔥 新增：应用自适应字体
      setTimeout(() => {
        this.renderAdaptiveContent()
      }, 100)
    }
  },

  /**
   * 下一张幻灯片
   */
  nextSlide() {
    const currentIndex = this.data.currentSlideIndex
    const totalSlides = this.data.totalSlides
    
    console.log(`🎯 [切换] 当前: ${currentIndex}, 总数: ${totalSlides}`)
    
    if (currentIndex < totalSlides - 1) {
      const newIndex = currentIndex + 1
      console.log(`🎯 [切换] 从幻灯片 ${currentIndex} 切换到 ${newIndex}`)
      
      this.setData({
        currentSlideIndex: newIndex,
        // 🔥 重置音频播放进度为0
        audioCurrentTime: 0,
        audioDuration: 0
      })
      this.recordSlideAccess(newIndex) // 🔥 添加幻灯片访问记录
      this.setCurrentSlideAudio()
      
      // 🔥 新增：应用自适应字体
      setTimeout(() => {
        this.renderAdaptiveContent()
      }, 100)
      
      // 🔥 在音频模式下，切换后自动播放新的音频（仅在非暂停状态下）
      if (this.data.audioMode && this.data.audioSettings.autoPlayNext && !this.data.isPaused) {
        setTimeout(() => {
          console.log(`🎵 [切换] 开始播放第${newIndex}张幻灯片音频`)
          this.playAudio()
        }, 500) // 延迟确保音频源设置完成
      } else if (this.data.audioMode && this.data.isPaused) {
        console.log(`🎵 [切换] 检测到暂停状态，不自动播放第${newIndex}张幻灯片音频`)
      }
      
      this.saveLearningProgress()
    } else {
      console.log('🎯 [切换] 已是最后一张，完成课程')
      this.completeCourse()
    }
  },

  /**
   * 跳转到指定幻灯片
   */
  goToSlide(e) {
    const slideIndex = parseInt(e.currentTarget.dataset.index)
    console.log(`🎯 [切换] 跳转到指定幻灯片: ${slideIndex}`)
    
    this.setData({
      currentSlideIndex: slideIndex,
      // 🔥 重置音频播放进度为0
      audioCurrentTime: 0,
      audioDuration: 0
    })
    this.recordSlideAccess(slideIndex) // 🔥 添加幻灯片访问记录
    this.setCurrentSlideAudio()
    this.saveLearningProgress()
    
    // 🔥 新增：应用自适应字体
    setTimeout(() => {
      this.renderAdaptiveContent()
    }, 100)
  },

  /**
   * 保存学习进度 - 增强版
   */
  async saveLearningProgress() {
    const progressData = this.data.learningProgress
    
    // 1. 保存到本地存储
    try {
      wx.setStorageSync(`course_progress_${this.data.courseId}`, progressData)
    } catch (error) {
      console.error('保存本地进度失败:', error)
    }
    
    // 2. 保存到服务器
    if (this.data.learningRecordId) {
      try {
        await this.saveLearningProgressToServer(progressData)
      } catch (error) {
        console.error('保存服务器进度失败:', error)
      }
    }
  },

  /**
   * 保存进度到服务器 - 新增功能
   */
  saveLearningProgressToServer(progressData) {
    return new Promise((resolve, reject) => {
      wx.request({
        url: require('../../utils/config.js').getBaseURLSync() + '/learning/progress',
        method: 'POST',
        header: {
          'Authorization': `Bearer ${this.getValidToken()}`,
          'Content-Type': 'application/json'
        },
        data: {
          learning_record_id: this.data.learningRecordId,
          course_id: this.data.courseId,
          current_slide: progressData.currentSlide,
          total_time: Math.floor(progressData.totalTime / 1000), // 转换为秒
          completion_rate: progressData.completionRate/10000, // 转换为0-100范围
          slides_progress: progressData.slides_progress || []
        },
        success: (res) => {
          if (res.statusCode === 200 && res.data.code === 200) {
            resolve(res.data)
          } else {
            reject(new Error(res.data.message || '保存进度失败'))
          }
        },
        fail: reject
      })
    })
  },

  /**
   * 音频错误处理 - 新增功能
   */
  handleAudioError(error) {
    console.error('音频播放错误:', error)
    
    // 检查是否是格式兼容性问题
    if (error.errMsg && (error.errMsg.includes('seek audio fail') || error.errMsg.includes('pause audio fail'))) {
      console.log('🔧 音频格式兼容性问题，尝试继续播放')
      // 对于格式问题，不显示错误弹窗，让用户继续使用
      return
    }
    
    wx.showModal({
      title: '音频播放失败',
      content: '当前音频无法播放，是否跳过到下一张？',
      success: (res) => {
        if (res.confirm) {
          this.nextSlide()
        }
      }
    })
  },

  /**
   * 检查是否有下一张幻灯片
   */
  hasNextSlide() {
    return this.data.currentSlideIndex < this.data.totalSlides - 1
  },

  /**
   * 检查是否有上一张幻灯片  
   */
  hasPrevSlide() {
    return this.data.currentSlideIndex > 0
  },

  /**
   * 记录幻灯片访问 - 新增功能
   */
  recordSlideAccess(slideIndex) {
    console.log(`📝 [访问] 记录幻灯片 ${slideIndex} 访问`)
    
    const now = Date.now()
    const progressData = this.data.learningProgress
    
    if (!progressData.slides_progress) {
      progressData.slides_progress = []
    }
    
    if (!progressData.slides_progress[slideIndex]) {
      progressData.slides_progress[slideIndex] = {
        slide_index: slideIndex,
        first_access: now,
        total_time: 0,
        audio_progress: 0,
        completed: false
      }
    }
    
    progressData.slides_progress[slideIndex].last_access = now
    progressData.currentSlide = slideIndex
    
    this.setData({ learningProgress: progressData })
    
    // 同步到进度追踪器
    if (this.data.progressTracker) {
      const slide = this.data.slides[slideIndex]
      console.log(`📝 [同步] 同步到进度追踪器: slide ${slideIndex}`)
      this.data.progressTracker.updateCurrentSlide(slideIndex, {
        title: slide ? slide.title : '',
        content: slide ? slide.content : ''
      })
    }
  },

  /**
   * 标记幻灯片完成 - 新增功能
   */
  markSlideCompleted(slideIndex) {
    const progressData = this.data.learningProgress
    
    if (progressData.slides_progress && progressData.slides_progress[slideIndex]) {
      progressData.slides_progress[slideIndex].completed = true
      
      // 更新完成的幻灯片列表
      if (!progressData.completedSlides.includes(slideIndex)) {
        progressData.completedSlides.push(slideIndex)
      }
      
      // 🔥 改进：基于当前播放位置和完成数计算智能完成率
      const positionBasedRate = Math.min(((slideIndex + 1) / this.data.totalSlides) * 100, 100)
      const completionBasedRate = (progressData.completedSlides.length / this.data.totalSlides) * 100
      
      // 使用较大值作为完成率，确保进度不会倒退
      progressData.completionRate = Math.max(positionBasedRate, completionBasedRate)
      
      this.setData({ learningProgress: progressData })
      
      // 同步到进度追踪器
      if (this.data.progressTracker) {
        this.data.progressTracker.completeSlide(slideIndex)
      }
      
      console.log(`📊 [进度] 幻灯片 ${slideIndex} 已完成，位置进度: ${positionBasedRate.toFixed(1)}%, 完成进度: ${completionBasedRate.toFixed(1)}%, 最终完成率: ${progressData.completionRate.toFixed(1)}%`)
    }
  },

  /**
   * 更新学习进度 - 新增功能
   */
  updateLearningProgress() {
    const now = Date.now()
    const progressData = this.data.learningProgress
    const slideIndex = this.data.currentSlideIndex
    
    // 更新总学习时间
    if (this.data.startTime) {
      progressData.totalTime = now - this.data.startTime
    }
    
    // 更新当前幻灯片的音频进度
    if (progressData.slides_progress && progressData.slides_progress[slideIndex]) {
      const audioProgress = this.data.audioDuration > 0 
        ? this.data.audioCurrentTime / this.data.audioDuration 
        : 0
      
      progressData.slides_progress[slideIndex].audio_progress = audioProgress
      
      // 如果音频播放超过90%，标记为完成
      if (audioProgress >= 0.9 && !progressData.slides_progress[slideIndex].completed) {
        this.markSlideCompleted(slideIndex)
    }
    }
    
    this.setData({ learningProgress: progressData })
  },

  /**
   * 完成课程
   */
  async completeCourse() {
    try {
      // 🔥 自动完成所有幻灯片和最后一张
      const lastSlideIndex = this.data.totalSlides - 1
      if (!this.data.learningProgress.completedSlides.includes(lastSlideIndex)) {
        this.markSlideCompleted(lastSlideIndex)
      }
      
      // 🔥 确保所有幻灯片都标记为完成
      if (this.data.progressTracker) {
        this.data.progressTracker.completeAllSlides()
      }
      
      // 🔥 停止进度追踪
      if (this.data.progressTracker) {
        this.data.progressTracker.stopTracking()
      }
      
      await courseAPI.completeCourse(this.data.courseId)
      
      wx.showModal({
        title: '恭喜',
        content: '您已完成本课程的学习！',
        showCancel: false,
        success: () => {
          wx.navigateBack()
        }
      })
    } catch (error) {
      console.error('完成课程失败:', error)
    }
  },

  /**
   * 切换幻灯片列表显示
   */
  toggleSlideList() {
    const newState = !this.data.showSlideList
    console.log('切换幻灯片列表显示:', newState)
    this.setData({
      showSlideList: newState
    })
  },

  /**
   * 切换笔记显示
   */
  toggleNotes() {
    const newState = !this.data.showNotes
    const currentSlide = this.data.slides[this.data.currentSlideIndex]
    console.log('切换备注显示:', newState)
    console.log('当前幻灯片备注:', currentSlide ? currentSlide.speaker_notes : '无当前幻灯片')
    this.setData({
      showNotes: newState
    })
  },

  /**
   * 切换进度显示
   */
  toggleProgress() {
    const newState = !this.data.showProgress
    console.log('切换进度显示:', newState)
    this.setData({
      showProgress: newState
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
    return `${this.data.currentSlideIndex + 1} / ${this.data.totalSlides}`
  },

  /**
   * 页面分享
   */
  onShareAppMessage() {
    const { courseInfo } = this.data
    if (!courseInfo) return {}
    
    return {
      title: courseInfo.title,
      desc: courseInfo.description,
      path: `/pages/course/detail/detail?id=${courseInfo.id}`
    }
  },

  /**
   * 切换音频设置面板
   */
  toggleAudioSettings() {
    const newState = !this.data.showAudioSettings
    console.log('切换音频设置显示:', newState)
    this.setData({
      showAudioSettings: newState
    })
  },

  /**
   * 切换下拉菜单显示
   */
  toggleDropdownMenu() {
    const newState = !this.data.showDropdownMenu
    console.log('切换下拉菜单显示:', newState)
    this.setData({
      showDropdownMenu: newState
    })
  },

  /**
   * 关闭下拉菜单
   */
  closeDropdownMenu() {
    console.log('关闭下拉菜单被调用')
    this.setData({
      showDropdownMenu: false
    })
  },

  /**
   * 测试方法 - 直接切换目录
   */
  testToggleSlideList() {
    console.log('测试方法：直接切换目录')
    this.toggleSlideList()
  },

  /**
   * 防止事件冒泡
   */
  preventBubble(e) {
    // 什么都不做，只是防止事件冒泡
    console.log('防止事件冒泡')
  },

  /**
   * 处理菜单操作
   */
  handleMenuAction(e) {
    console.log('handleMenuAction被调用，事件对象:', e)
    
    if (!e || !e.currentTarget) {
      console.error('事件对象或currentTarget为空')
      return
    }
    
    const action = e.currentTarget.dataset.action
    console.log('菜单操作被触发:', action, '完整dataset:', e.currentTarget.dataset)
    
    if (!action) {
      console.error('action为空，无法执行操作')
      return
    }
    
    // 关闭下拉菜单
    console.log('关闭下拉菜单')
    this.setData({
      showDropdownMenu: false
    })
    
    // 执行对应操作
    switch (action) {
      case 'catalog':
        console.log('执行目录切换')
        this.toggleSlideList()
        break
      case 'notes':
        console.log('执行备注切换')
        this.toggleNotes()
        break
      case 'progress':
        console.log('执行进度切换')
        this.toggleProgress()
        break
      case 'generate-audio':
        console.log('🎙️ 执行音频生成')
        this.generateAudio()
        break
      case 'settings':
        console.log('执行设置切换')
        this.toggleAudioSettings()
        break
      default:
        console.log('未知的菜单操作:', action)
    }
  },

  /**
   * 播放速度调整
   */
  onPlaybackRateChange(e) {
    const rate = parseFloat(e.currentTarget.dataset.value)
    this.setData({ playbackRate: rate })
    
    if (this.data.audioContext && this.data.audioContext.playbackRate) {
      this.data.audioContext.playbackRate = rate
    }
    
    console.log('播放速度已调整为:', rate)
  },

  /**
   * 自动播放切换
   */
  onAutoPlayChange(e) {
    const autoPlay = e.detail.value
    this.setData({ 'audioSettings.autoPlayNext': autoPlay })
    console.log('自动播放设置:', autoPlay)
  },

  /**
   * 音画同步自动播放切换
   */
  onSyncAutoPlayChange(e) {
    const autoPlay = e.detail.value
    this.setData({ 
      'syncConfig.autoPlay': autoPlay,
      'audioSettings.autoPlayNext': autoPlay
    })
    
    // 同步更新AudioSyncManager的设置
    if (this.data.syncManager) {
      this.data.syncManager.updateOptions({ autoPlayNext: autoPlay })
    }
    
    console.log('音画同步自动播放设置:', autoPlay)
  },

  /**
   * 语音类型切换
   */
  onVoiceTypeChange(e) {
    	const voiceTypes = ['zhixiaobai', 'zhimao', 'xiaoyun', 'xiaogang', 'xiaomei', 'xiaofeng']
    const voiceType = voiceTypes[e.detail.value]
    this.setData({ 'audioSettings.voiceType': voiceType })
    console.log('语音类型已切换为:', voiceType)
  },

  /**
   * 音量调整
   */
  onVolumeChange(e) {
    const volume = parseInt(e.detail.value)
    this.setData({ 'audioSettings.volume': volume })
    
    if (this.data.audioContext) {
      this.data.audioContext.volume = volume / 100
    }
    
    console.log('音量已调整为:', volume)
  },

  /**
   * 重置音频设置
   */
  resetAudioSettings() {
    this.setData({
      playbackRate: 1.0,
      'audioSettings.volume': 80,
      		'audioSettings.voiceType': 'zhixiaobai'
    })
    
    if (this.data.audioContext) {
      this.data.audioContext.playbackRate = 1.0
      this.data.audioContext.volume = 0.8
    }
    
    wx.showToast({
      title: '设置已重置',
      icon: 'success'
    })
  },

  /**
   * 设置当前幻灯片音频
   */
  setCurrentSlideAudio() {
    const currentSlide = this.data.slides[this.data.currentSlideIndex]
    if (!currentSlide) {
      console.log('当前幻灯片不存在')
      return
    }

    console.log('设置当前幻灯片音频:', currentSlide.audio_url)
    
    if (currentSlide.audio_url) {
      this.setData({
        audioSrc: currentSlide.audio_url,
        currentAudioUrl: currentSlide.audio_url,
        hasAudio: true,
        // 🔥 重置音频播放进度为0
        audioCurrentTime: 0,
        audioDuration: 0
      })
      
      console.log('🎵 [音频] 设置新音频源:', currentSlide.audio_url)
      
      // 🔴 移除自动播放逻辑，防止重复播放
      // 音频播放应该由onAudioEnded()中的nextSlide()触发
      // 或者用户手动点击播放按钮
      /*
      if (this.data.audioMode && this.data.syncConfig.autoPlay) {
        setTimeout(() => {
          this.playAudio()
        }, 500) // 延迟500ms确保音频加载完成
      }
      */
    } else {
      this.setData({
        audioSrc: '',
        currentAudioUrl: '',
        hasAudio: false,
        // 🔥 重置音频播放进度为0
        audioCurrentTime: 0,
        audioDuration: 0
      })
      console.log('🎵 [音频] 当前幻灯片没有音频文件')
    }
  },

  /**
   * 获取有效的访问token
   */
  getValidToken() {
    let token = wx.getStorageSync('access_token')
    if (!token) {
      // 如果没有access_token，尝试使用旧的token格式
      token = wx.getStorageSync('token')
      if (!token) {
        // 使用测试token作为后备
        token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo0LCJvcGVuaWQiOiJtb2NrX29wZW5pZF8xMjM0NSIsIm5pY2tuYW1lIjoi5rWL6K-V55So5oi3IiwidmlwX2xldmVsIjoxLCJpc3MiOiJhaS1jbGFzc3Jvb20iLCJzdWIiOiJ1c2VyOjQiLCJleHAiOjE3NTMzMjg2MTIsIm5iZiI6MTc1MzI0MjIxMiwiaWF0IjoxNzUzMjQyMjEyfQ.xAd8JOccLXdj4mGCprUlc4YSG6XVIygPzzMJb26oRRk'
        wx.setStorageSync('access_token', token)
      }
    }
    return token
  },

  /**
   * 格式化时间显示
   */
  formatTime(seconds) {
    if (!seconds || isNaN(seconds)) return '00:00'
    
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = Math.floor(seconds % 60)
    
    return `${minutes.toString().padStart(2, '0')}:${remainingSeconds.toString().padStart(2, '0')}`
  },

  /**
   * 格式化进度显示
   */
  formatProgress() {
    return `${this.data.currentSlideIndex + 1}/${this.data.totalSlides}`
  },

  /**
   * 重置音频播放进度
   */
  resetAudioProgress() {
    console.log('🎵 [播放器] 重置音频播放进度为0')
    this.setData({
      audioCurrentTime: 0,
      audioDuration: 0
    })
  },

  /**
   * 初始化字体适配器
   */
  initFontAdapter() {
    const slides = this.data.slides
    if (!slides || slides.length === 0) {
      console.log('没有幻灯片数据，跳过字体适配初始化')
      return
    }

    // 创建字体样式器实例
    this.slideStyler = new SlideStyler()
    this.contentAnalyzer = new ContentAnalyzer()
    
    console.log('🔧 字体适配器已初始化')
    
    // 立即分析当前幻灯片内容
    this.renderAdaptiveContent()
  },

  /**
   * 内容自适应渲染
   */
  renderAdaptiveContent() {
    const currentSlide = this.data.slides[this.data.currentSlideIndex]
    if (!currentSlide) {
      console.log('当前没有幻灯片内容')
      return
    }

    // 分析内容
    const content = this.formatSlideContent(currentSlide)
    const analysis = this.contentAnalyzer.analyzeContent(content)
    
    console.log(`📊 [自适应] 幻灯片${this.data.currentSlideIndex + 1}: ${analysis.charCount}字符, ${analysis.contentClass}`)
    
    // 更新数据
    this.setData({
      currentSlideCharCount: analysis.charCount,
      contentAnalysis: analysis
    })
    
    // 应用自适应样式
    this.applyAdaptiveStyles(analysis)
  },

  /**
   * 应用自适应样式
   */
  applyAdaptiveStyles(analysis) {
    try {
      // 获取当前幻灯片内容
      const currentSlide = this.data.slides[this.data.currentSlideIndex]
      if (!currentSlide) {
        console.log('当前没有幻灯片内容')
        return
      }
      
      // 格式化内容
      const content = this.formatSlideContent(currentSlide)
      
      // 使用SlideStyler获取样式配置
      const styleConfig = this.slideStyler.applyStyles(content)
      
      // 使用数据绑定方式应用样式类
      this.setData({
        currentContentClass: styleConfig.className,
        currentFontSize: styleConfig.fontSize,
        currentLineHeight: styleConfig.lineHeight,
        currentSpacing: styleConfig.spacing,
        currentProcessedContent: styleConfig.processedContent,
        cssVariables: styleConfig.cssVariables
      })
      
      console.log(`🎨 [样式] 应用样式类: ${styleConfig.className}`)
    } catch (error) {
      console.error('应用自适应样式失败:', error)
    }
  },

  /**
   * 格式化幻灯片内容为字符串
   */
  formatSlideContent(slide) {
    let content = ''
    
    // 添加标题
    if (slide.title) {
      content += slide.title + '\n'
    }
    
    // 添加主要内容
    if (slide.content) {
      if (Array.isArray(slide.content)) {
        content += slide.content.join('\n')
      } else {
        content += slide.content
      }
    }
    
    // 添加要点列表
    if (slide.bullet_points && Array.isArray(slide.bullet_points)) {
      content += '\n' + slide.bullet_points.join('\n')
    }
    
    return content.trim()
  },

  /**
   * 设置当前幻灯片内容 - 增强自适应支持
   */
  setCurrentSlideContent() {
    const currentSlide = this.data.slides[this.data.currentSlideIndex]
    if (!currentSlide) return
    
    // 渲染内容
    this.setData({
      currentSlideTitle: currentSlide.title,
      currentSlideContent: currentSlide.content
    })
    
    // 应用自适应样式
    setTimeout(() => {
      this.renderAdaptiveContent()
    }, 100)
  },

  /**
   * 预览音色 - 新增方法
   */
  previewVoice() {
    console.log('开始预览音色:', this.data.audioSettings.voiceType)
    
    // 获取当前幻灯片内容
    const currentSlide = this.data.slides[this.data.currentSlideIndex]
    if (!currentSlide) {
      wx.showToast({
        title: '没有可预览的内容',
        icon: 'none'
      })
      return
    }
    
    // 格式化内容用于预览
    const content = this.formatSlideContent(currentSlide)
    if (!content || content.trim().length === 0) {
      wx.showToast({
        title: '内容为空，无法预览',
        icon: 'none'
      })
      return
    }
    
    // 显示加载提示
    wx.showLoading({
      title: '生成预览音频...'
    })
    
    // 调用后端API生成预览音频
    const token = this.getValidToken()
    if (!token) {
      wx.hideLoading()
      wx.showToast({
        title: '请先登录',
        icon: 'none'
      })
      return
    }
    
    // 构建预览请求
    const previewData = {
      content: content.substring(0, 200), // 限制预览内容长度
      voice_type: this.data.audioSettings.voiceType,
      volume: this.data.audioSettings.volume / 100
    }
    
    // 调用预览API
    const { getBaseURLSync } = require('../../utils/config')
    const baseURL = getBaseURLSync()
    
    wx.request({
      url: `${baseURL}/audio/preview`,
      method: 'POST',
      header: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: previewData,
      success: (res) => {
        wx.hideLoading()
        
        if (res.statusCode === 200 && res.data.success) {
          // 播放预览音频
          this.playPreviewAudio(res.data.audio_url)
        } else {
          wx.showToast({
            title: res.data.message || '预览生成失败',
            icon: 'none'
          })
        }
      },
      fail: (error) => {
        wx.hideLoading()
        console.error('预览请求失败:', error)
        wx.showToast({
          title: '网络错误，请重试',
          icon: 'none'
        })
      }
    })
  },

  /**
   * 播放预览音频
   */
  playPreviewAudio(audioUrl) {
    if (!audioUrl) {
      wx.showToast({
        title: '音频URL无效',
        icon: 'none'
      })
      return
    }
    
    // 创建预览音频上下文
    const previewAudioContext = wx.createInnerAudioContext()
    previewAudioContext.src = audioUrl
    previewAudioContext.volume = this.data.audioSettings.volume / 100
    
    // 监听播放事件
    previewAudioContext.onPlay(() => {
      console.log('预览音频开始播放')
      wx.showToast({
        title: '正在播放预览',
        icon: 'none',
        duration: 2000
      })
    })
    
    previewAudioContext.onEnded(() => {
      console.log('预览音频播放结束')
      previewAudioContext.destroy()
    })
    
    previewAudioContext.onError((error) => {
      console.error('预览音频播放失败:', error)
      wx.showToast({
        title: '预览播放失败',
        icon: 'none'
      })
      previewAudioContext.destroy()
    })
    
    // 开始播放
    previewAudioContext.play()
  }
})