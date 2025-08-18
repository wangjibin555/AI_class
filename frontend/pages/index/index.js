const authAPI = require('../../apis/auth.js')

Page({
  data: {
    isLoggedIn: false,
    userInfo: {
      nickname: '测试用户',
      credits: 0
    },
    loading: false,
    // 小程序本地资源使用绝对路径（以/开头），根目录为小程序代码根
    // 图片放在 frontend/images/ 下时，运行时路径应写成 /images/xxx.png
    heroImage: '/images/首页.png'
  },

  onLoad(options) {
    console.log('首页加载成功')
    this.checkLoginStatus()
  },

  onShow() {
    console.log('首页显示')
    this.checkLoginStatus()
  },

  // 检查登录状态
  async checkLoginStatus() {
    try {
      const token = wx.getStorageSync('access_token')
      
      if (token) {
        // 如果有token，从服务器获取最新用户信息
        await this.loadUserInfo()
      } else {
        this.setData({
          isLoggedIn: false,
          userInfo: {
            nickname: '测试用户',
            credits: 0
          }
        })
      }
    } catch (e) {
      console.error('检查登录状态失败:', e)
      this.setData({
        isLoggedIn: false,
        userInfo: {
          nickname: '测试用户',
          credits: 0
        }
      })
    }
  },

  // 加载用户信息
  async loadUserInfo() {
    this.setData({ loading: true })
    
    try {
      const response = await authAPI.getProfile()
      console.log('首页获取用户信息:', response)
      
      // 处理响应数据结构
      const userData = response.data || response
      
      this.setData({
        isLoggedIn: true,
        userInfo: {
          nickname: userData.nickname || '微信用户',
          credits: userData.credits || 0,
          avatar_url: userData.avatar_url,
          vip_level: userData.vip_level || 0
        }
      })
      
      // 同步更新本地存储
      wx.setStorageSync('user_info', userData)
      
    } catch (error) {
      console.error('加载用户信息失败:', error)
      
      // 如果API调用失败，尝试使用本地存储的数据
      const localUserInfo = wx.getStorageSync('user_info')
      if (localUserInfo) {
        this.setData({
          isLoggedIn: true,
          userInfo: {
            nickname: localUserInfo.nickname || '微信用户',
            credits: localUserInfo.credits || 0,
            avatar_url: localUserInfo.avatar_url,
            vip_level: localUserInfo.vip_level || 0
          }
        })
      } else {
        // 如果本地也没有数据，设置为未登录状态
        this.setData({
          isLoggedIn: false,
          userInfo: {
            nickname: '测试用户',
            credits: 0
          }
        })
      }
    } finally {
      this.setData({ loading: false })
    }
  },

  // 跳转到登录页
  goToLogin() {
    console.log('跳转到登录页')
    wx.navigateTo({
      url: '/pages/login/login'
    })
  },

  // 跳转到个人中心
  goToProfile() {
    console.log('跳转到个人中心')
    wx.navigateTo({
      url: '/pages/profile/profile'
    })
  },

  // 创建课件
  createCourse() {
    console.log('创建课件')
    if (!this.data.isLoggedIn) {
      wx.showToast({
        title: '请先登录',
        icon: 'none'
      })
      this.goToLogin()
      return
    }
    
    // 跳转到创建课程页面
    wx.navigateTo({
      url: '/pages/course/create/create'
    })
  },

  // 我的课件
  myCourses() {
    console.log('我的课件')
    if (!this.data.isLoggedIn) {
      wx.showToast({
        title: '请先登录',
        icon: 'none'
      })
      this.goToLogin()
      return
    }
    
    wx.navigateTo({
      url: '/pages/course/list/list'
    })
  },

  // 学习记录
  studyHistory() {
    console.log('学习记录')
    if (!this.data.isLoggedIn) {
      wx.showToast({
        title: '请先登录',
        icon: 'none'
      })
      this.goToLogin()
      return
    }
    
    // 跳转到学习记录页面
    wx.navigateTo({
      url: '/pages/profile/learning/learning'
    })
  },

  // AI助手
  aiAssistant() {
    console.log('AI助手')
    if (!this.data.isLoggedIn) {
      wx.showToast({
        title: '请先登录',
        icon: 'none'
      })
      this.goToLogin()
      return
    }
    
    // 跳转到AI助手页面
    wx.navigateTo({
      url: '/pages/ai-assistant/ai-assistant'
    })
  },

  // 分享功能
  onShareAppMessage() {
    return {
      title: 'AI课堂 - 智能学习平台',
      path: '/pages/index/index'
    }
  }
}) 