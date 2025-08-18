// pages/ppt/preview/preview.js
Page({
  /**
   * 页面的初始数据
   */
  data: {
    pptURL: '',
    title: '',
    isLoading: true,
    loadError: false,
    isFullScreen: false,
    showActionSheet: false,
    showShareModal: false,
    currentSlide: 1,
    totalSlides: 0,
    zoomLevel: 100
  },

  /**
   * 生命周期函数--监听页面加载
   */
  onLoad(options) {
    console.log('PPT预览页面加载', options)
    
    if (!options.url) {
      wx.showModal({
        title: '错误',
        content: 'PPT链接不存在',
        showCancel: false,
        success: () => {
          wx.navigateBack()
        }
      })
      return
    }

    const pptURL = decodeURIComponent(options.url)
    const title = options.title ? decodeURIComponent(options.title) : 'PPT预览'
    const currentSlide = parseInt(options.slide) || 1
    const totalSlides = parseInt(options.total) || 0
    
    console.log('PPT URL:', pptURL)
    console.log('标题:', title)
    console.log('当前页:', currentSlide, '总页数:', totalSlides)
    
    // 验证URL格式
    if (!pptURL.startsWith('http://') && !pptURL.startsWith('https://')) {
      console.error('无效的URL格式:', pptURL)
      wx.showModal({
        title: '错误',
        content: 'PPT链接格式错误',
        showCancel: false,
        success: () => {
          wx.navigateBack()
        }
      })
      return
    }
    
    // 如果URL是API格式，转换为静态文件URL供web-view使用
    let staticPptURL = pptURL
    if (pptURL.includes('/api/v1/ppt/preview/')) {
      staticPptURL = pptURL.replace('/api/v1/ppt/preview/', '/ppt/')
    }
    
    console.log('原始URL:', pptURL)
    console.log('转换后的静态URL:', staticPptURL)
    
    this.setData({
      pptURL: staticPptURL,
      originalURL: pptURL, // 保存原始URL以备后用
      title: title,
      currentSlide: currentSlide,
      totalSlides: totalSlides
    })

    // 设置页面标题
    wx.setNavigationBarTitle({
      title: title.length > 10 ? title.substring(0, 10) + '...' : title
    })

    // 如果总页数为0，尝试动态获取
    if (totalSlides === 0) {
      // 优先使用API获取，失败后再尝试解析HTML
      this.getTotalSlidesFromAPI(staticPptURL)
    } else {
      // 如果已经有总页数，关闭加载状态
      this.setData({
        isLoading: false,
        loadError: false
    })
    }
  },

  /**
   * 页面显示时
   */
  onShow() {
    console.log('页面显示时状态检查 - 当前页:', this.data.currentSlide, '总页数:', this.data.totalSlides)
    // 如果已经有PPT URL但还没有总页数，重新获取
    if (this.data.pptURL && this.data.totalSlides === 0) {
      console.log('重新获取PPT总页数')
      this.getTotalSlidesFromAPI(this.data.pptURL)
    }
  },

  /**
   * 从API获取总页数
   */
  getTotalSlidesFromAPI(pptURL) {
    // 从PPT URL中提取文件名
    const urlParts = pptURL.split('/')
    const filename = urlParts[urlParts.length - 1]
    
    if (!filename) {
      console.error('无法从URL中提取文件名:', pptURL)
      this.getTotalSlidesFromHTML(pptURL)
      return
    }
    
    const { getBaseURLSync } = require('../../../utils/config')
    const baseURL = getBaseURLSync()
    const infoURL = `${baseURL}/ppt/info/${filename}`
     
     console.log('从API获取PPT总页数:', infoURL)
    
    wx.request({
      url: infoURL,
      method: 'GET',
      success: (res) => {
        console.log('API获取PPT信息成功:', res.data)
        if (res.statusCode === 200 && res.data && res.data.data) {
          const data = res.data.data
          const totalSlides = data.total_slides || 1
          
    this.setData({
            totalSlides: totalSlides,
            isLoading: false,
      loadError: false
    })
          
          console.log('API更新总页数为:', totalSlides)
          console.log('更新后页面状态 - 当前页:', this.data.currentSlide, '总页数:', this.data.totalSlides)
        } else {
          console.log('API响应格式错误，尝试解析HTML')
          // API失败，尝试解析HTML
          this.getTotalSlidesFromHTML(pptURL)
        }
      },
      fail: (error) => {
        console.error('API获取PPT信息失败:', error)
        // API失败，尝试解析HTML
        this.getTotalSlidesFromHTML(pptURL)
      }
    })
  },

  /**
   * 从HTML内容获取总页数（备用方案）
   */
  getTotalSlidesFromHTML(pptURL) {
    // 如果URL是API格式，转换为静态文件URL
    let staticURL = pptURL
    if (pptURL.includes('/api/v1/ppt/preview/')) {
      // 从API URL提取文件名，构建静态文件URL
      const filename = pptURL.split('/').pop()
      staticURL = pptURL.replace('/api/v1/ppt/preview/', '/ppt/')
    }
    
    console.log('开始获取PPT总页数:', staticURL)
    
    wx.request({
      url: staticURL,
      method: 'GET',
      success: (res) => {
        if (res.statusCode === 200 && res.data) {
          const htmlContent = res.data
          // 尝试从HTML中解析总页数
          const totalSlides = this.extractTotalSlides(htmlContent)
          console.log('从HTML解析到的总页数:', totalSlides)
          
          if (totalSlides > 0) {
            this.setData({
              totalSlides: totalSlides
            })
          } else {
            console.warn('未能从HTML中解析到有效的总页数')
            // 设置默认值，避免显示0页
            this.setData({
              totalSlides: 1
            })
          }
        }
      },
      fail: (error) => {
        console.error('获取PPT总页数失败:', error)
        // 设置默认值
        this.setData({
          totalSlides: 1
        })
      }
    })
  },

  /**
   * 从HTML内容中提取总页数
   */
  extractTotalSlides(htmlContent) {
    try {
      // 尝试多种方式解析总页数
      
      // 方法1: 查找reveal.js的幻灯片总数
      const slideMatch = htmlContent.match(/slide-number[^>]*>[\s\S]*?(\d+)\s*\/\s*(\d+)/i)
      if (slideMatch && slideMatch[2]) {
        return parseInt(slideMatch[2])
      }
      
      // 方法2: 查找section标签的数量（reveal.js格式）
      const sectionMatches = htmlContent.match(/<section[^>]*>/gi)
      if (sectionMatches && sectionMatches.length > 0) {
        return sectionMatches.length
      }
      
      // 方法3: 查找data-total-slides属性
      const totalSlidesMatch = htmlContent.match(/data-total-slides\s*=\s*['"'](\d+)['"']/i)
      if (totalSlidesMatch && totalSlidesMatch[1]) {
        return parseInt(totalSlidesMatch[1])
      }
      
      // 方法4: 查找class="slide"的元素数量
      const slideClassMatches = htmlContent.match(/class\s*=\s*['"'][^'"]*slide[^'"]*['"']/gi)
      if (slideClassMatches && slideClassMatches.length > 0) {
        return slideClassMatches.length
      }
      
      console.log('未找到有效的页数信息')
      return 0
    } catch (error) {
      console.error('解析HTML页数时出错:', error)
      return 0
    }
  },

  /**
   * web-view加载完成
   */
  onWebViewLoad() {
    console.log('Web-view加载完成')
    console.log('Web-view加载的URL:', this.data.pptURL)
    
    // 只有在还没有关闭加载状态时才设置
    if (this.data.isLoading) {
    this.setData({
      isLoading: false,
      loadError: false
    })
    }
    
    console.log('Web-view加载完成后状态 - 当前页:', this.data.currentSlide, '总页数:', this.data.totalSlides)
  },

  /**
   * web-view消息处理
   */
  onWebViewMessage(e) {
    console.log('接收到webview消息:', e.detail.data)
    const messages = e.detail.data
    if (messages && messages.length > 0) {
      const message = messages[0]
      if (message.type === 'slideInfo') {
        this.setData({
          currentSlide: message.current || this.data.currentSlide,
          totalSlides: message.total || this.data.totalSlides
        })
      }
    }
  },

  /**
   * web-view加载失败
   */
  onWebViewError(e) {
    console.error('Web-view加载失败:', e)
    console.error('失败详情:', e.detail)
    this.setData({
      isLoading: false,
      loadError: true
    })
    
    // 显示错误提示
    wx.showToast({
      title: '加载失败，请检查网络',
      icon: 'none',
      duration: 3000
    })
  },

  /**
   * 获取PPT信息
   */
  getPPTInfo() {
    console.log('尝试获取PPT信息')
    
    // 从PPT URL中提取文件名
    const pptURL = this.data.pptURL
    const urlParts = pptURL.split('/')
    const filename = urlParts[urlParts.length - 1]
    
    if (!filename) {
      console.error('无法从URL中提取文件名:', pptURL)
      return
    }
    
    const { getBaseURLSync } = require('../../../utils/config')
    const baseURL = getBaseURLSync()
    const infoURL = `${baseURL}/ppt/info/${filename}`
    
    console.log('请求PPT信息:', infoURL)
    
    wx.request({
      url: infoURL,
      method: 'GET',
      success: (res) => {
        console.log('PPT信息获取成功:', res.data)
        if (res.statusCode === 200 && res.data && res.data.data) {
          const data = res.data.data
          const totalSlides = data.total_slides || 1
          
          this.setData({
            totalSlides: totalSlides
          })
          
          console.log('更新总页数为:', totalSlides)
        }
      },
      fail: (error) => {
        console.error('获取PPT信息失败:', error)
        // 保持默认值，不影响预览功能
      }
    })
  },

  /**
   * 重新加载
   */
  reloadPPT() {
    this.setData({
      isLoading: true,
      loadError: false
    })
    
    // 触发webview重新加载
    setTimeout(() => {
      this.setData({
        pptURL: this.data.pptURL + '?t=' + Date.now()
      })
    }, 100)
  },

  /**
   * 切换全屏模式
   */
  toggleFullScreen() {
    const isFullScreen = !this.data.isFullScreen
    this.setData({ 
      isFullScreen: isFullScreen,
      showActionSheet: false 
    })
    
    // 隐藏/显示导航栏
    if (isFullScreen) {
      wx.hideNavigationBarLoading()
    }
    
    wx.showToast({
      title: isFullScreen ? '已进入全屏' : '已退出全屏',
      icon: 'success',
      duration: 1500
    })
  },

  /**
   * 上一页
   */
  previousSlide() {
    if (this.data.currentSlide > 1) {
      const newSlide = this.data.currentSlide - 1
      this.setData({ currentSlide: newSlide })
      this.navigateToSlide(newSlide)
    }
  },

  /**
   * 下一页
   */
  nextSlide() {
    if (this.data.currentSlide < this.data.totalSlides) {
      const newSlide = this.data.currentSlide + 1
      this.setData({ currentSlide: newSlide })
      this.navigateToSlide(newSlide)
    }
  },

  /**
   * 导航到指定页
   */
  navigateToSlide(slideNumber) {
    // 这里需要webview内的PPT支持页面跳转
    // 可以通过URL参数或postMessage实现
    console.log('导航到第', slideNumber, '页')
    
    // 示例：如果PPT支持URL参数
    const baseURL = this.data.pptURL.split('#')[0].split('?')[0]
    const newURL = `${baseURL}#slide=${slideNumber}`
    this.setData({ pptURL: newURL })
  },

  /**
   * 缩小
   */
  zoomOut() {
    const newZoom = Math.max(50, this.data.zoomLevel - 25)
    this.setData({ zoomLevel: newZoom })
    this.applyZoom(newZoom)
    
    wx.showToast({
      title: `缩小到 ${newZoom}%`,
      icon: 'none',
      duration: 1000
    })
  },

  /**
   * 放大
   */
  zoomIn() {
    const newZoom = Math.min(200, this.data.zoomLevel + 25)
    this.setData({ zoomLevel: newZoom })
    this.applyZoom(newZoom)
    
    wx.showToast({
      title: `放大到 ${newZoom}%`,
      icon: 'none',
      duration: 1000
    })
  },

  /**
   * 重置缩放
   */
  resetZoom() {
    this.setData({ zoomLevel: 100 })
    this.applyZoom(100)
    
    wx.showToast({
      title: '已重置缩放',
      icon: 'success',
      duration: 1000
    })
  },

  /**
   * 应用缩放
   */
  applyZoom(zoom) {
    // 这里需要通过webview通信实现缩放
    // 或者通过CSS transform实现
    console.log('应用缩放:', zoom + '%')
  },

  /**
   * 显示更多操作
   */
  showMoreActions() {
    this.setData({ showActionSheet: true })
  },

  /**
   * 隐藏更多操作
   */
  hideMoreActions() {
    this.setData({ showActionSheet: false })
  },

  /**
   * 显示分享弹窗
   */
  showShareModal() {
    this.setData({ 
      showShareModal: true,
      showActionSheet: false 
    })
  },

  /**
   * 隐藏分享弹窗
   */
  hideShareModal() {
    this.setData({ showShareModal: false })
  },

  /**
   * 分享PPT
   */
  sharePPT() {
    this.showShareModal()
  },

  /**
   * 复制链接
   */
  copyLink() {
          wx.setClipboardData({
            data: this.data.pptURL,
            success: () => {
              wx.showToast({
          title: '链接已复制到剪贴板',
          icon: 'success',
          duration: 2000
              })
        this.hideShareModal()
        this.hideMoreActions()
      },
      fail: () => {
        wx.showToast({
          title: '复制失败',
          icon: 'none'
          })
      }
    })
  },

  /**
   * 生成二维码
   */
  generateQRCode() {
    wx.showToast({
      title: '二维码功能开发中',
      icon: 'none',
      duration: 2000
    })
    this.hideShareModal()
  },

  /**
   * 报告问题
   */
  reportIssue() {
    wx.showModal({
      title: '报告问题',
      content: '请描述您遇到的问题，我们会尽快处理。',
      editable: true,
      placeholderText: '请输入问题描述...',
      success: (res) => {
        if (res.confirm) {
          wx.showToast({
            title: '问题已提交，感谢反馈',
            icon: 'success',
            duration: 2000
          })
        }
      }
    })
    this.hideMoreActions()
  },

  /**
   * 下载PPT
   */
  downloadPPT() {
    wx.showLoading({
      title: '准备下载...'
    })

    // 隐藏所有弹窗
    this.setData({ 
      showShareModal: false,
      showActionSheet: false 
    })

    wx.downloadFile({
      url: this.data.pptURL,
      success: (res) => {
        wx.hideLoading()
        if (res.statusCode === 200) {
          // 尝试保存到本地
          wx.saveFileToDisk({
            filePath: res.tempFilePath,
            success: () => {
              wx.showToast({
                title: '下载成功',
                icon: 'success',
                duration: 2000
              })
            },
            fail: () => {
              // 如果保存失败，尝试打开文档
              wx.openDocument({
                filePath: res.tempFilePath,
                fileType: 'html',
                success: () => {
                  console.log('打开文档成功')
                  wx.showToast({
                    title: '文件已打开',
                    icon: 'success'
                  })
                },
                fail: (error) => {
                  console.error('打开文档失败:', error)
                  wx.showModal({
                    title: '下载完成',
                    content: '文件已下载但无法直接打开，请在文件管理器中查看',
                    showCancel: false
                  })
                }
              })
            }
          })
        } else {
          wx.showToast({
            title: '下载失败',
            icon: 'none',
            duration: 2000
          })
        }
      },
      fail: (error) => {
        wx.hideLoading()
        console.error('下载失败:', error)
        wx.showToast({
          title: '网络错误，下载失败',
          icon: 'none',
          duration: 2000
        })
      }
    })
  },

  /**
   * 返回上一页
   */
  goBack() {
    if (this.data.isFullScreen) {
      this.toggleFullScreen()
      return
    }
    
    wx.navigateBack({
      fail: () => {
        // 如果没有上一页，跳转到首页
        wx.switchTab({
          url: '/pages/index/index'
        })
      }
    })
  },

  /**
   * 页面卸载
   */
  onUnload() {
    // 清理资源
    this.setData({
      showActionSheet: false,
      showShareModal: false,
      isFullScreen: false
    })
  },

  /**
   * 页面分享
   */
  onShareAppMessage() {
    return {
      title: this.data.title,
      path: `/pages/ppt/preview/preview?url=${encodeURIComponent(this.data.pptURL)}&title=${encodeURIComponent(this.data.title)}`,
      imageUrl: '' // 可以设置分享图片
    }
  }
}) 