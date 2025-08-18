// pages/ppt-preview/ppt-preview.js
const app = getApp()
const { get } = require('../../utils/request.js')

Page({
  data: {
    // 任务信息
    taskId: '',
    
    // 预览状态
    isLoading: true,
    hasError: false,
    loadingMessage: '正在加载PPT预览...',
    errorMessage: '',
    
    // PPT信息
    pptTitle: '',
    slideInfo: '',
    htmlPreviewUrl: '',
    currentSlide: 1,      // 当前页码
    totalSlides: 0,       // 总页数
    
    // 下载相关
    pptxDownloadUrl: '',  // PPTX文件下载链接
    hasDownload: false,   // 是否可下载
    downloading: false,   // 下载状态
    
    // UI状态
    controlBarVisible: true,
    autoHideTimer: null,
    controlBarTimer: null, // 控制栏自动隐藏定时器
    
    // 手势相关
    touchStartX: null,
    touchStartY: null,
    touchStartTime: null,
    lastTap: null,        // 双击检测
  },

  onLoad(options) {
    console.log('📱 PPT预览页面加载', options)
    
    // 获取任务ID，如果没有则使用测试任务ID
    const taskId = options.taskId || 'workflow_direct_result'
    this.setData({
      taskId: taskId
    })
    
    console.log('🔍 使用任务ID:', taskId)
    this.loadPPTPreview()
    
    // 设置页面样式
    this.setupPageStyle()
  },

  onShow() {
    // 隐藏导航栏
    wx.hideHomeButton()
    
    // 调试信息：记录按钮状态
    setTimeout(() => {
      console.log('=== PPT预览页面调试信息 ===')
      console.log('页面数据状态:', {
        taskId: this.data.taskId,
        controlBarVisible: this.data.controlBarVisible,
        hasDownload: this.data.hasDownload,
        pptxDownloadUrl: this.data.pptxDownloadUrl,
        htmlPreviewUrl: this.data.htmlPreviewUrl
      })
    }, 1000)
  },

  onHide() {
    // 清理定时器
    if (this.data.autoHideTimer) {
      clearTimeout(this.data.autoHideTimer)
    }
  },

  // 设置页面样式
  setupPageStyle() {
    // 设置状态栏样式
    wx.setNavigationBarColor({
      frontColor: '#ffffff',
      backgroundColor: '#000000',
      animation: {
        duration: 0,
        timingFunc: 'easeIn'
      }
    })
  },

  // 加载PPT预览
  async loadPPTPreview() {
    try {
      this.setData({
        isLoading: true,
        hasError: false,
        loadingMessage: '正在加载PPT预览...'
      })

      // 轮询获取HTML预览状态
      await this.pollPreviewStatus()
      
    } catch (error) {
      console.error('❌ 加载PPT预览失败:', error)
      this.showError(error.message || '加载失败，请重试')
    }
  },

  // 轮询预览状态
  async pollPreviewStatus() {
    const maxAttempts = 90 // 最多轮询90次 (3分钟，每2秒一次)
    let attempts = 0
    
    const poll = async () => {
      attempts++
      console.log(`🔄 轮询预览状态 ${attempts}/${maxAttempts}`)
      
      try {
        const response = await this.getPreviewStatus()
        const data = response // request.js已经解析了data字段
        
        console.log('📊 预览状态:', data)
        
        if (data.ready_for_preview && data.html_preview_url) {
          // 转换完成，显示预览
          const app = getApp()
          // 🔧 构建WebView代理预览URL，解决小程序域名限制问题
          // 确保baseURL是字符串类型，避免Promise错误
          const baseURL = typeof app.globalData.baseURL === 'string' ? app.globalData.baseURL : (app.globalData.baseURL || '')
          
          // 🚀 使用WebView代理路由，绕过小程序域名限制
          // 将原始的 /api/v1/tmp/html/preview/xxx.html 转换为 /api/v1/webview/html/xxx.html
          let webviewProxyUrl = data.html_preview_url
          if (webviewProxyUrl && webviewProxyUrl.includes('/tmp/html/preview/')) {
            // 提取文件名部分
            const fileName = webviewProxyUrl.split('/tmp/html/preview/')[1]
            // 🔧 修复路径重复问题：直接替换路径部分，保持原有的/api/v1前缀
            webviewProxyUrl = webviewProxyUrl.replace('/tmp/html/preview/', '/webview/html/')
            console.log('🔄 转换为WebView代理URL:', webviewProxyUrl)
          }
          
          // 🔧 避免/api/v1路径重复：如果baseURL已包含/api/v1，则移除webviewProxyUrl中的/api/v1
          let fullUrl
          if (baseURL.endsWith('/api/v1') && webviewProxyUrl.startsWith('/api/v1/')) {
            // baseURL已包含/api/v1，移除webviewProxyUrl开头的/api/v1
            const cleanWebviewUrl = webviewProxyUrl.substring('/api/v1'.length)
            fullUrl = `${baseURL}${cleanWebviewUrl}`
          } else {
            fullUrl = `${baseURL}${webviewProxyUrl}`
          }
          
          console.log('🔗 构建完整预览URL:', fullUrl)
          console.log('📍 BaseURL:', baseURL)
          console.log('📍 原始路径:', data.html_preview_url)
          console.log('📍 WebView代理路径:', webviewProxyUrl)
          console.log('🚀 使用WebView代理，解决小程序域名限制问题')
          
          // 🆕 使用GET请求验证HTML文件是否可访问
          this.verifyHtmlFileWithGet(fullUrl).then((isAccessible) => {
            if (isAccessible) {
              console.log('✅ GET验证成功，HTML文件可访问')
              this.setData({
                htmlPreviewUrl: fullUrl
              })
            } else {
              console.warn('⚠️ GET验证失败，但仍尝试加载')
              this.setData({
                htmlPreviewUrl: fullUrl
              })
            }
          }).catch((error) => {
            console.error('❌ GET验证出错:', error)
            // 即使验证失败，也尝试加载
            this.setData({
              htmlPreviewUrl: fullUrl
            })
          })
          
          // 处理下载链接
          let downloadUrl = ''
          if (data.pptx_download_url) {
            if (data.pptx_download_url.startsWith('http')) {
              downloadUrl = data.pptx_download_url
            } else {
              // 相对路径，构建完整URL，避免重复/api/v1
              const baseURL = typeof app.globalData.baseURL === 'string' ? app.globalData.baseURL : (app.globalData.baseURL || '')
              const baseUrlParts = baseURL.replace('/api/v1', '')
              downloadUrl = `${baseUrlParts}${data.pptx_download_url}`
            }
            console.log('📥 PPTX下载链接:', downloadUrl)
          }
          
          this.setData({
            isLoading: false,
            pptTitle: data.task_id, // 可以从其他地方获取更好的标题
            slideInfo: '全屏预览模式',
            pptxDownloadUrl: downloadUrl,
            hasDownload: data.has_download || false
          })
          
          this.startAutoHideTimer()
          return
        }
        
        // 更新加载消息
        if (data.message) {
          this.setData({
            loadingMessage: data.message
          })
        }
        
        // 检查是否失败
        if (data.conversion_status === 'failed') {
          throw new Error(data.message || '转换失败')
        }
        
        // 继续轮询
        if (attempts < maxAttempts) {
          setTimeout(poll, 2000) // 2秒后重试
        } else {
          throw new Error('转换超时(3分钟)，请重试或联系技术支持')
        }
        
      } catch (error) {
        console.error('❌ 轮询状态失败:', error)
        throw error
      }
    }
    
    await poll()
  },

  // 获取预览状态
  async getPreviewStatus() {
    try {
      console.log('🌐 发起API请求:', `/coze/tasks/${this.data.taskId}/html-preview`)
      console.log('🔧 请求配置:', { needAuth: true })
      const app = getApp()
      console.log('📍 BaseURL:', app.globalData.baseURL)
      
      const response = await get(`/coze/tasks/${this.data.taskId}/html-preview`, { needAuth: true })
      console.log('✅ API响应成功:', response)
      return response
    } catch (error) {
      console.error('❌ API请求失败:', error)
      
      // 如果任务不存在且不是测试任务，尝试使用测试任务
      if (error.message.includes('任务不存在') && this.data.taskId !== 'workflow_direct_result') {
        console.log('🔄 任务不存在，尝试使用测试任务')
        this.setData({
          taskId: 'workflow_direct_result',
          loadingMessage: '任务已过期，正在加载演示PPT...'
        })
        return await this.getPreviewStatus()
      }
      
      // 更详细的错误处理
      let errorMessage = '获取预览状态失败'
      if (error.message) {
        errorMessage += ': ' + error.message
      }
      if (error.statusCode) {
        errorMessage += ` (状态码: ${error.statusCode})`
      }
      throw new Error(errorMessage)
    }
  },

  // 显示错误
  showError(message) {
    this.setData({
      isLoading: false,
      hasError: true,
      errorMessage: message
    })
  },

  // 重试
  onRetry() {
    this.loadPPTPreview()
  },

  // Web-view消息处理
  onWebViewMessage(e) {
    console.log('📨 收到Web-view消息:', e.detail.data)
    
    const messages = e.detail.data
    messages.forEach(message => {
      switch (message.type) {
        case 'slideChanged':
          this.setData({
            slideInfo: `第 ${message.current} / ${message.total} 页`
          })
          break
        case 'presentationLoaded':
          this.setData({
            pptTitle: message.title || this.data.pptTitle,
            slideInfo: `共 ${message.totalSlides} 页`
          })
          break
        case 'error':
          console.error('Web-view错误:', message.error)
          break
      }
    })
  },

  // Web-view加载完成
  onWebViewLoad(e) {
    console.log('✅ Web-view加载完成:', e)
    console.log('🔗 当前加载的URL:', this.data.htmlPreviewUrl)
    
    // 隐藏加载状态
    this.setData({
      isLoading: false
    })
    
    // 设置自动隐藏控制栏
    this.startAutoHideTimer()
  },

  // Web-view加载错误
  onWebViewError(e) {
    console.error('❌ Web-view加载错误:', e)
    console.error('🔗 失败的URL:', this.data.htmlPreviewUrl)
    console.error('📋 错误详情:', JSON.stringify(e))
    // 正确处理错误信息
    let errorMsg = '未知错误'
    if (e.detail) {
      if (typeof e.detail === 'string') {
        errorMsg = e.detail
      } else if (typeof e.detail === 'object') {
        errorMsg = JSON.stringify(e.detail)
      }
    }
    
    this.showError(`预览页面加载失败: ${errorMsg}`)
  },

  // 切换控制栏显示
  onToggleControls() {
    const visible = !this.data.controlBarVisible
    this.setData({
      controlBarVisible: visible
    })
    
    if (visible) {
      this.startAutoHideTimer()
    }
  },

  // 控制栏点击（阻止事件冒泡）
  onControlBarTap(e) {
    e.stopPropagation()
  },

  // 开始自动隐藏定时器
  startAutoHideTimer() {
    if (this.data.autoHideTimer) {
      clearTimeout(this.data.autoHideTimer)
    }
    
    this.data.autoHideTimer = setTimeout(() => {
      this.setData({
        controlBarVisible: false
      })
    }, 3000) // 3秒后自动隐藏
  },

  // 返回
  onBack() {
    wx.navigateBack({
      delta: 1
    })
  },

  // 切换全屏
  onToggleFullscreen() {
    // 小程序中已经是全屏，这里可以添加其他逻辑
    wx.showToast({
      title: '全屏模式',
      icon: 'none'
    })
  },

  // 分享
  onShare() {
    wx.showShareMenu({
      withShareTicket: true,
      showShareItems: ['shareAppMessage', 'shareTimeline']
    })
  },

  // 下载PPTX文件
  async onDownload() {
    if (!this.data.hasDownload || !this.data.pptxDownloadUrl) {
      wx.showToast({
        title: '暂无可下载文件',
        icon: 'none'
      })
      return
    }

    if (this.data.downloading) {
      wx.showToast({
        title: '正在下载中...',
        icon: 'loading'
      })
      return
    }

    console.log('📥 开始下载PPTX文件:', this.data.pptxDownloadUrl)

    // 显示下载确认
    wx.showModal({
      title: '下载PPT',
      content: '确定要下载此PPT文件吗？',
      confirmText: '下载',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          this.startDownload()
        }
      }
    })
  },

  // 开始下载
  async startDownload() {
    this.setData({ downloading: true })

    wx.showLoading({
      title: '准备下载...',
      mask: true
    })

    try {
      // 使用微信小程序的downloadFile API
      const downloadTask = wx.downloadFile({
        url: this.data.pptxDownloadUrl,
        header: {
          'Authorization': `Bearer ${wx.getStorageSync('token') || ''}`,
        },
        success: (res) => {
          console.log('📥 下载成功:', res)
          
          if (res.statusCode === 200) {
            // 下载成功，保存到本地
            wx.saveFile({
              tempFilePath: res.tempFilePath,
              success: (saveRes) => {
                console.log('💾 文件保存成功:', saveRes.savedFilePath)
                wx.hideLoading()
                wx.showToast({
                  title: '下载完成',
                  icon: 'success'
                })
                
                // 提示用户文件保存位置
                setTimeout(() => {
                  wx.showModal({
                    title: '下载完成',
                    content: `PPT文件已保存到本地\n可在文件管理器中查看`,
                    showCancel: false,
                    confirmText: '知道了'
                  })
                }, 1500)
              },
              fail: (err) => {
                console.error('💾 文件保存失败:', err)
                wx.hideLoading()
                wx.showToast({
                  title: '保存失败',
                  icon: 'error'
                })
              }
            })
          } else {
            throw new Error(`下载失败，状态码: ${res.statusCode}`)
          }
        },
        fail: (err) => {
          console.error('📥 下载失败:', err)
          wx.hideLoading()
          wx.showToast({
            title: '下载失败',
            icon: 'error'
          })
        },
        complete: () => {
          this.setData({ downloading: false })
        }
      })

      // 监听下载进度
      downloadTask.onProgressUpdate((res) => {
        console.log('📈 下载进度:', res.progress + '%')
        wx.showLoading({
          title: `下载中... ${res.progress}%`,
          mask: true
        })
      })

    } catch (error) {
      console.error('📥 下载异常:', error)
      wx.hideLoading()
      wx.showToast({
        title: '下载失败',
        icon: 'error'
      })
      this.setData({ downloading: false })
    }
  },

  // 分享给朋友
  onShareAppMessage() {
    return {
      title: this.data.pptTitle || 'AI生成的PPT',
      path: `/pages/ppt-preview/ppt-preview?taskId=${this.data.taskId}`,
      imageUrl: '' // 可以添加预览图
    }
  },

  // 分享到朋友圈
  onShareTimeline() {
    return {
      title: this.data.pptTitle || 'AI生成的PPT',
      query: `taskId=${this.data.taskId}`,
      imageUrl: '' // 可以添加预览图
    }
  },

  // 移动端手势支持
  onTouchStart(e) {
    this.touchStartX = e.touches[0].clientX
    this.touchStartY = e.touches[0].clientY
    this.touchStartTime = Date.now()
  },

  onTouchEnd(e) {
    if (!this.touchStartX || !this.touchStartY) return
    
    const touchEndX = e.changedTouches[0].clientX
    const touchEndY = e.changedTouches[0].clientY
    const touchEndTime = Date.now()
    
    const deltaX = touchEndX - this.touchStartX
    const deltaY = touchEndY - this.touchStartY
    const deltaTime = touchEndTime - this.touchStartTime
    
    const minSwipeDistance = 80  // 增加最小滑动距离，避免误触
    const maxSwipeTime = 500     // 增加最大滑动时间
    
    // 检测快速滑动手势
    if (deltaTime < maxSwipeTime) {
      if (Math.abs(deltaX) > Math.abs(deltaY) && Math.abs(deltaX) > minSwipeDistance) {
        if (deltaX > 0) {
          // 右滑手势 - 上一页
          console.log('🤚 检测到右滑手势 - 上一页')
          this.previousSlide()
        } else {
          // 左滑手势 - 下一页
          console.log('🤚 检测到左滑手势 - 下一页')
          this.nextSlide()
        }
      } else if (Math.abs(deltaY) > Math.abs(deltaX) && Math.abs(deltaY) > minSwipeDistance) {
        if (deltaY > 0) {
          // 下滑手势 - 显示控制栏
          console.log('🤚 检测到下滑手势 - 显示控制栏')
          this.showControls()
        } else {
          // 上滑手势 - 隐藏控制栏
          console.log('🤚 检测到上滑手势 - 隐藏控制栏')
          this.hideControls()
        }
      }
    }
    
    // 重置触摸坐标
    this.touchStartX = null
    this.touchStartY = null
    this.touchStartTime = null
  },

  // 双击检测
  onDoubleTap(e) {
    const currentTime = Date.now()
    const tapLength = currentTime - (this.lastTap || 0)
    
    if (tapLength < 300 && tapLength > 0) {
      // 双击进入/退出全屏
      console.log('🤚 检测到双击手势')
      this.onToggleFullscreen()
    }
    
    this.lastTap = currentTime
  },

  // 长按检测
  onLongPress(e) {
    console.log('🤚 检测到长按手势')
    // 长按显示更多选项
    wx.showActionSheet({
      itemList: ['分享给朋友', '复制链接', '保存到相册'],
      success: (res) => {
        switch (res.tapIndex) {
          case 0:
            this.onShare()
            break
          case 1:
            this.copyLink()
            break
          case 2:
            this.saveToAlbum()
            break
        }
      }
    })
  },

  // 复制链接
  copyLink() {
    wx.setClipboardData({
      data: this.data.htmlPreviewUrl,
      success: () => {
        wx.showToast({
          title: '链接已复制',
          icon: 'success'
        })
      }
    })
  },

  // 保存到相册（占位功能）
  saveToAlbum() {
    wx.showToast({
      title: '功能开发中',
      icon: 'none'
    })
  },

  // PPT翻页功能
  previousSlide() {
    if (!this.data.currentSlide || this.data.currentSlide <= 1) {
      wx.showToast({
        title: '已是第一页',
        icon: 'none',
        duration: 1000
      })
      return
    }
    
    const newSlide = this.data.currentSlide - 1
    this.setData({ currentSlide: newSlide })
    this.navigateToSlide(newSlide)
    
    // 显示页面提示
    wx.showToast({
      title: `第 ${newSlide} 页`,
      icon: 'none',
      duration: 800
    })
  },

  nextSlide() {
    if (!this.data.currentSlide || !this.data.totalSlides || this.data.currentSlide >= this.data.totalSlides) {
      wx.showToast({
        title: '已是最后一页',
        icon: 'none',
        duration: 1000
      })
      return
    }
    
    const newSlide = this.data.currentSlide + 1
    this.setData({ currentSlide: newSlide })
    this.navigateToSlide(newSlide)
    
    // 显示页面提示
    wx.showToast({
      title: `第 ${newSlide} 页`,
      icon: 'none',
      duration: 800
    })
  },

  // 导航到指定页面
  navigateToSlide(slideNumber) {
    console.log('导航到第', slideNumber, '页')
    
    // 通过webview的postMessage通知PPT内容页跳转
    try {
      // 如果PPT支持URL hash参数
      if (this.data.htmlPreviewUrl) {
        const baseURL = this.data.htmlPreviewUrl.split('#')[0].split('?')[0]
        const newURL = `${baseURL}#slide=${slideNumber}`
        this.setData({ htmlPreviewUrl: newURL })
      }
    } catch (error) {
      console.error('导航到幻灯片失败:', error)
    }
  },

  // 控制栏显示/隐藏
  showControls() {
    this.setData({ controlBarVisible: true })
    
    // 3秒后自动隐藏
    clearTimeout(this.controlBarTimer)
    this.controlBarTimer = setTimeout(() => {
      this.hideControls()
    }, 3000)
  },

  hideControls() {
    this.setData({ controlBarVisible: false })
    clearTimeout(this.controlBarTimer)
  },

  onToggleControls() {
    console.log('🎯 控制栏切换触发')
    if (this.data.controlBarVisible) {
      this.hideControls()
    } else {
      this.showControls()
    }
  },

  // 触摸覆盖层点击调试
  onTouchOverlayTap(e) {
    console.log('🔍 触摸覆盖层被点击:', e)
    console.log('点击坐标:', { x: e.detail.x, y: e.detail.y })
    
    // 检查是否点击到了按钮区域
    try {
      const systemInfo = wx.getWindowInfo()
      console.log('系统信息:', systemInfo)
    } catch (e) {
      console.warn('获取系统信息失败:', e)
    }
  },

  // 缩放切换
  onZoomToggle() {
    console.log('🔍 缩放功能触发')
    
    // 显示缩放选项
    wx.showActionSheet({
      itemList: ['适应屏幕', '原始大小', '放大150%', '放大200%'],
      success: (res) => {
        const zoomOptions = [
          { label: '适应屏幕', value: 100, toast: '💻 适应屏幕' },
          { label: '原始大小', value: 100, toast: '📱 原始大小' },
          { label: '放大150%', value: 150, toast: '🔍 放大到150%' },
          { label: '放大200%', value: 200, toast: '🔍 放大到200%' }
        ]
        
        const selected = zoomOptions[res.tapIndex]
        console.log('选择缩放:', selected)
        
        wx.showToast({
          title: selected.toast,
          icon: 'none',
          duration: 1500
        })
        
        // 这里可以添加实际的缩放逻辑
        // 由于是web-view，缩放需要通过postMessage与内嵌页面通信
      },
      fail: () => {
        console.log('用户取消缩放选择')
      }
    })
  },

  // 预览功能
  onPreview() {
    console.log('👁️ 预览功能触发')
    wx.showToast({
      title: '👁️ 预览模式切换',
      icon: 'none',
      duration: 1500
    })
    
    // 可以在这里添加预览模式切换逻辑
    console.log('当前页面状态:', {
      controlBarVisible: this.data.controlBarVisible,
      htmlPreviewUrl: this.data.htmlPreviewUrl,
      hasDownload: this.data.hasDownload
    })
  },

  // 复制链接功能
  onCopyLink() {
    console.log('🔗 复制链接功能触发')
    console.log('当前预览URL:', this.data.htmlPreviewUrl)
    
    if (this.data.htmlPreviewUrl) {
      wx.setClipboardData({
        data: this.data.htmlPreviewUrl,
        success: () => {
          console.log('✅ 链接复制成功:', this.data.htmlPreviewUrl)
          wx.showToast({
            title: '🔗 链接已复制',
            icon: 'success',
            duration: 1500
          })
        },
        fail: (error) => {
          console.error('❌ 链接复制失败:', error)
          wx.showToast({
            title: '复制失败',
            icon: 'error'
          })
        }
      })
    } else {
      console.warn('⚠️ 没有可用的预览链接')
      wx.showToast({
        title: '暂无预览链接',
        icon: 'none'
      })
    }
  },

  // 页面性能优化
  onReady() {
    // 预加载优化
    this.optimizePerformance()
  },

  optimizePerformance() {
    // 设置页面性能优化
    try {
      const deviceInfo = wx.getDeviceInfo()
      const windowInfo = wx.getWindowInfo()
      const systemInfo = { ...deviceInfo, ...windowInfo }
      console.log('📱 设备信息:', systemInfo)
      
      // 根据设备性能调整渲染策略
      if (systemInfo.platform === 'ios') {
        // iOS优化
        this.setData({
          renderOptimization: 'ios'
        })
      } else if (systemInfo.platform === 'android') {
        // Android优化
        this.setData({
          renderOptimization: 'android'
        })
      }
    } catch (e) {
      console.warn('获取设备信息失败，使用默认优化策略:', e)
      this.setData({
        renderOptimization: 'default'
      })
    }
  },

  // 🆕 使用GET请求验证HTML文件是否可访问
  async verifyHtmlFileWithGet(url) {
    console.log('🔍 开始GET验证HTML文件:', url)
    
    return new Promise((resolve) => {
      const app = getApp()
      
      wx.request({
        url: url,
        method: 'GET',
        header: {
          'Authorization': `Bearer ${wx.getStorageSync('token') || ''}`,
          'Content-Type': 'text/html'
        },
        timeout: 10000, // 10秒超时
        success: (res) => {
          console.log('✅ GET请求成功:', {
            statusCode: res.statusCode,
            contentLength: res.data ? res.data.length : 0,
            headers: res.header
          })
          
          if (res.statusCode === 200) {
            // 检查响应内容是否包含HTML标记
            const hasHtmlContent = res.data && (
              typeof res.data === 'string' && 
              (res.data.includes('<html') || res.data.includes('<!DOCTYPE'))
            )
            
            if (hasHtmlContent) {
              console.log('✅ GET验证成功：HTML文件存在且内容有效')
              resolve(true)
            } else {
              console.warn('⚠️ GET验证警告：响应不是有效的HTML内容')
              resolve(false)
            }
          } else {
            console.warn(`⚠️ GET验证失败：HTTP ${res.statusCode}`)
            resolve(false)
          }
        },
        fail: (error) => {
          console.error('❌ GET请求失败:', error)
          // 网络错误或其他问题
          resolve(false)
        }
      })
    })
  }
})