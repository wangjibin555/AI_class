// components/audio/audio-progress.js
// 音频生成进度组件

Component({
  /**
   * 组件的属性列表
   */
  properties: {
    visible: {
      type: Boolean,
      value: false
    },
    courseId: {
      type: Number,
      value: 0
    },
    progress: {
      type: Object,
      value: {
        percentage: 0,
        status: 'pending',
        totalSlides: 0,
        completedSlides: 0,
        currentSlide: '',
        errorMessage: '',
        slideProgress: 0,
        slideProgressText: ''
      }
    },
    allowMinimize: {
      type: Boolean,
      value: true
    },
    allowCancel: {
      type: Boolean,
      value: true
    }
  },

  /**
   * 组件的初始数据
   */
  data: {
    showDetails: false,
    estimatedTime: '',
    generationSpeed: '',
    audioQuality: '高质量',
    voiceEngine: 'AI智能语音',
    progressSteps: [
      {
        id: 'analyze',
        title: '内容分析',
        description: '分析幻灯片内容和结构',
        status: 'pending'
      },
      {
        id: 'generate',
        title: '语音合成',
        description: '使用AI引擎生成自然语音',
        status: 'pending'
      },
      {
        id: 'optimize',
        title: '音频优化',
        description: '优化音频质量和时长',
        status: 'pending'
      },
      {
        id: 'sync',
        title: '同步处理',
        description: '与幻灯片内容同步',
        status: 'pending'
      }
    ],
    pollTimer: null,
    startTime: null,
    lastUpdateTime: null
  },

  /**
   * 组件生命周期
   */
  lifetimes: {
    attached() {
      this.setData({
        startTime: Date.now()
      })
    },

    detached() {
      this.stopPolling()
    }
  },

  /**
   * 监听属性变化
   */
  observers: {
    'visible': function(visible) {
      if (visible) {
        // 不再自动开始轮询，由父组件传递进度数据
        this.updateProgressSteps()
      } else {
        this.stopPolling()
      }
    },

    'progress': function(progress) {
      // 完全依赖外部传入的progress，不做任何内部修改
      if (progress) {
        console.log('🔄 收到父组件进度数据:', progress)
        console.log('🔍 详细数据检查:', {
          percentage: progress.percentage,
          completedSlides: progress.completedSlides,
          totalSlides: progress.totalSlides,
          completedSlidesType: typeof progress.completedSlides,
          completedSlidesLength: progress.completedSlides?.length
        })
        
        // 只更新辅助数据，不修改progress本身
        this.updateProgressSteps()
        this.updateEstimatedTime()
        this.updateGenerationSpeed()
      }
    }
  },

  /**
   * 组件的方法列表
   */
  methods: {
    /**
     * 开始轮询进度
     */
    startPolling() {
      this.stopPolling() // 确保清除之前的定时器
      
      const poll = () => {
        this.fetchProgress()
      }
      
      // 立即执行一次
      poll()
      
      // 设置定时轮询
      const timer = setInterval(poll, 2000) // 每2秒轮询一次
      this.setData({ pollTimer: timer })
    },

    /**
     * 停止轮询
     */
    stopPolling() {
      const { pollTimer } = this.data
      if (pollTimer) {
        clearInterval(pollTimer)
        this.setData({ pollTimer: null })
      }
    },

    /**
     * 获取进度信息
     */
    fetchProgress() {
      const { courseId } = this.properties
      if (!courseId) return

      const app = getApp()
      const { getToken } = require('../../utils/auth.js')
      const token = getToken()
      
      console.log('🔐 检查认证信息:', {
        token: token ? '已设置' : '未设置',
        baseURL: app.globalData.baseURL,
        courseId: courseId
      })
      
      if (!token) {
        console.error('❌ 未找到认证token')
        return
      }
      
      wx.request({
        url: `${app.globalData.baseURL}/audio/status/${courseId}`,
        method: 'GET',
        header: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        success: (response) => {
          try {
            if (response.data && response.data.code === 200) {
              const progressData = response.data.data.progress
              if (progressData) {
                this.updateProgress(progressData)
              }
            }
          } catch (error) {
            console.error('解析音频生成进度失败:', error)
          }
        },
        fail: (error) => {
          console.error('获取音频生成进度失败:', error)
        }
      })
    },

    /**
     * 更新进度数据
     */
    updateProgress(progressData) {
      const progress = {
        percentage: progressData.progress || 0,
        status: progressData.status || 'processing',
        totalSlides: progressData.total_slides || 0,
        completedSlides: progressData.completed_slides || 0,
        currentSlide: progressData.current_slide || '',
        errorMessage: progressData.error_message || '',
        slideProgress: this.calculateSlideProgress(progressData),
        slideProgressText: this.getSlideProgressText(progressData)
      }

      this.setData({ 
        progress,
        lastUpdateTime: Date.now()
      })

      // 触发进度更新事件
      this.triggerEvent('progressUpdate', { progress })

      // 如果完成，停止轮询
      if (progress.status === 'completed' || progress.status === 'failed') {
        this.stopPolling()
      }
    },

    /**
     * 计算当前幻灯片进度
     */
    calculateSlideProgress(progressData) {
      if (!progressData.current_slide_progress) return 0
      
      // 根据不同的处理阶段计算进度
      const stage = progressData.current_stage || 'analyzing'
      const stageProgress = {
        'analyzing': 25,
        'generating': 75,
        'optimizing': 90,
        'completed': 100
      }
      
      return stageProgress[stage] || 0
    },

    /**
     * 获取幻灯片进度文本
     */
    getSlideProgressText(progressData) {
      const stage = progressData.current_stage || 'analyzing'
      const stageTexts = {
        'analyzing': '正在分析内容...',
        'generating': '正在生成语音...',
        'optimizing': '正在优化音频...',
        'completed': '处理完成'
      }
      
      return stageTexts[stage] || '正在处理...'
    },

    /**
     * 更新进度步骤状态
     */
    updateProgressSteps() {
      const progress = this.properties.progress || {}
      const steps = [...this.data.progressSteps]
      
      // 根据进度更新步骤状态
      if (progress.percentage >= 25) {
        steps[0].status = 'completed'
        steps[0].time = '已完成'
      }
      if (progress.percentage >= 50) {
        steps[1].status = 'completed'
        steps[1].time = '已完成'
      } else if (progress.percentage >= 25) {
        steps[1].status = 'processing'
      }
      if (progress.percentage >= 75) {
        steps[2].status = 'completed'
        steps[2].time = '已完成'
      } else if (progress.percentage >= 50) {
        steps[2].status = 'processing'
      }
      if (progress.percentage >= 100) {
        steps[3].status = 'completed'
        steps[3].time = '已完成'
      } else if (progress.percentage >= 75) {
        steps[3].status = 'processing'
      }

      // 处理错误状态
      if (progress.status === 'failed') {
        const currentStepIndex = Math.floor(progress.percentage / 25)
        if (currentStepIndex < steps.length) {
          steps[currentStepIndex].status = 'failed'
        }
      }

      this.setData({ progressSteps: steps })
    },

    /**
     * 更新预估时间
     */
    updateEstimatedTime() {
      const progress = this.properties.progress || {}
      const { startTime, lastUpdateTime } = this.data
      
      const completedCount = Array.isArray(progress.completedSlides) ? 
        progress.completedSlides.length : (progress.completedSlides || 0)
      
      if (!startTime || completedCount === 0) {
        this.setData({ estimatedTime: '计算中...' })
        return
      }

      const elapsed = (lastUpdateTime || Date.now()) - startTime
      const avgTimePerSlide = elapsed / completedCount
      const remainingSlides = progress.totalSlides - completedCount
      const estimatedMs = remainingSlides * avgTimePerSlide
      
      console.log('🔧 剩余时间计算:', {
        elapsed: elapsed,
        completedCount: completedCount,
        totalSlides: progress.totalSlides,
        avgTimePerSlide: avgTimePerSlide,
        remainingSlides: remainingSlides,
        estimatedMs: estimatedMs
      })

      if (estimatedMs <= 0) {
        this.setData({ estimatedTime: '即将完成' })
        return
      }

      const minutes = Math.floor(estimatedMs / 60000)
      const seconds = Math.floor((estimatedMs % 60000) / 1000)

      let timeText = ''
      if (minutes > 0) {
        timeText = `${minutes}分${seconds}秒`
      } else {
        timeText = `${seconds}秒`
      }

      this.setData({ estimatedTime: timeText })
    },

    /**
     * 更新生成速度
     */
    updateGenerationSpeed() {
      const progress = this.properties.progress || {}
      const { startTime, lastUpdateTime } = this.data
      
      const completedCount = Array.isArray(progress.completedSlides) ? 
        progress.completedSlides.length : (progress.completedSlides || 0)
      
      if (!startTime || completedCount === 0) {
        this.setData({ generationSpeed: '计算中...' })
        return
      }

      const elapsed = (lastUpdateTime || Date.now()) - startTime
      const speed = completedCount / (elapsed / 1000) // 每秒完成的幻灯片数
      
      if (speed >= 1) {
        this.setData({ generationSpeed: `${speed.toFixed(1)} 张/秒` })
      } else {
        const timePerSlide = elapsed / completedCount / 1000
        this.setData({ generationSpeed: `${timePerSlide.toFixed(1)} 秒/张` })
      }
    },

    /**
     * 切换详情显示
     */
    onToggleDetails() {
      console.log('🔘 点击查看详情按钮')
      this.setData({
        showDetails: !this.data.showDetails
      })
    },

    /**
     * 最小化进度窗口
     */
    onMinimize() {
      console.log('🔘 点击最小化按钮')
      this.triggerEvent('minimize')
    },

    /**
     * 取消生成
     */
    onCancel() {
      console.log('🔘 点击取消生成按钮')
      wx.showModal({
        title: '确认取消',
        content: '确定要取消音频生成吗？已生成的音频将保留。',
        success: (res) => {
          if (res.confirm) {
            this.cancelGeneration()
          }
        }
      })
    },

    /**
     * 执行取消生成
     */
    cancelGeneration() {
      const { courseId } = this.properties
      
      const app = getApp()
      const { getToken } = require('../../utils/auth.js')
      const token = getToken()
      
      if (!token) {
        console.error('❌ 未找到认证token')
        wx.showToast({
          title: '认证失败',
          icon: 'error'
        })
        return
      }
      
      wx.request({
        url: `${app.globalData.baseURL}/audio/cancel/${courseId}`,
        method: 'POST',
        header: {
          'Authorization': `Bearer ${token}`
        },
        success: (response) => {
          try {
            if (response.data && response.data.code === 200) {
              this.triggerEvent('cancel')
              wx.showToast({
                title: '已取消生成',
                icon: 'success'
              })
            }
          } catch (error) {
            console.error('解析取消响应失败:', error)
            wx.showToast({
              title: '取消失败',
              icon: 'error'
            })
          }
        },
        fail: (error) => {
          console.error('取消音频生成失败:', error)
          wx.showToast({
            title: '取消失败',
            icon: 'error'
          })
        }
      })
    },

    /**
     * 重试生成
     */
    onRetry() {
      this.triggerEvent('retry')
    },

    /**
     * 完成生成
     */
    onComplete() {
      this.triggerEvent('complete')
    },

    /**
     * 遮罩点击
     */
    onMaskTap() {
      // 可以选择是否允许点击遮罩关闭
      // this.triggerEvent('close')
    }
  }
})