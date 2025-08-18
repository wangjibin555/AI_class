// pages/course/detail/detail.js
const courseAPI = require('../../../apis/course.js')
const userAPI = require('../../../apis/user.js')
const auth = require('../../../utils/auth.js')
const request = require('../../../utils/request.js')

Page({
  /**
   * 页面的初始数据
   */
  data: {
    courseId: null,
    courseInfo: null,
    slides: [],
    quiz: null,
    isLoading: false,
    isPlaying: false,
    currentSlide: 0,
    totalSlides: 0,
    
    // 用户相关
    userInfo: null,
    isOwner: false,
    isBookmarked: false,
    learningProgress: 0,
    
    // 界面状态
    showSlideList: false,
    showQuizInfo: false,
    showShareDialog: false,
    
    // 播放相关
    audioContext: null,
    isAudioPlaying: false,
    audioDuration: 0,
    audioCurrentTime: 0,
    
    // 操作状态
    liking: false,
    bookmarking: false,
    sharing: false,
    
    // 练习生成相关
    showExerciseModal: false,
    generating: false,
    hasExistingQuiz: false, // 是否已有练习
    exerciseOptions: {
      difficulty: 'normal',
      questionCount: 8
    },
    
    // 错误状态
    hasError: false,
    errorMessage: ''
  },

  /**
   * 生命周期函数--监听页面加载
   */
  onLoad(options) {
    console.log('课程详情页加载', options)
    console.log('options.id:', options.id, '类型:', typeof options.id)
    
    // 增强ID验证
    let courseId = options.id
    
    // 检查ID是否存在
    if (!courseId) {
      console.error('URL参数中缺少课程ID')
      this.showErrorAndGoBack('URL参数错误：缺少课程ID')
      return
    }
    
    // 检查ID是否为字符串"undefined"
    if (courseId === 'undefined' || courseId === undefined) {
      console.error('课程ID为undefined:', courseId)
      this.showErrorAndGoBack('课程ID无效：undefined')
      return
    }
    
    // 尝试转换为数字
    const numericId = parseInt(courseId, 10)
    if (isNaN(numericId) || numericId <= 0) {
      console.error('课程ID不是有效的正整数:', courseId)
      this.showErrorAndGoBack(`课程ID格式错误：${courseId}`)
      return
    }
    
    // 使用数字ID
    courseId = numericId
    console.log('验证后的课程ID:', courseId)
    this.setData({ courseId })
    
    // 检查登录状态
    if (!auth.isLoggedIn()) {
      auth.requireLogin()
      return
    }
    
    this.loadCourseDetail()
    this.loadUserInfo()
  },

  /**
   * 显示错误并返回上一页
   */
  showErrorAndGoBack(message) {
    console.error('课程详情页错误:', message)
    this.setData({
      hasError: true,
      errorMessage: message
    })
    
    wx.showModal({
      title: '页面错误',
      content: message,
      showCancel: false,
      confirmText: '返回',
      success: () => {
        wx.navigateBack({
          fail: () => {
            // 如果无法返回，跳转到课程列表页面
            wx.reLaunch({
              url: '/pages/course/list/list'
            })
          }
        })
      }
    })
  },

  /**
   * 查看PPT
   */
  viewPPT() {
    const courseInfo = this.data.courseInfo
    if (!courseInfo || !courseInfo.ppt_file_path) {
      wx.showToast({
        title: 'PPT文件不存在',
        icon: 'none'
      })
      return
    }

    // 从配置中获取文件服务URL
    const { getFileURLSync } = require('../../utils/config.js')
    const fileBaseURL = getFileURLSync()
    const pptFileName = courseInfo.ppt_file_path.replace('ppt/', '')
    const downloadURL = `${fileBaseURL}/api/v1/ppt/download/${pptFileName}`
    // 直接访问PPT HTML文件，而不是通过API
    const previewURL = `${fileBaseURL}/ppt/${pptFileName}`
    
    console.log('准备查看PPT - 下载URL:', downloadURL)
    console.log('准备查看PPT - 预览URL:', previewURL)
    
    // 在小程序中打开PPT（使用web-view或下载）
    wx.showModal({
      title: '查看PPT',
      content: '选择查看方式',
      cancelText: '下载查看',
      confirmText: '在线预览',
      success: (res) => {
        if (res.confirm) {
          // 在线预览 - 跳转到预览页面
          this.previewPPTOnline(previewURL)
        } else if (res.cancel) {
          // 下载查看
          this.downloadPPT(downloadURL)
        }
      }
    })
  },

  /**
   * 在线预览PPT
   */
  previewPPTOnline(pptURL) {
    console.log('准备在线预览PPT:', pptURL)
    
    // 检查URL有效性
    if (!pptURL) {
      wx.showToast({
        title: 'PPT链接无效',
        icon: 'none'
      })
      return
    }
    
    // 跳转到web-view页面显示PPT
    wx.navigateTo({
      url: `/pages/ppt/preview/preview?url=${encodeURIComponent(pptURL)}&title=${encodeURIComponent(this.data.courseInfo?.title || 'PPT预览')}`,
      success: () => {
        console.log('成功跳转到PPT预览页面')
      },
      fail: (error) => {
        console.error('跳转到PPT预览页面失败:', error)
        
        // 如果跳转失败，提供备选方案
        wx.showModal({
          title: '预览页面跳转失败',
          content: '无法打开预览页面，是否复制链接到剪贴板？',
          confirmText: '复制链接',
          cancelText: '取消',
          success: (res) => {
            if (res.confirm) {
              wx.setClipboardData({
                data: pptURL,
                success: () => {
                  wx.showToast({
                    title: '链接已复制，请在浏览器中打开',
                    icon: 'success',
                    duration: 3000
                  })
                }
              })
            }
          }
        })
      }
    })
  },

  /**
   * 下载PPT
   */
  downloadPPT(pptURL) {
    console.log('准备下载PPT:', pptURL)
    
    // 检查是否在开发者工具中运行
    let isDevTools = false
    try {
      const deviceInfo = wx.getDeviceInfo()
      isDevTools = deviceInfo.platform === 'devtools'
    } catch (e) {
      console.warn('获取设备信息失败:', e)
      isDevTools = false
    }
    
    if (isDevTools) {
      wx.showModal({
        title: '开发者工具限制',
        content: '文件下载功能需要在真机上测试，开发者工具暂不支持。是否复制链接到剪贴板？',
        confirmText: '复制链接',
        cancelText: '取消',
        success: (res) => {
          if (res.confirm) {
            wx.setClipboardData({
              data: pptURL,
              success: () => {
                wx.showToast({
                  title: '链接已复制',
                  icon: 'success'
                })
              }
            })
          }
        }
      })
      return
    }
    
    wx.showLoading({
      title: '准备下载...'
    })

    wx.downloadFile({
      url: pptURL,
      success: (res) => {
        wx.hideLoading()
        console.log('下载响应:', res)
        
        if (res.statusCode === 200) {
          console.log('下载成功，文件路径:', res.tempFilePath)
          
          // 下载成功，尝试保存文件
          wx.saveFileToDisk({
            filePath: res.tempFilePath,
            success: () => {
              console.log('文件保存成功')
              wx.showToast({
                title: '下载成功',
                icon: 'success'
              })
            },
            fail: (saveError) => {
              console.log('文件保存失败，尝试打开文件:', saveError)
              
              // 如果保存失败，尝试用浏览器打开
              wx.showModal({
                title: '保存失败',
                content: '无法保存到本地，是否用浏览器打开查看？',
                confirmText: '打开',
                cancelText: '取消',
                success: (modalRes) => {
                  if (modalRes.confirm) {
                    // 复制链接让用户在浏览器中打开
                    wx.setClipboardData({
                      data: pptURL,
                success: () => {
                  wx.showToast({
                          title: '链接已复制，请在浏览器中打开',
                          icon: 'success',
                          duration: 3000
                  })
                      }
                    })
                  }
                }
              })
            }
          })
        } else {
          console.error('下载失败，状态码:', res.statusCode)
          wx.showToast({
            title: `下载失败 (${res.statusCode})`,
            icon: 'none'
          })
        }
      },
      fail: (error) => {
        wx.hideLoading()
        console.error('下载失败:', error)
        wx.showModal({
          title: '下载失败',
          content: '网络下载失败，是否复制链接到剪贴板？',
          confirmText: '复制链接',
          cancelText: '取消',
          success: (res) => {
            if (res.confirm) {
              wx.setClipboardData({
                data: pptURL,
                success: () => {
                  wx.showToast({
                    title: '链接已复制',
                    icon: 'success'
                  })
                }
              })
            }
          }
        })
      }
    })
  },

  /**
   * 直接预览PPT
   */
  previewPPTDirect() {
    const courseInfo = this.data.courseInfo
    if (!courseInfo || !courseInfo.ppt_file_path) {
      wx.showToast({
        title: 'PPT文件不存在',
        icon: 'none'
      })
      return
    }

    // 构建PPT文件的完整URL
    // 从配置中获取文件服务URL
    const { getFileURLSync } = require('../../utils/config.js')
    const baseURL = getFileURLSync()
    const pptFileName = courseInfo.ppt_file_path.replace('ppt/', '')
    const previewURL = `${baseURL}/api/v1/ppt/preview/${pptFileName}`
    
    console.log('直接预览PPT - 预览URL:', previewURL)
    
    // 使用通用的预览方法
    this.previewPPTOnline(previewURL)
  },

  /**
   * 直接下载PPT
   */
  downloadPPTDirect() {
    const courseInfo = this.data.courseInfo
    if (!courseInfo || !courseInfo.ppt_file_path) {
      wx.showToast({
        title: 'PPT文件不存在',
        icon: 'none'
      })
      return
    }

    // 从配置中获取文件服务URL
    const { getFileURLSync } = require('../../utils/config.js')
    const fileBaseURL = getFileURLSync()
    const pptFileName = courseInfo.ppt_file_path.replace('ppt/', '')
    const downloadURL = `${fileBaseURL}/api/v1/ppt/download/${pptFileName}`
    
    console.log('直接下载PPT - 下载URL:', downloadURL)
    
    // 使用通用的下载方法
    this.downloadPPT(downloadURL)
  },

  /**
   * 生命周期函数--监听页面初次渲染完成
   */
  onReady() {
    // 创建音频上下文
    this.setData({
      audioContext: wx.createAudioContext('courseAudio')
    })
  },

  /**
   * 生命周期函数--监听页面显示
   */
  onShow() {
    if (this.data.courseId && auth.isLoggedIn()) {
      // 如果页面已经有课程信息，只刷新练习状态
      if (this.data.courseInfo) {
        console.log('页面已有课程信息，只刷新练习状态')
        // 先基于当前信息快速检查
        this.checkQuizStatus()
        // 然后异步刷新最新状态
        setTimeout(() => {
          this.refreshQuizStatus()
        }, 300)
      } else {
        // 如果没有课程信息，重新加载
        console.log('页面无课程信息，重新加载课程详情')
        this.loadCourseDetail()
      }
    }
  },

  /**
   * 生命周期函数--监听页面隐藏
   */
  onHide() {
    // 暂停音频播放
    if (this.data.isAudioPlaying) {
      this.pauseAudio()
    }
  },

  /**
   * 生命周期函数--监听页面卸载
   */
  onUnload() {
    // 清理音频资源
    if (this.data.audioContext) {
      // 检查是否有destroy方法，如果没有则使用stop方法
      if (typeof this.data.audioContext.destroy === 'function') {
        this.data.audioContext.destroy()
      } else if (typeof this.data.audioContext.stop === 'function') {
        this.data.audioContext.stop()
      }
    }
  },

  /**
   * 页面相关事件处理函数--监听用户下拉动作
   */
  onPullDownRefresh() {
    this.loadCourseDetail()
  },

  /**
   * 页面上拉触底事件的处理函数
   */
  onReachBottom() {
    // 暂无操作
  },

  /**
   * 用户点击右上角分享
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
   * 加载课程详情
   */
  async loadCourseDetail() {
    if (this.data.isLoading) return
    
    const courseId = this.data.courseId
    if (!courseId || courseId === 'undefined') {
      console.error('courseId is null, undefined, or "undefined"')
      wx.showToast({
        title: '课程ID不存在',
        icon: 'none'
      })
      setTimeout(() => {
        wx.navigateBack()
      }, 1500)
      return
    }
    
    this.setData({ isLoading: true })
    
    try {
      console.log('正在加载课程详情，ID:', courseId)
      const response = await courseAPI.getCourseDetail(courseId)
      console.log('课程详情API响应:', response)
      
      // 处理响应数据结构
      let courseInfo = null
      if (response && response.data) {
        courseInfo = response.data
      } else if (response) {
        courseInfo = response
      }
      
      console.log('课程信息:', courseInfo)
      console.log('时间字段检查 - created_at:', courseInfo.created_at, 'updated_at:', courseInfo.updated_at)
      
      // 测试时间格式化
      if (courseInfo.created_at) {
        const testTime = new Date(courseInfo.created_at)
        console.log('时间解析测试 - created_at:', courseInfo.created_at, '解析结果:', testTime, '是否有效:', !isNaN(testTime.getTime()))
      }
      if (courseInfo.updated_at) {
        const testTime = new Date(courseInfo.updated_at)
        console.log('时间解析测试 - updated_at:', courseInfo.updated_at, '解析结果:', testTime, '是否有效:', !isNaN(testTime.getTime()))
      }
      
      if (!courseInfo) {
        throw new Error('课程信息为空')
      }
      
      this.setData({
        courseInfo,
        slides: courseInfo.slides || [],
        quiz: courseInfo.quiz || null,
        totalSlides: courseInfo.slides ? courseInfo.slides.length : 0,
        isOwner: courseInfo.user_id === this.data.userInfo?.id,
        hasExistingQuiz: false // 初始设为false，由API检查更新
      })
      
      // 异步检查练习状态
      this.checkQuizStatus(courseInfo)
      
      // 设置页面标题
      wx.setNavigationBarTitle({
        title: courseInfo.title || '课程详情'
      })
      
      // 加载学习进度
      this.loadLearningProgress()
      
    } catch (error) {
      console.error('加载课程详情失败:', error)
      wx.showToast({
        title: '加载失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ isLoading: false })
      wx.stopPullDownRefresh()
    }
  },

  /**
   * 加载用户信息
   */
  async loadUserInfo() {
    try {
      const response = await userAPI.getProfile()
      this.setData({
        userInfo: response.data || response
      })
    } catch (error) {
      console.error('加载用户信息失败:', error)
    }
  },

  /**
   * 加载学习进度
   */
  async loadLearningProgress() {
    try {
      // TODO: 实现学习进度API
      // const progress = await courseAPI.getLearningProgress(this.data.courseId)
      // this.setData({
      //   learningProgress: progress.progress || 0,
      //   currentSlide: progress.current_slide || 0,
      //   isBookmarked: progress.is_bookmarked || false
      // })
    } catch (error) {
      console.error('加载学习进度失败:', error)
    }
  },

  /**
   * 开始课程学习 - 增强版支持音频模式
   */
  startCourse() {
    console.log('startCourse 被调用')
    console.log('courseInfo:', this.data.courseInfo)
    console.log('slides:', this.data.slides)
    console.log('slides.length:', this.data.slides ? this.data.slides.length : 'slides is null/undefined')
    
    // 检查课程信息
    if (!this.data.courseInfo) {
      console.error('课程信息为空')
      wx.showModal({
        title: '课程信息错误',
        content: '课程信息加载失败，请返回重试',
        showCancel: false,
        confirmText: '确定',
        success: () => {
          wx.navigateBack()
        }
      })
      return
    }

    // 检查幻灯片数据 - 更宽松的检查
    if (!this.data.slides || this.data.slides.length === 0) {
      console.warn('幻灯片数据为空，但允许继续')
      
      // 如果没有幻灯片，但有PPT文件，可以直接跳转到播放器
      if (this.data.courseInfo.ppt_file_url || this.data.courseInfo.ppt_file_path) {
        console.log('没有幻灯片数据，但有PPT文件，直接跳转到播放器')
        this.startAudioLearning()
        return
      }
      
      // 既没有幻灯片也没有PPT文件
      wx.showModal({
        title: '课程内容提示',
        content: '该课程暂无幻灯片内容，是否继续学习？',
        confirmText: '继续',
        cancelText: '取消',
        success: (res) => {
          if (res.confirm) {
            this.startAudioLearning()
          }
        }
      })
      return
    }

    // 检查是否有PPT文件
    if (this.data.courseInfo.ppt_file_url || this.data.courseInfo.ppt_file_path) {
      // 有PPT文件，提供多种学习模式选项
      wx.showActionSheet({
        itemList: ['🎧 音频讲解模式', '📺 预览PPT', '💾 下载PPT', '📖 文本学习模式'],
        success: (res) => {
          switch (res.tapIndex) {
            case 0:
              this.startAudioLearning()
              break
            case 1:
              this.previewPPT()
              break
            case 2:
              this.downloadPPT()
              break
            case 3:
              this.startTextLearning()
              break
          }
        }
      })
    } else {
      // 没有PPT文件，选择学习模式
      wx.showActionSheet({
        itemList: ['🎧 音频讲解模式', '📖 文本学习模式'],
        success: (res) => {
          switch (res.tapIndex) {
            case 0:
              this.startAudioLearning()
              break
            case 1:
              this.startTextLearning()
              break
          }
        }
      })
    }
  },

  /**
   * 开始音频讲解学习 - 新增功能
   */
  async startAudioLearning() {
    console.log('开始音频讲解学习')
    
    // 1. 记录学习开始时间
    const startTime = new Date().getTime()
    
    try {
      // 2. 调用API记录学习开始
      await this.recordLearningStart(startTime, 'audio_ppt')
      
      // 3. 跳转到音频播放器页面
      wx.navigateTo({
        url: `/pages/player/player?courseId=${this.data.courseId}&mode=audio&startTime=${startTime}&slideIndex=0`
      })
    } catch (error) {
      console.error('开始学习记录失败:', error)
      // 即使记录失败也允许继续学习
      wx.navigateTo({
        url: `/pages/player/player?courseId=${this.data.courseId}&mode=audio&slideIndex=0`
      })
    }
  },

  /**
   * 开始文本学习模式
   */
  async startTextLearning() {
    console.log('开始文本学习')
    
    // 1. 记录学习开始时间
    const startTime = new Date().getTime()
    
    try {
      // 2. 调用API记录学习开始
      await this.recordLearningStart(startTime, 'text_only')
      
      // 3. 跳转到文本播放器页面
      wx.navigateTo({
        url: `/pages/player/player?courseId=${this.data.courseId}&mode=text&startTime=${startTime}&slideIndex=0`
      })
    } catch (error) {
      console.error('开始学习记录失败:', error)
      // 即使记录失败也允许继续学习
      wx.navigateTo({
        url: `/pages/player/player?courseId=${this.data.courseId}&mode=text&slideIndex=0`
      })
    }
  },

  /**
   * 记录学习开始 - 新增API调用
   */
  recordLearningStart(startTime, learningType) {
    return request.post('/learning/start', {
          course_id: this.data.courseId,
          start_time: startTime,
          learning_type: learningType
    }, { 
      needAuth: true 
    }).then(data => {
      console.log('学习记录创建成功:', data)
            // 保存学习记录ID到本地
      if (data && data.learning_record_id) {
        wx.setStorageSync(`learning_record_${this.data.courseId}`, data.learning_record_id)
      }
      return data
    }).catch(error => {
          console.error('学习记录API调用失败:', error)
      throw error
    })
  },

  /**
   * 开始学习 - 保持向后兼容
   */
  startLearning() {
    // 调用音频学习模式
    this.startAudioLearning()
  },

  /**
   * 预览PPT
   */
  previewPPT() {
    const pptURL = this.data.courseInfo.ppt_file_url
    if (!pptURL) {
      wx.showToast({
        title: 'PPT文件不存在',
        icon: 'none'
      })
      return
    }
    
    // 在小程序中预览HTML文件
    wx.showModal({
      title: '预览PPT',
      content: 'PPT文件已生成，是否在浏览器中打开预览？',
      success: (res) => {
        if (res.confirm) {
          // 复制链接到剪贴板
          wx.setClipboardData({
            data: pptURL,
            success: () => {
              wx.showToast({
                title: '链接已复制，请在浏览器中打开',
                icon: 'success'
              })
            }
          })
        }
      }
    })
  },

  /**
   * 下载PPT
   */
  downloadPPT() {
    const pptURL = this.data.courseInfo.ppt_file_url
    if (!pptURL) {
      wx.showToast({
        title: 'PPT文件不存在',
        icon: 'none'
      })
      return
    }
    
    wx.showLoading({
      title: '下载中...'
    })

    // 下载PPT文件
    wx.downloadFile({
      url: pptURL,
      success: (res) => {
        wx.hideLoading()
        if (res.statusCode === 200) {
          wx.showToast({
            title: '下载成功',
            icon: 'success'
          })
        } else {
          wx.showToast({
            title: '下载失败',
            icon: 'none'
          })
        }
      },
      fail: (err) => {
        wx.hideLoading()
        console.error('下载PPT失败:', err)
        wx.showToast({
          title: '下载失败',
          icon: 'none'
        })
      }
    })
  },

  /**
   * 查看幻灯片列表
   */
  showSlideList() {
    this.setData({
      showSlideList: !this.data.showSlideList
    })
  },

  /**
   * 选择幻灯片
   */
  selectSlide(e) {
    const slideIndex = e.currentTarget.dataset.index
    this.setData({
      currentSlide: slideIndex,
      showSlideList: false
    })
  },

    /**
   * 查看练习信息
   */
    showQuizInfo() {
      this.setData({
        showQuizInfo: !this.data.showQuizInfo
      })
    },  

      /**
   * 开始练习
   */
  startQuiz() {
    if (!this.data.quiz) {
      wx.showToast({
        title: '该课程暂无练习',
        icon: 'none'
      })
      return
    }
    
    // 检查练习状态
    if (this.data.quiz.status === 'generating') {
      wx.showToast({
        title: '练习正在生成中，请稍后再试',
        icon: 'none',
        duration: 2000
      })
      // 自动刷新状态
      setTimeout(() => {
        this.checkQuizStatus()
      }, 2000)
      return
    }
    
    // 跳转到练习页面
    wx.navigateTo({
      url: `/pages/quiz/quiz?courseId=${this.data.courseId}&quizId=${this.data.quiz.id}`
    })
  },



  /**
   * 点赞/取消点赞
   */
  async toggleLike() {
    if (this.data.liking) return
    
    this.setData({ liking: true })
    
    try {
      // TODO: 实现点赞API
      // await courseAPI.toggleLike(this.data.courseId)
      
      // 更新本地数据
      const courseInfo = { ...this.data.courseInfo }
      courseInfo.like_count = (courseInfo.like_count || 0) + 1
      this.setData({ courseInfo })
      
      wx.showToast({
        title: '点赞成功',
        icon: 'success'
      })
    } catch (error) {
      console.error('点赞失败:', error)
      wx.showToast({
        title: '点赞失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ liking: false })
    }
  },

  /**
   * 收藏/取消收藏
   */
  async toggleBookmark() {
    if (this.data.bookmarking) return
    
    this.setData({ bookmarking: true })
    
    try {
      // TODO: 实现收藏API
      // await courseAPI.toggleBookmark(this.data.courseId)
      
      const isBookmarked = !this.data.isBookmarked
      this.setData({ isBookmarked })
      
      wx.showToast({
        title: isBookmarked ? '收藏成功' : '取消收藏',
        icon: 'success'
      })
    } catch (error) {
      console.error('收藏失败:', error)
      wx.showToast({
        title: '操作失败: ' + error.message,
        icon: 'none'
      })
    } finally {
      this.setData({ bookmarking: false })
    }
  },

  /**
   * 分享课程
   */
  shareCourse() {
    this.setData({
      showShareDialog: true
    })
  },

  /**
   * 关闭分享对话框
   */
  closeShareDialog() {
    this.setData({
      showShareDialog: false
    })
  },

  /**
   * 刷新练习状态（重新获取课程详情来检查练习）
   */
  async refreshQuizStatus() {
    if (!this.data.courseId) {
      return
    }
    
    try {
      console.log('🔄 通过重新获取课程详情来刷新练习状态，课程ID:', this.data.courseId)
      const response = await courseAPI.getCourseDetail(this.data.courseId)
      console.log('🔄 获取到课程详情:', response)
      
      // 处理响应数据结构
      let courseInfo = null
      if (response && response.data) {
        courseInfo = response.data
      } else if (response) {
        courseInfo = response
      }
      
      if (courseInfo) {
        // 检查练习状态
        this.checkQuizStatus(courseInfo)
        console.log('🔄 练习状态刷新完成')
      }
    } catch (error) {
      console.log('🔄 刷新练习状态失败，使用当前缓存状态:', error.message)
      // 如果API调用失败，使用当前已有的状态，不做修改
    }
  },

  /**
   * 检查练习状态 - 使用专门的API
   */
  async checkQuizStatus(courseInfo = null) {
    if (!this.data.courseId) {
      console.log('课程ID不存在，无法检查练习状态')
      return
    }
    
    try {
      console.log('🔍 开始检查练习状态，课程ID:', this.data.courseId)
      
      // 使用专门的API检查练习
      const checkResult = await courseAPI.checkCourseQuiz(this.data.courseId)
      console.log('🔍 API检查结果:', checkResult)
      
      let hasQuiz = false
      let quizId = null
      let quizStatus = 'none'
      
      if (checkResult) {
        // 检查各种可能的响应格式
        if (checkResult.exists === true || checkResult.has_quiz === true) {
          hasQuiz = true
          quizId = checkResult.quiz_id || checkResult.id || checkResult.quizId
          quizStatus = checkResult.status || 'completed'
        } else if (checkResult.quiz_id || checkResult.id || checkResult.quizId) {
          hasQuiz = true
          quizId = checkResult.quiz_id || checkResult.id || checkResult.quizId
          quizStatus = checkResult.status || 'completed'
        }
      }
      
      console.log('🔍 最终检测结果 - hasQuiz:', hasQuiz, 'quizId:', quizId, 'status:', quizStatus)
      
      // 更新状态
      this.setData({
        hasExistingQuiz: hasQuiz,
        quiz: hasQuiz ? { 
          id: quizId, 
          status: quizStatus,
          title: checkResult?.title,
          question_count: checkResult?.question_count
        } : null
      })
      
      return { hasQuiz, quizId }
      
    } catch (error) {
      console.log('🔍 API检查练习失败，回退到本地检查:', error.message)
      
      // 回退到原有的本地检查逻辑
      const course = courseInfo || this.data.courseInfo
      if (!course) {
        console.log('课程信息不存在，无法检查练习状态')
        return
      }
      
      console.log('使用本地信息检查练习状态:', course)
      
      let hasQuiz = false
      let quizId = null
      
      // 多种方式检查练习存在性
      if (course.quiz) {
        if (course.quiz.id) {
          hasQuiz = true
          quizId = course.quiz.id
        } else if (course.quiz.quiz_id) {
          hasQuiz = true
          quizId = course.quiz.quiz_id
        }
      }
      
      // 检查是否有quiz数组或其他格式
      if (!hasQuiz && course.quizzes && Array.isArray(course.quizzes) && course.quizzes.length > 0) {
        hasQuiz = true
        quizId = course.quizzes[0].id || course.quizzes[0].quiz_id
      }
      
      // 检查练习计数
      if (!hasQuiz && course.quiz_count && course.quiz_count > 0) {
        hasQuiz = true
      }
      
      console.log('本地检测结果 - hasQuiz:', hasQuiz, 'quizId:', quizId)
      
      // 更新状态
      this.setData({
        hasExistingQuiz: hasQuiz,
        quiz: hasQuiz ? (this.data.quiz || { id: quizId }) : null
      })
      
      return { hasQuiz, quizId }
    }
  },



  /**
   * 生成练习或查看练习
   */
  generateExercise() {
    if (!this.data.isOwner) {
      wx.showToast({
        title: '只有课程创建者才能生成练习',
        icon: 'none'
      })
      return
    }
    
    // 如果正在生成中，提示用户
    if (this.data.quiz && this.data.quiz.status === 'generating') {
      wx.showToast({
        title: '练习正在生成中，请稍候',
        icon: 'none'
      })
      // 自动刷新状态
      setTimeout(() => {
        this.checkQuizStatus()
      }, 2000)
      return
    }
    
    // 如果已有练习，直接跳转到练习页面
    if (this.data.hasExistingQuiz && this.data.quiz && this.data.quiz.id && this.data.quiz.status === 'completed') {
      this.startQuiz()
      return
    }
    
    // 检查课程是否有source_url
    if (!this.data.courseInfo || !this.data.courseInfo.source_url) {
      wx.showToast({
        title: '课程缺少源URL，无法生成练习',
        icon: 'none'
      })
      return
    }
    
    this.setData({
      showExerciseModal: true
    })
  },

  /**
   * 选择难度等级
   */
  selectDifficulty(e) {
    const difficulty = e.currentTarget.dataset.difficulty
    this.setData({
      'exerciseOptions.difficulty': difficulty
    })
  },

  /**
   * 修改题目数量
   */
  changeQuestionCount(e) {
    const action = e.currentTarget.dataset.action
    let count = this.data.exerciseOptions.questionCount
    
    if (action === 'increase' && count < 20) {
      count++
    } else if (action === 'decrease' && count > 3) {
      count--
    }
    
    this.setData({
      'exerciseOptions.questionCount': count
    })
  },

  /**
   * 取消生成练习
   */
  cancelGenerateExercise() {
    this.setData({
      showExerciseModal: false
    })
  },

  /**
   * 确认生成练习
   */
  async confirmGenerateExercise() {
    if (this.data.generating) {
      return
    }

    this.setData({
      generating: true
    })

    try {
      const options = {
        title: `${this.data.courseInfo.title} - 练习题`,
        difficulty: this.data.exerciseOptions.difficulty,
        questionCount: this.data.exerciseOptions.questionCount
      }

      console.log('开始生成练习，选项：', options)
      console.log('课程ID：', this.data.courseId)
      
      const response = await courseAPI.generateExercise(this.data.courseId, options)
      
      console.log('练习生成API响应：', response)
      console.log('响应类型：', typeof response)
      console.log('响应键：', Object.keys(response || {}))
      
      // 安全地获取quiz_id
      let quizId = null
      if (response) {
        // 方式1: response.quiz_id (直接在根部)
        if (response.quiz_id) {
          quizId = response.quiz_id
        }
        // 方式2: response.data.quiz_id (嵌套在data中)
        else if (response.data && response.data.quiz_id) {
          quizId = response.data.quiz_id
        }
        // 方式3: response.id (使用id字段)
        else if (response.id) {
          quizId = response.id
        }
        // 方式4: response.data.id (嵌套的id)
        else if (response.data && response.data.id) {
          quizId = response.data.id
        }
      }
      
      console.log('提取的quizId:', quizId)
      
      if (!quizId) {
        console.error('无法从响应中提取quiz_id，完整响应：', JSON.stringify(response, null, 2))
        throw new Error(`响应中缺少quiz_id。响应结构：${JSON.stringify(response)}`)
      }
      
      wx.showToast({
        title: '练习生成成功！',
        icon: 'success'
      })

      // 关闭弹窗并更新状态
      this.setData({
        showExerciseModal: false,
        generating: false,
        hasExistingQuiz: true, // 标记已有练习
        quiz: { id: quizId, status: 'completed' } // 更新quiz信息
      })
      
      // 立即刷新练习状态以获取最新数据
      setTimeout(() => {
        this.checkQuizStatus()
      }, 1000)
      
      // 同时更新courseInfo中的quiz信息，确保下次检查时能正确识别
      if (this.data.courseInfo) {
        const updatedCourseInfo = {
          ...this.data.courseInfo,
          quiz: { id: quizId },
          quiz_count: 1
        }
        this.setData({
          courseInfo: updatedCourseInfo
        })
      }

      // 跳转到练习页面，传递用户选择的参数
      setTimeout(() => {
        wx.navigateTo({
          url: `/pages/quiz/quiz?courseId=${this.data.courseId}&quizId=${quizId}&difficulty=${this.data.exerciseOptions.difficulty}&questionCount=${this.data.exerciseOptions.questionCount}`
        })
      }, 1500)

    } catch (error) {
      console.error('生成练习失败：', error)
      
      let errorMessage = '生成练习失败'
      let showRetryButton = false
      
      // 特殊处理HTTP 409：已存在练习
      if (error.message && error.message.includes('HTTP 409')) {
        console.log('处理HTTP 409错误，错误对象:', error)
        
        // 尝试从错误对象中获取quiz_id
        let quizId = null
        
        // 从responseData中获取quiz_id - 支持多种响应格式
        if (error.responseData) {
          // 尝试新的统一响应格式：{code, message, data: {quiz_id}}
          if (error.responseData.data && error.responseData.data.quiz_id) {
            quizId = error.responseData.data.quiz_id
          }
          // 尝试直接在响应根部的格式：{error, quiz_id}
          else if (error.responseData.quiz_id) {
            quizId = error.responseData.quiz_id
          }
        }
        
        console.log('从错误对象中获取的quizId:', quizId)
        
        // 标记练习已存在
        this.setData({
          hasExistingQuiz: true,
          quiz: { id: quizId },
          generating: false,
          showExerciseModal: false
        })
        
        // 同时更新courseInfo中的quiz信息
        if (this.data.courseInfo && quizId) {
          const updatedCourseInfo = {
            ...this.data.courseInfo,
            quiz: { id: quizId },
            quiz_count: 1
          }
          this.setData({
            courseInfo: updatedCourseInfo
          })
        }
        
        wx.showModal({
          title: '练习已存在',
          content: '该课程已有AI生成的练习，是否查看现有练习？',
          showCancel: true,
          cancelText: '取消',
          confirmText: '查看练习',
          success: (res) => {
            if (res.confirm && quizId) {
              // 跳转到现有练习，传递课程ID和练习ID
              wx.navigateTo({
                url: `/pages/quiz/quiz?courseId=${this.data.courseId}&quizId=${quizId}`
              })
            }
          }
        })
        return
      }
      
      // 解析其他类型的错误
      if (error.message) {
        if (error.message.includes('timeout') || error.message.includes('超时')) {
          errorMessage = '网络超时，系统正在重试中，请稍后再试'
          showRetryButton = true
        } else if (error.message.includes('network') || error.message.includes('网络')) {
          errorMessage = '网络连接异常，请检查网络后重试'
          showRetryButton = true
        } else if (error.message.includes('HTTP 500')) {
          errorMessage = '服务器暂时繁忙，请稍后重试'
          showRetryButton = true
        } else {
          errorMessage = error.message
        }
      } else if (error.error) {
        if (typeof error.error === 'string') {
          if (error.error.includes('timeout') || error.error.includes('超时')) {
            errorMessage = '生成时间较长，系统正在后台处理，请稍后查看结果'
          } else if (error.error.includes('工作流调用失败')) {
            errorMessage = 'AI服务暂时繁忙，请稍后重试'
            showRetryButton = true
          } else {
            errorMessage = error.error
          }
        } else {
          errorMessage = '生成练习失败，请稍后重试'
        }
      }
      
      // 显示错误信息
      if (showRetryButton) {
        wx.showModal({
          title: '生成练习失败',
          content: errorMessage,
          showCancel: true,
          cancelText: '取消',
          confirmText: '重试',
          success: (res) => {
            if (res.confirm) {
              // 用户选择重试
              this.confirmGenerateExercise()
              return
            }
            // 用户取消，重置状态
            this.setData({
              generating: false
            })
          }
        })
      } else {
        wx.showToast({
          title: errorMessage,
          icon: 'none',
          duration: 4000
        })
        
        this.setData({
          generating: false
        })
      }
    }
  },

  /**
   * 删除课程
   */
  deleteCourse() {
    if (!this.data.isOwner) {
      wx.showToast({
        title: '只有课程创建者才能删除',
        icon: 'none'
      })
      return
    }
    
    wx.showModal({
      title: '确认删除',
      content: '删除后无法恢复，确定要删除这门课程吗？',
      success: async (res) => {
        if (res.confirm) {
          try {
            await courseAPI.deleteCourse(this.data.courseId)
            wx.showToast({
              title: '删除成功',
              icon: 'success'
            })
            
            setTimeout(() => {
              wx.navigateBack()
            }, 1500)
          } catch (error) {
            console.error('删除失败:', error)
            wx.showToast({
              title: '删除失败: ' + error.message,
              icon: 'none'
            })
          }
        }
      }
    })
  },

  /**
   * 播放音频
   */
  playAudio() {
    if (!this.data.audioContext) return
    
    const { slides, currentSlide } = this.data
    if (!slides[currentSlide] || !slides[currentSlide].audio_url) {
      wx.showToast({
        title: '该幻灯片暂无音频',
        icon: 'none'
      })
      return
    }
    
    this.data.audioContext.src = slides[currentSlide].audio_url
    this.data.audioContext.play()
    this.setData({ isAudioPlaying: true })
  },

  /**
   * 暂停音频
   */
  pauseAudio() {
    if (this.data.audioContext) {
      this.data.audioContext.pause()
      this.setData({ isAudioPlaying: false })
    }
  },

  /**
   * 格式化时间 - 显示具体年月日时分
   */
  formatTime(timestamp) {
    if (!timestamp) return ''
    
    try {
      const time = new Date(timestamp)
      
      // 检查时间是否有效
      if (isNaN(time.getTime())) {
        console.error('无效的时间格式:', timestamp)
        return '时间格式错误'
      }
      
      // 显示具体的年月日时分格式
      const year = time.getFullYear()
      const month = String(time.getMonth() + 1).padStart(2, '0')
      const day = String(time.getDate()).padStart(2, '0')
      const hours = String(time.getHours()).padStart(2, '0')
      const minutes = String(time.getMinutes()).padStart(2, '0')
      
      return `${year}-${month}-${day} ${hours}:${minutes}`
    } catch (error) {
      console.error('时间格式化错误:', error, '原始时间:', timestamp)
      return '时间解析错误'
    }
  },

  /**
   * 格式化时长
   */
  formatDuration(seconds) {
    if (!seconds || seconds === 0) return '0分钟'
    
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    
    if (hours > 0) {
      return `${hours}小时${minutes}分钟`
    } else {
      return `${minutes}分钟`
    }
  },

  /**
   * 获取状态文本
   */
  getStatusText(status) {
    const statusMap = {
      'draft': '草稿',
      'processing': '处理中',
      'completed': '已完成',
      'failed': '失败'
    }
    return statusMap[status] || '未知状态'
  },

  /**
   * 获取状态颜色
   */
  getStatusColor(status) {
    const colorMap = {
      'generating': '#FF9500',
      'completed': '#34C759',
      'failed': '#FF3B30'
    }
    return colorMap[status] || '#666666'
  },

  /**
   * 返回上一页
   */
  goBack() {
    wx.navigateBack({
      delta: 1
    })
  },

  /**
   * 回到首页
   */
  goHome() {
    wx.switchTab({
      url: '/pages/index/index'
    })
  }
})