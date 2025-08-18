// utils/audio-sync.js
// 音画同步管理器

class AudioSyncManager {
  constructor(options = {}) {
    this.options = {
      tolerance: options.tolerance || 200,        // 同步容差（毫秒）
      checkInterval: options.checkInterval || 100, // 检查间隔（毫秒）
      autoCorrect: options.autoCorrect !== false,  // 自动纠正
      preloadNext: options.preloadNext !== false,  // 预加载下一张
      autoPlayNext: options.autoPlayNext !== false, // 自动播放下一张
      syncMode: options.syncMode || 'auto',        // 同步模式：auto, manual, strict
      ...options
    }
    
    console.log('AudioSyncManager构造函数，最终options:', this.options)
    
    this.audioContext = null
    this.slides = []
    this.currentSlideIndex = 0
    this.isPlaying = false
    this.isPaused = false
    
    // 同步状态
    this.syncStatus = {
      isInSync: true,
      lastSyncTime: Date.now(),
      driftAmount: 0,
      correctionCount: 0
    }
    
    // 事件监听器
    this.eventListeners = {
      syncError: [],
      syncCorrected: [],
      slideChange: [],
      playStateChange: [],
      bufferUpdate: []
    }
    
    // 定时器
    this.syncCheckTimer = null
    this.preloadTimer = null
    
    // 缓存
    this.audioCache = new Map()
    this.slideTimings = []
    
    this.init()
  }
  
  init() {
    this.bindEvents()
    console.log('AudioSyncManager initialized with options:', this.options)
  }
  
  // 设置音频上下文
  setAudioContext(audioContext) {
    if (this.audioContext) {
      this.unbindAudioEvents()
    }
    
    this.audioContext = audioContext
    this.bindAudioEvents()
  }
  
  // 设置幻灯片数据
  setSlides(slides) {
    this.slides = slides
    this.calculateSlideTimings()
    this.preloadAudioFiles()
  }
  
  // 更新设置
  updateOptions(newOptions) {
    this.options = { ...this.options, ...newOptions }
    console.log('AudioSyncManager: 设置已更新', this.options)
  }
  
  // 计算幻灯片时间点
  calculateSlideTimings() {
    let currentTime = 0
    this.slideTimings = this.slides.map((slide, index) => {
      const timing = {
        slideIndex: index,
        startTime: currentTime,
        duration: slide.duration || 10, // 默认10秒
        endTime: currentTime + (slide.duration || 10),
        audioUrl: slide.audio_url,
        title: slide.title
      }
      currentTime = timing.endTime
      return timing
    })
    
    console.log('Slide timings calculated:', this.slideTimings)
  }
  
  // 预加载音频文件
  async preloadAudioFiles() {
    if (!this.options.preloadNext) return
    
    const preloadPromises = this.slides.slice(0, 3).map(slide => {
      if (slide.audio_url && !this.audioCache.has(slide.id)) {
        return this.preloadAudio(slide.audio_url, slide.id)
      }
    }).filter(Boolean)
    
    try {
      await Promise.all(preloadPromises)
      console.log('Audio files preloaded')
    } catch (error) {
      console.warn('Some audio files failed to preload:', error)
    }
  }
  
  // 预加载单个音频
  preloadAudio(audioUrl, slideId) {
    return new Promise((resolve, reject) => {
      const audio = wx.createInnerAudioContext()
      audio.src = audioUrl
      
      // 设置超时
      const timeout = setTimeout(() => {
        audio.destroy()
        reject(new Error(`音频预加载超时: ${audioUrl}`))
      }, 5000) // 5秒超时
      
      audio.onCanplay(() => {
        clearTimeout(timeout)
        this.audioCache.set(slideId, {
          url: audioUrl,
          preloaded: true,
          duration: audio.duration || 0
        })
        audio.destroy()
        resolve()
      })
      
      audio.onError((error) => {
        clearTimeout(timeout)
        console.warn(`音频预加载失败: ${audioUrl}`, error)
        audio.destroy()
        // 音频预加载失败时，仍然缓存该音频URL，但标记为未预加载
        this.audioCache.set(slideId, {
          url: audioUrl,
          preloaded: false,
          duration: 0,
          error: error
        })
        // 不要reject，而是resolve，避免阻塞其他音频的预加载
        resolve()
      })
    })
  }
  
  // 绑定音频事件
  bindAudioEvents() {
    if (!this.audioContext) return
    
    this.audioContext.onTimeUpdate(() => {
      this.handleTimeUpdate()
    })
    
    this.audioContext.onPlay(() => {
      this.isPlaying = true
      this.isPaused = false
      this.startSyncCheck()
      this.emit('playStateChange', { playing: true, paused: false })
    })
    
    this.audioContext.onPause(() => {
      this.isPaused = true
      this.isPlaying = false
      this.stopSyncCheck()
      console.log('AudioSyncManager: 音频已暂停')
      this.emit('playStateChange', { playing: false, paused: true })
    })
    
    this.audioContext.onStop(() => {
      this.isPlaying = false
      this.isPaused = false
      this.stopSyncCheck()
      this.emit('playStateChange', { playing: false, paused: false })
    })
    
    this.audioContext.onEnded(() => {
      this.handleAudioEnded()
    })
    
    this.audioContext.onError((error) => {
      console.error('Audio context error:', error)
      
      // 检查是否是格式兼容性问题
      if (error.errMsg && (error.errMsg.includes('seek audio fail') || error.errMsg.includes('pause audio fail'))) {
        console.log('🔧 AudioSyncManager: 音频格式兼容性问题，继续运行')
        // 对于格式问题，不触发同步错误
        return
      }
      
      this.emit('syncError', { type: 'audio_error', error })
    })
  }
  
  // 解绑音频事件
  unbindAudioEvents() {
    if (this.audioContext) {
      this.audioContext.offTimeUpdate()
      this.audioContext.offPlay()
      this.audioContext.offPause()
      this.audioContext.offStop()
      this.audioContext.offEnded()
      this.audioContext.offError()
    }
  }
  
  // 绑定通用事件
  bindEvents() {
    // 页面可见性变化
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', () => {
        if (document.hidden && this.isPlaying) {
          this.handlePageHidden()
        } else if (!document.hidden && this.isPaused) {
          this.handlePageVisible()
        }
      })
    }
  }
  
  // 处理时间更新
  handleTimeUpdate() {
    if (!this.isPlaying || !this.audioContext) return
    
    const currentTime = this.audioContext.currentTime * 1000 // 转换为毫秒
    
    // 注意：由于我们使用独立的音频文件而不是连续音频流，
    // 基于时间的自动同步不适用，这里只更新同步状态
    // 幻灯片切换由音频结束事件处理
    
    // 更新同步状态
    this.updateSyncStatus(currentTime)
  }
  
  // 计算期望的幻灯片索引
  calculateExpectedSlideIndex(currentTime) {
    for (let i = 0; i < this.slideTimings.length; i++) {
      const timing = this.slideTimings[i]
      if (currentTime >= timing.startTime && currentTime < timing.endTime) {
        return i
      }
    }
    return this.slideTimings.length - 1 // 返回最后一张
  }
  
  // 切换到指定幻灯片
  switchToSlide(slideIndex, currentTime) {
    if (slideIndex < 0 || slideIndex >= this.slides.length) return
    
    const previousIndex = this.currentSlideIndex
    this.currentSlideIndex = slideIndex
    
    console.log(`Switching from slide ${previousIndex} to ${slideIndex} at ${currentTime}ms`)
    
    this.emit('slideChange', {
      previousIndex,
      currentIndex: slideIndex,
      slide: this.slides[slideIndex],
      timing: this.slideTimings[slideIndex],
      currentTime
    })
    
    // 预加载下一张音频
    this.preloadNextSlideAudio(slideIndex)
  }
  
  // 预加载下一张幻灯片音频
  preloadNextSlideAudio(currentIndex) {
    if (!this.options.preloadNext) return
    
    const nextIndex = currentIndex + 1
    if (nextIndex < this.slides.length) {
      const nextSlide = this.slides[nextIndex]
      if (nextSlide.audio_url && !this.audioCache.has(nextSlide.id)) {
        this.preloadAudio(nextSlide.audio_url, nextSlide.id).catch(console.warn)
      }
    }
  }
  
  // 更新同步状态
  updateSyncStatus(currentTime) {
    const expectedTiming = this.slideTimings[this.currentSlideIndex]
    if (!expectedTiming) return
    
    const slideStartTime = expectedTiming.startTime
    const slideCurrentTime = currentTime - slideStartTime
    const maxSlideTime = expectedTiming.duration * 1000
    
    // 计算漂移量
    const drift = Math.abs(slideCurrentTime - (currentTime - slideStartTime))
    this.syncStatus.driftAmount = drift
    
    // 检查是否在容差范围内
    const wasInSync = this.syncStatus.isInSync
    this.syncStatus.isInSync = drift <= this.options.tolerance
    
    if (wasInSync && !this.syncStatus.isInSync) {
      console.warn(`Sync lost: drift ${drift}ms > tolerance ${this.options.tolerance}ms`)
      this.emit('syncError', {
        type: 'drift',
        drift,
        tolerance: this.options.tolerance,
        currentTime,
        slideIndex: this.currentSlideIndex
      })
      
      // 自动纠正
      if (this.options.autoCorrect) {
        this.correctSync()
      }
    }
    
    this.syncStatus.lastSyncTime = Date.now()
  }
  
  // 纠正同步
  correctSync() {
    // 🔥 暂时禁用自动同步纠正，因为我们使用独立音频文件
    // 幻灯片切换应该由音频结束事件驱动，而不是时间计算
    console.log('🔴 [AudioSync] 自动同步纠正已禁用 - 使用音频结束事件驱动切换')
    return
    
    /* 原来的代码 - 暂时禁用
    if (!this.audioContext || !this.isPlaying) return
    
    const currentTime = this.audioContext.currentTime * 1000
    const expectedSlideIndex = this.calculateExpectedSlideIndex(currentTime)
    
    if (expectedSlideIndex !== this.currentSlideIndex) {
      console.log(`Correcting sync: jumping to slide ${expectedSlideIndex}`)
      this.switchToSlide(expectedSlideIndex, currentTime)
      this.syncStatus.correctionCount++
      
      this.emit('syncCorrected', {
        correctedSlideIndex: expectedSlideIndex,
        previousSlideIndex: this.currentSlideIndex,
        currentTime,
        correctionCount: this.syncStatus.correctionCount
      })
    }
    */
  }
  
  // 开始同步检查
  startSyncCheck() {
    this.stopSyncCheck()
    this.syncCheckTimer = setInterval(() => {
      this.handleTimeUpdate()
    }, this.options.checkInterval)
  }
  
  // 停止同步检查
  stopSyncCheck() {
    if (this.syncCheckTimer) {
      clearInterval(this.syncCheckTimer)
      this.syncCheckTimer = null
    }
  }
  
  // 处理音频结束
  handleAudioEnded() {
    this.isPlaying = false
    this.stopSyncCheck()
    
    console.log('AudioSyncManager: 音频播放结束，当前幻灯片:', this.currentSlideIndex)
    
    // 检查是否被用户手动暂停
    if (this.isPaused) {
      console.log('AudioSyncManager: 检测到用户暂停，不自动播放下一张')
      this.emit('playStateChange', { playing: false, paused: true, ended: true })
      return
    }
    
    // 根据设置决定是否自动播放下一张
    if (this.options.autoPlayNext && this.currentSlideIndex < this.slides.length - 1) {
      console.log('AudioSyncManager: 自动播放下一张幻灯片')
      setTimeout(() => {
        this.playNextSlide()
      }, 500) // 延迟500ms，避免切换过快
    } else {
      console.log('AudioSyncManager: 播放结束，不自动切换')
      this.emit('playStateChange', { playing: false, ended: true })
    }
  }
  
  // 播放下一张幻灯片
  playNextSlide() {
    const nextIndex = this.currentSlideIndex + 1
    if (nextIndex < this.slides.length) {
      console.log('AudioSyncManager: 准备播放下一张幻灯片:', nextIndex)
      this.playSlide(nextIndex)
    } else {
      console.log('AudioSyncManager: 已是最后一张幻灯片')
    }
  }
  
  // 播放指定幻灯片
  playSlide(slideIndex) {
    if (slideIndex < 0 || slideIndex >= this.slides.length) return
    
    const slide = this.slides[slideIndex]
    if (!slide.audio_url) {
      console.warn(`Slide ${slideIndex} has no audio URL`)
      return
    }
    
    // 暂停时间同步检查，避免在切换过程中产生冲突
    this.stopSyncCheck()
    
    this.currentSlideIndex = slideIndex
    this.isPlaying = true
    
    // 🔴 移除直接音频控制，改为发出事件让播放器处理
    // if (this.audioContext) {
    //   this.audioContext.src = slide.audio_url
    //   this.audioContext.play()
    // }
    
    this.emit('slideChange', {
      currentIndex: slideIndex,
      slide: slide,
      timing: this.slideTimings[slideIndex]
    })
    
    // 延迟重启同步检查，给音频加载一些时间
    setTimeout(() => {
      if (this.isPlaying) {
        this.startSyncCheck()
      }
    }, 500)
  }
  
  // 跳转到指定时间点
  seekTo(timeMs) {
    if (!this.audioContext) return
    
    const timeSeconds = timeMs / 1000
    this.audioContext.seek(timeSeconds)
    
    // 更新当前幻灯片
    const expectedSlideIndex = this.calculateExpectedSlideIndex(timeMs)
    if (expectedSlideIndex !== this.currentSlideIndex) {
      this.switchToSlide(expectedSlideIndex, timeMs)
    }
  }
  
  // 设置播放速度
  setPlaybackRate(rate) {
    if (this.audioContext && this.audioContext.playbackRate !== undefined) {
      this.audioContext.playbackRate = rate
      
      // 重新计算时间点
      this.calculateSlideTimings()
    }
  }
  
  // 处理页面隐藏
  handlePageHidden() {
    if (this.isPlaying && !this.isPaused) {
      this.audioContext?.pause()
    }
  }
  
  // 处理页面显示
  handlePageVisible() {
    if (this.isPaused) {
      this.audioContext?.play()
    }
  }
  
  // 事件监听
  on(eventName, callback) {
    if (!this.eventListeners[eventName]) {
      this.eventListeners[eventName] = []
    }
    this.eventListeners[eventName].push(callback)
  }
  
  // 移除事件监听
  off(eventName, callback) {
    if (!this.eventListeners[eventName]) return
    
    const index = this.eventListeners[eventName].indexOf(callback)
    if (index > -1) {
      this.eventListeners[eventName].splice(index, 1)
    }
  }
  
  // 触发事件
  emit(eventName, data) {
    if (!this.eventListeners[eventName]) return
    
    this.eventListeners[eventName].forEach(callback => {
      try {
        callback(data)
      } catch (error) {
        console.error(`Error in ${eventName} event listener:`, error)
      }
    })
  }
  
  // 获取同步状态
  getSyncStatus() {
    return {
      ...this.syncStatus,
      currentSlideIndex: this.currentSlideIndex,
      totalSlides: this.slides.length,
      isPlaying: this.isPlaying,
      isPaused: this.isPaused,
      currentTime: this.audioContext?.currentTime || 0
    }
  }
  
  // 销毁
  destroy() {
    this.stopSyncCheck()
    this.unbindAudioEvents()
    
    if (this.preloadTimer) {
      clearTimeout(this.preloadTimer)
    }
    
    // 清理缓存
    this.audioCache.clear()
    
    // 清理事件监听器
    this.eventListeners = {}
    
    console.log('AudioSyncManager destroyed')
  }
}

module.exports = AudioSyncManager 