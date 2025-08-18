/**
 * 音频生成进度辅助工具
 * 提供统一的音频生成进度管理接口
 */

class AudioProgressHelper {
  constructor() {
    this.progressCallbacks = new Map()
    this.pollTimers = new Map()
  }

  /**
   * 启动音频生成
   * @param {number} courseId - 课程ID
   * @param {Object} options - 生成选项
   * @returns {Promise<boolean>} 是否成功启动
   */
  async startGeneration(courseId, options = {}) {
    try {
      const app = getApp()
      const response = await wx.request({
        url: `${app.globalData.baseURL}/audio/generate`,
        method: 'POST',
        header: {
          'Authorization': `Bearer ${app.globalData.token}`,
          'Content-Type': 'application/json'
        },
        data: {
          course_id: courseId,
          voice_type: options.voiceType || 'zhixiaobai',
          speed: options.speed || 1,
          volume: options.volume || 80,
          regenerate: options.regenerate || false
        }
      })

      if (response.data && response.data.code === 200) {
        // 开始轮询进度
        this.startPolling(courseId)
        return true
      } else {
        throw new Error(response.data?.message || '启动音频生成失败')
      }
    } catch (error) {
      console.error('启动音频生成失败:', error)
      throw error
    }
  }

  /**
   * 开始轮询进度
   * @param {number} courseId - 课程ID
   * @param {number} interval - 轮询间隔（毫秒）
   */
  startPolling(courseId, interval = 2000) {
    this.stopPolling(courseId) // 确保清除之前的定时器

    const poll = async () => {
      try {
        const progress = await this.fetchProgress(courseId)
        
        // 触发回调
        const callback = this.progressCallbacks.get(courseId)
        if (callback) {
          callback(progress)
        }

        // 如果完成或失败，停止轮询
        if (progress.status === 'completed' || progress.status === 'failed') {
          this.stopPolling(courseId)
        }
      } catch (error) {
        console.error('获取音频生成进度失败:', error)
      }
    }

    // 立即执行一次
    poll()

    // 设置定时轮询
    const timer = setInterval(poll, interval)
    this.pollTimers.set(courseId, timer)
  }

  /**
   * 停止轮询
   * @param {number} courseId - 课程ID
   */
  stopPolling(courseId) {
    const timer = this.pollTimers.get(courseId)
    if (timer) {
      clearInterval(timer)
      this.pollTimers.delete(courseId)
    }
  }

  /**
   * 获取进度信息
   * @param {number} courseId - 课程ID
   * @returns {Promise<Object>} 进度信息
   */
  async fetchProgress(courseId) {
    try {
      const app = getApp()
      const response = await wx.request({
        url: `${app.globalData.baseURL}/audio/status/${courseId}`,
        method: 'GET',
        header: {
          'Authorization': `Bearer ${app.globalData.token}`
        }
      })

      if (response.data && response.data.code === 200) {
        const progressData = response.data.data.progress
        return this.normalizeProgress(progressData)
      } else {
        throw new Error('获取进度失败')
      }
    } catch (error) {
      console.error('获取音频生成进度失败:', error)
      throw error
    }
  }

  /**
   * 标准化进度数据
   * @param {Object} rawProgress - 原始进度数据
   * @returns {Object} 标准化的进度数据
   */
  normalizeProgress(rawProgress) {
    if (!rawProgress) {
      return {
        status: 'idle',
        percentage: 0,
        totalSlides: 0,
        completedSlides: 0,
        currentSlide: '',
        errorMessage: '',
        slideProgress: 0,
        slideProgressText: ''
      }
    }

    return {
      status: rawProgress.status || 'processing',
      percentage: rawProgress.progress || 0,
      totalSlides: rawProgress.total_slides || 0,
      completedSlides: rawProgress.completed_slides || 0,
      currentSlide: rawProgress.current_slide || '',
      errorMessage: rawProgress.error_message || '',
      slideProgress: this.calculateSlideProgress(rawProgress),
      slideProgressText: this.getSlideProgressText(rawProgress)
    }
  }

  /**
   * 计算当前幻灯片进度
   */
  calculateSlideProgress(progressData) {
    if (!progressData.current_slide_progress) return 0
    
    const stage = progressData.current_stage || 'analyzing'
    const stageProgress = {
      'analyzing': 25,
      'generating': 75,
      'optimizing': 90,
      'completed': 100
    }
    
    return stageProgress[stage] || 0
  }

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
  }

  /**
   * 注册进度回调
   * @param {number} courseId - 课程ID
   * @param {Function} callback - 回调函数
   */
  onProgress(courseId, callback) {
    this.progressCallbacks.set(courseId, callback)
  }

  /**
   * 移除进度回调
   * @param {number} courseId - 课程ID
   */
  offProgress(courseId) {
    this.progressCallbacks.delete(courseId)
    this.stopPolling(courseId)
  }

  /**
   * 取消音频生成
   * @param {number} courseId - 课程ID
   * @returns {Promise<boolean>} 是否成功取消
   */
  async cancelGeneration(courseId) {
    try {
      const app = getApp()
      const response = await wx.request({
        url: `${app.globalData.baseURL}/audio/cancel/${courseId}`,
        method: 'POST',
        header: {
          'Authorization': `Bearer ${app.globalData.token}`
        }
      })

      if (response.data && response.data.code === 200) {
        this.stopPolling(courseId)
        return true
      } else {
        throw new Error('取消失败')
      }
    } catch (error) {
      console.error('取消音频生成失败:', error)
      throw error
    }
  }

  /**
   * 清理资源
   */
  cleanup() {
    // 停止所有轮询
    for (const courseId of this.pollTimers.keys()) {
      this.stopPolling(courseId)
    }
    
    // 清除所有回调
    this.progressCallbacks.clear()
  }
}

// 创建全局实例
const audioProgressHelper = new AudioProgressHelper()

module.exports = audioProgressHelper