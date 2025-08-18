// components/audio/audio-controller.js
// 增强音频控制器组件

Component({
  /**
   * 组件的属性列表
   */
  properties: {
    courseData: {
      type: Object,
      value: null
    },
    audioMode: {
      type: Boolean,
      value: false
    },
    currentSlideIndex: {
      type: Number,
      value: 0
    },
    autoPlay: {
      type: Boolean,
      value: false
    },
    courseId: {
      type: Number,
      value: 0
    },
    audioGenerating: {
      type: Boolean,
      value: false
    },
    audioGenerationProgress: {
      type: Object,
      value: null
    },
    showProgressModal: {
      type: Boolean,
      value: false
    }
  },

  /**
   * 组件的初始数据
   */
  data: {
    audioContext: null,
    isPlaying: false,
    currentTime: 0,
    duration: 0,
    playbackRate: 1.0,
    volume: 80,
    loading: false,
    error: null,
    
    // 音频生成进度
    audioProgress: null,
    generationInProgress: false,
    
    // 同步配置
    syncConfig: {
      tolerance: 200,      // 200ms同步容差
      checkInterval: 100,  // 100ms检查间隔
      autoCorrect: true    // 自动纠正
    },
    
    // 控制面板状态
    showControls: true,
    showSpeedPanel: false,
    showVolumePanel: false,
    showProgressDetail: false,
    
    // 播放设置
    playbackRates: [0.5, 0.75, 1.0, 1.25, 1.5, 2.0],
    
    // 音频文件状态
    audioFiles: [],
    currentAudioIndex: 0
  },

  /**
   * 组件生命周期
   */
  lifetimes: {
    attached() {
      this.initAudioController()
    },
    
    detached() {
      this.destroyAudioController()
    }
  },

  /**
   * 属性监听器
   */
  observers: {
    'audioGenerationProgress': function(newProgress) {
      if (newProgress) {
        this.setData({
          audioProgress: {
            percentage: newProgress.percentage || 0,
            message: newProgress.message || '正在处理...',
            current_slide: newProgress.currentSlide || 0,
            total_slides: newProgress.totalSlides || 0
          }
        })
      }
    }
  },

  /**
   * 组件方法
   */
  methods: {
    /**
     * 初始化音频控制器
     */
    initAudioController() {
      if (!this.data.audioMode) return
      
      const audioContext = wx.createInnerAudioContext()
      
      // 设置音频事件监听
      audioContext.onCanplay(() => {
        console.log('音频可以播放')
        this.setData({
          duration: audioContext.duration
        })
        this.triggerEvent('canplay')
      })
      
      audioContext.onTimeUpdate(() => {
        this.setData({
          currentTime: audioContext.currentTime,
          duration: audioContext.duration
        })
        this.triggerEvent('timeupdate', {
          currentTime: audioContext.currentTime,
          duration: audioContext.duration
        })
        this.checkSyncAccuracy()
      })
      
      audioContext.onPlay(() => {
        this.setData({ isPlaying: true })
        this.triggerEvent('play')
      })
      
      audioContext.onPause(() => {
        this.setData({ isPlaying: false })
        this.triggerEvent('pause')
      })
      
      audioContext.onEnded(() => {
        this.setData({ isPlaying: false })
        this.triggerEvent('ended')
      })
      
      audioContext.onError((error) => {
        console.error('音频播放错误:', error)
        this.triggerEvent('error', { error })
      })
      
      this.setData({ audioContext })
    },
    
    /**
     * 播放指定幻灯片音频
     */
    async playSlide(slideIndex, autoPlay = true) {
      const slides = this.data.courseData?.slides || []
      if (slideIndex < 0 || slideIndex >= slides.length) {
        console.error('幻灯片索引超出范围')
        return
      }
      
      const slide = slides[slideIndex]
      const audioContext = this.data.audioContext
      
      if (!audioContext) {
        console.error('音频上下文未初始化')
        return
      }
      
      // 切换音频源
      if (slide.audio_url) {
        audioContext.src = slide.audio_url
        audioContext.playbackRate = this.data.playbackRate
        
        if (autoPlay) {
          try {
            await audioContext.play()
            this.triggerEvent('slidechange', { slideIndex, audioUrl: slide.audio_url })
          } catch (error) {
            console.error('播放音频失败:', error)
            this.triggerEvent('error', { error, slideIndex })
          }
        }
      } else {
        console.warn(`幻灯片 ${slideIndex + 1} 没有音频文件`)
        this.triggerEvent('noaudio', { slideIndex })
      }
    },
    
    /**
     * 播放/暂停切换
     */
    togglePlay() {
      if (this.data.isPlaying) {
        this.pause()
      } else {
        this.play()
      }
    },
    
    /**
     * 播放音频
     */
    async play() {
      const audioContext = this.data.audioContext
      if (audioContext) {
        try {
          await audioContext.play()
        } catch (error) {
          console.error('播放失败:', error)
          this.triggerEvent('error', { error })
        }
      }
    },
    
    /**
     * 暂停音频
     */
    pause() {
      const audioContext = this.data.audioContext
      if (audioContext) {
        audioContext.pause()
      }
    },
    
    /**
     * 设置播放进度
     */
    seek(time) {
      const audioContext = this.data.audioContext
      if (audioContext && this.data.duration > 0) {
        audioContext.seek(time)
      }
    },
    
    /**
     * 设置播放速度
     */
    setPlaybackRate(rate) {
      if (rate < 0.5 || rate > 2.0) {
        console.error('播放速度必须在0.5-2.0之间')
        return
      }
      
      this.setData({ 
        playbackRate: rate,
        selectedSpeed: rate
      })
      
      const audioContext = this.data.audioContext
      if (audioContext) {
        audioContext.playbackRate = rate
      }
      
      this.triggerEvent('speedchange', { rate })
    },
    
    /**
     * 设置音量
     */
    setVolume(volume) {
      if (volume < 0 || volume > 100) {
        console.error('音量必须在0-100之间')
        return
      }
      
      this.setData({ volume })
      
      const audioContext = this.data.audioContext
      if (audioContext) {
        audioContext.volume = volume / 100
      }
      
      this.triggerEvent('volumechange', { volume })
    },
    
    /**
     * 检查同步精度
     */
    checkSyncAccuracy() {
      if (!this.data.syncConfig.autoCorrect) return
      
      const audioTime = this.data.currentTime * 1000
      const slideTime = this.getSlideDisplayTime()
      const diff = Math.abs(audioTime - slideTime)
      
      if (diff > this.data.syncConfig.tolerance) {
        console.warn(`音画同步偏差: ${diff}ms`)
        this.triggerEvent('syncerror', { difference: diff, audioTime, slideTime })
      }
    },
    
    /**
     * 获取幻灯片显示时间 - 简化实现
     */
    getSlideDisplayTime() {
      // 这里应该根据实际的幻灯片显示逻辑来计算时间
      // 简化实现：假设幻灯片切换时间与音频时间一致
      return this.data.currentTime * 1000
    },
    
    /**
     * 销毁音频控制器
     */
    destroyAudioController() {
      const audioContext = this.data.audioContext
      if (audioContext) {
        audioContext.destroy()
      }
    },
    
    /**
     * 进度条拖拽事件
     */
    onProgressChange(e) {
      const value = e.detail.value
      const seekTime = (value / 100) * this.data.duration
      this.seek(seekTime)
    },
    
    /**
     * 显示/隐藏速度面板
     */
    toggleSpeedPanel() {
      this.setData({
        showSpeedPanel: !this.data.showSpeedPanel,
        showVolumePanel: false
      })
    },
    
    /**
     * 显示/隐藏音量面板
     */
    toggleVolumePanel() {
      this.setData({
        showVolumePanel: !this.data.showVolumePanel,
        showSpeedPanel: false
      })
    },
    
    /**
     * 选择播放速度
     */
    selectSpeed(e) {
      const speed = e.currentTarget.dataset.speed
      this.setPlaybackRate(speed)
      this.setData({ showSpeedPanel: false })
    },
    
    /**
     * 音量调节
     */
    onVolumeChange(e) {
      const volume = e.detail.value
      this.setVolume(volume)
    },
    
    /**
     * 格式化时间显示
     */
    formatTime(seconds) {
      if (!seconds || isNaN(seconds)) return '00:00'
      
      const mins = Math.floor(seconds / 60)
      const secs = Math.floor(seconds % 60)
      return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
    },

    /**
     * 初始化音频播放器
     */
    initAudioPlayer() {
      if (!this.data.audioMode) return
      
      const audioContext = wx.createInnerAudioContext()
      
      // 设置音频事件监听
      audioContext.onCanplay(() => {
        console.log('音频可以播放')
        this.setData({
          duration: audioContext.duration,
          loading: false,
          error: null
        })
        this.triggerEvent('canplay')
      })
      
      audioContext.onTimeUpdate(() => {
        this.setData({
          currentTime: audioContext.currentTime,
          duration: audioContext.duration
        })
        this.triggerEvent('timeupdate', {
          currentTime: audioContext.currentTime,
          duration: audioContext.duration
        })
        this.checkSyncAccuracy()
      })
      
      audioContext.onPlay(() => {
        this.setData({ isPlaying: true })
        this.triggerEvent('play')
      })
      
      audioContext.onPause(() => {
        this.setData({ isPlaying: false })
        this.triggerEvent('pause')
      })
      
      audioContext.onStop(() => {
        this.setData({ 
          isPlaying: false,
          currentTime: 0
        })
        this.triggerEvent('stop')
      })
      
      audioContext.onEnded(() => {
        this.setData({ 
          isPlaying: false,
          currentTime: 0
        })
        this.triggerEvent('ended')
        this.playNext() // 自动播放下一个
      })
      
      audioContext.onError((res) => {
        console.error('音频播放错误:', res)
        
        // 检查错误类型，对于格式兼容性问题，尝试容错处理
        if (res.errMsg && res.errMsg.includes('seek audio fail')) {
          console.log('🔧 检测到音频格式兼容性问题，尝试继续播放')
          // 不显示错误提示，让音频继续尝试播放
          return
        }
        
        this.setData({ 
          isPlaying: false,
          loading: false,
          error: '音频播放失败'
        })
        this.triggerEvent('error', res)
        
        // 只在严重错误时才显示提示
        if (res.errCode !== -1) {
          wx.showToast({
            title: '音频播放失败',
            icon: 'none'
          })
        }
      })
      
      this.setData({ audioContext })
    },

    /**
     * 查询音频生成进度
     */
    checkAudioProgress() {
      if (!this.data.courseId) return
      
      this.setData({ generationInProgress: true })
      
      const app = getApp()
      wx.request({
        url: `${app.globalData.baseURL}/content/audio-progress/${this.data.courseId}`,
        method: 'GET',
        header: {
          'Authorization': `Bearer ${app.globalData.token}`
        },
        success: (response) => {
          try {
            if (response.data.success) {
              const progress = response.data.data
              this.setData({ 
                audioProgress: progress,
                audioFiles: progress.audio_files || []
              })
              
              // 如果音频生成完成，更新播放列表
              if (progress.status === 'completed') {
                this.updateAudioPlaylist(progress.audio_files)
                this.setData({ generationInProgress: false })
              } else if (progress.status === 'failed') {
                this.setData({ 
                  generationInProgress: false,
                  error: progress.error_message || '音频生成失败'
                })
              }
            }
          } catch (error) {
            console.error('解析音频进度响应失败:', error)
            this.setData({ 
              generationInProgress: false,
              error: '解析音频进度失败'
            })
          }
        },
        fail: (error) => {
          console.error('查询音频进度失败:', error)
          this.setData({ 
            generationInProgress: false,
            error: '查询音频进度失败'
          })
        }
      })
    },

    /**
     * 开始音频生成并显示进度
     */
    startAudioGeneration() {
      if (!this.data.courseId) {
        wx.showToast({
          title: '课程ID不存在',
          icon: 'error'
        })
        return
      }

      // 显示进度模态框
      this.setData({ 
        generationInProgress: true,
        showProgressModal: true
      })

      const app = getApp()
      wx.request({
        url: `${app.globalData.baseURL}/audio/generate`,
        method: 'POST',
        header: {
          'Authorization': `Bearer ${app.globalData.token}`,
          'Content-Type': 'application/json'
        },
        data: {
          course_id: this.data.courseId,
          voice_type: 'zhixiaobai',
          regenerate: false
        },
        success: (response) => {
          try {
            if (response.data && response.data.code === 200) {
              wx.showToast({
                title: '开始生成音频',
                icon: 'success'
              })
              
              // 开始轮询进度
              this.startProgressPolling()
            } else {
              throw new Error(response.data?.message || '启动音频生成失败')
            }
          } catch (error) {
            console.error('解析音频生成响应失败:', error)
            this.setData({ 
              generationInProgress: false,
              showProgressModal: false,
              error: error.message || '解析响应失败'
            })
            
            wx.showToast({
              title: '启动失败',
              icon: 'error'
            })
          }
        },
        fail: (error) => {
          console.error('启动音频生成失败:', error)
          this.setData({ 
            generationInProgress: false,
            showProgressModal: false,
            error: error.errMsg || '启动音频生成失败'
          })
          
          wx.showToast({
            title: '启动失败',
            icon: 'error'
          })
        }
      })
    },

    /**
     * 开始进度轮询
     */
    startProgressPolling() {
      // 进度组件会自动处理轮询，这里只需要监听事件
    },

    /**
     * 处理进度更新事件
     */
    onProgressUpdate(e) {
      const { progress } = e.detail
      
      // 更新本地进度状态
      this.setData({
        audioGenerationProgress: progress
      })

      // 如果完成，更新音频列表
      if (progress.status === 'completed') {
        this.checkAudioProgress()
        this.setData({ 
          generationInProgress: false,
          showProgressModal: false
        })
        
        wx.showToast({
          title: '音频生成完成！',
          icon: 'success'
        })
      } else if (progress.status === 'failed') {
        this.setData({ 
          generationInProgress: false,
          error: progress.errorMessage || '音频生成失败'
        })
      }
    },

    /**
     * 处理进度模态框最小化
     */
    onProgressMinimize() {
      this.setData({ showProgressModal: false })
    },

    /**
     * 处理进度模态框取消
     */
    onProgressCancel() {
      this.setData({ 
        showProgressModal: false,
        generationInProgress: false
      })
    },

    /**
     * 处理进度重试
     */
    onProgressRetry() {
      this.startAudioGeneration()
    },

    /**
     * 处理进度完成
     */
    onProgressComplete() {
      this.setData({ showProgressModal: false })
      // 可以触发播放或其他操作
      this.triggerEvent('audioReady')
    },

    /**
     * 显示详细进度
     */
    showDetailProgress() {
      this.setData({ showProgressModal: true })
    },

    /**
     * 更新音频播放列表
     */
    updateAudioPlaylist(audioFiles) {
      const validAudioFiles = audioFiles.filter(file => 
        file.status === 'completed' && file.file_url
      )
      
      this.setData({ 
        audioFiles: validAudioFiles,
        currentAudioIndex: 0
      })
      
      // 如果有可用的音频文件，加载第一个
      if (validAudioFiles.length > 0) {
        this.loadAudio(validAudioFiles[0].file_url)
      }
    },

    /**
     * 加载音频文件
     */
    loadAudio(audioUrl) {
      if (!audioUrl || !this.data.audioContext) return
      
      this.setData({ 
        loading: true,
        error: null
      })
      
      try {
        this.data.audioContext.src = audioUrl
        this.data.audioContext.title = `幻灯片 ${this.data.currentAudioIndex + 1} 音频`
      } catch (error) {
        console.error('加载音频失败:', error)
        this.setData({ 
          loading: false,
          error: '加载音频失败'
        })
      }
    },

    /**
     * 播放下一个音频
     */
    playNext() {
      const { audioFiles, currentAudioIndex } = this.data
      
      if (currentAudioIndex < audioFiles.length - 1) {
        const nextIndex = currentAudioIndex + 1
        this.setData({ currentAudioIndex: nextIndex })
        this.loadAudio(audioFiles[nextIndex].file_url)
        
        if (this.data.autoPlay) {
          this.play()
        }
      }
    },

    /**
     * 播放上一个音频
     */
    playPrevious() {
      const { audioFiles, currentAudioIndex } = this.data
      
      if (currentAudioIndex > 0) {
        const prevIndex = currentAudioIndex - 1
        this.setData({ currentAudioIndex: prevIndex })
        this.loadAudio(audioFiles[prevIndex].file_url)
        
        if (this.data.autoPlay) {
          this.play()
        }
      }
    },

    /**
     * 切换播放/暂停
     */
    togglePlay() {
      if (this.data.isPlaying) {
        this.pause()
      } else {
        this.play()
      }
    },

    /**
     * 播放音频
     */
    play() {
      if (!this.data.audioContext) return
      
      try {
        this.data.audioContext.play()
      } catch (error) {
        console.error('播放失败:', error)
        this.setData({ error: '播放失败' })
      }
    },

    /**
     * 暂停音频
     */
    pause() {
      if (!this.data.audioContext) return
      
      try {
        this.data.audioContext.pause()
      } catch (error) {
        console.error('暂停失败:', error)
      }
    },

    /**
     * 停止音频
     */
    stop() {
      if (!this.data.audioContext) return
      
      try {
        this.data.audioContext.stop()
        this.setData({ currentTime: 0 })
      } catch (error) {
        console.error('停止失败:', error)
      }
    },

    /**
     * 设置播放进度
     */
    seekTo(time) {
      if (!this.data.audioContext) return
      
      try {
        this.data.audioContext.seek(time)
        this.setData({ currentTime: time })
      } catch (error) {
        console.error('跳转失败:', error)
      }
    },

    /**
     * 设置播放速度
     */
    setPlaybackRate(rate) {
      if (!this.data.audioContext) return
      
      try {
        this.data.audioContext.playbackRate = rate
        this.setData({ playbackRate: rate })
      } catch (error) {
        console.error('设置播放速度失败:', error)
      }
    },

    /**
     * 设置音量
     */
    setVolume(volume) {
      if (!this.data.audioContext) return
      
      try {
        this.data.audioContext.volume = volume / 100
        this.setData({ volume: volume })
      } catch (error) {
        console.error('设置音量失败:', error)
      }
    },

    /**
     * 切换进度详情显示
     */
    toggleProgressDetail() {
      this.setData({ 
        showProgressDetail: !this.data.showProgressDetail 
      })
    },

    /**
     * 刷新音频进度
     */
    refreshProgress() {
      this.checkAudioProgress()
    }
  },

  /**
   * 组件观察器
   */
  observers: {
    'currentSlideIndex': function(newIndex) {
      if (this.data.audioMode && this.data.autoPlay) {
        this.playSlide(newIndex, true)
      }
    }
  }
}) 