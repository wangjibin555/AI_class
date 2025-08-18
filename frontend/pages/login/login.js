Page({
  data: {
    isLoading: false,
    showPrivacy: false
  },

  onLoad(options) {
    console.log('登录页面加载成功')
    
    // 检查是否已经登录
    const token = wx.getStorageSync('access_token')
    if (token) {
      this.redirectToMain()
      return
    }

    // 如果有重定向参数，保存起来
    if (options.redirect) {
      this.setData({
        redirectUrl: decodeURIComponent(options.redirect)
      })
    }
  },

  onShow() {
    console.log('登录页面显示')
    
    // 每次显示时检查登录状态
    const token = wx.getStorageSync('access_token')
    if (token) {
      this.redirectToMain()
    }
  },

  /**
   * 获取用户信息授权回调
   */
  onGetUserInfo(e) {
    console.log('获取用户信息:', e.detail)

    if (e.detail.errMsg === 'getUserInfo:ok') {
      // 用户同意授权，开始登录流程
      this.startWechatLogin(e.detail)
    } else {
      // 用户拒绝授权
      wx.showToast({
        title: '需要授权才能登录',
        icon: 'none',
        duration: 2000
      })
    }
  },

  /**
   * 开始微信登录流程
   */
  startWechatLogin(userInfo = {}) {
    this.setData({ isLoading: true })

    // 首先获取微信登录code
    wx.login({
      success: (res) => {
        if (res.code) {
          console.log('微信登录code获取成功:', res.code)
          // 调用真实的登录API
          this.callWechatLoginAPI(res.code, userInfo)
        } else {
          console.error('微信登录code获取失败:', res)
          this.showLoginError('获取登录凭证失败')
        }
      },
      fail: (err) => {
        console.error('wx.login失败:', err)
        this.showLoginError('微信登录失败')
      }
    })
  },

  /**
   * 调用真实的微信登录API
   */
  async callWechatLoginAPI(code, userInfo) {
    try {
      const authAPI = require('../../apis/auth.js')
      const auth = require('../../utils/auth.js')
      
      // 调用后端登录API
      const response = await authAPI.wechatLogin(code, userInfo)
      
      console.log('登录API响应:', response)
      
      // 使用新的保存登录信息方法
      const saveSuccess = auth.saveLoginInfo(response)
      
      if (!saveSuccess) {
        throw new Error('保存登录信息失败')
      }
      
      console.log('登录成功:', response)

      // 显示成功提示
      wx.showToast({
        title: '登录成功',
        icon: 'success',
        duration: 1500
      })

      // 延迟跳转，让用户看到成功提示
      setTimeout(() => {
        this.redirectToMain()
      }, 1500)

    } catch (error) {
      console.error('登录失败:', error)
      this.showLoginError('登录失败：' + error.message)
    }
  },

  /**
   * 显示登录错误
   */
  showLoginError(message) {
    this.setData({ isLoading: false })
    
    wx.showModal({
      title: '登录失败',
      content: message,
      showCancel: false,
      confirmText: '确定'
    })
  },

  /**
   * 跳转到主页面
   */
  redirectToMain() {
    this.setData({ isLoading: false })
    
    const redirectUrl = this.data.redirectUrl
    
    if (redirectUrl) {
      // 如果有重定向URL，跳转到指定页面
      wx.redirectTo({
        url: redirectUrl,
        fail: () => {
          // 如果跳转失败，fallback到首页
          this.goToIndex()
        }
      })
    } else {
      // 默认跳转到首页
      this.goToIndex()
    }
  },

  /**
   * 跳转到首页
   */
  goToIndex() {
    wx.switchTab({
      url: '/pages/index/index',
      fail: () => {
        // 如果首页不是tab页面，使用redirectTo
        wx.redirectTo({
          url: '/pages/index/index'
        })
      }
    })
  },

  /**
   * 显示隐私政策
   */
  showPrivacyPolicy() {
    this.setData({ showPrivacy: true })
  },

  /**
   * 显示用户协议
   */
  showUserAgreement() {
    wx.showModal({
      title: '用户协议',
      content: '感谢您使用AI课堂。我们致力于为您提供优质的学习服务，请遵守平台使用规范，共同维护良好的学习环境。',
      showCancel: false,
      confirmText: '我知道了'
    })
  },

  /**
   * 隐藏隐私政策弹窗
   */
  hidePrivacyModal() {
    this.setData({ showPrivacy: false })
  },

  /**
   * 阻止弹窗关闭（点击内容区域）
   */
  preventClose() {
    // 空函数，阻止事件冒泡
  },

  /**
   * 分享功能
   */
  onShareAppMessage() {
    return {
      title: 'AI课堂 - 智能学习，让知识更简单',
      path: '/pages/login/login'
    }
  }
}) 