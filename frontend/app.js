// app.js
const { getBaseURL, getBaseURLSync, fetchServerConfig, getEnvInfo } = require('./utils/config.js')

// 小程序入口
App({
  /**
   * 小程序初始化
   */
  async onLaunch(options) {
    console.log('小程序启动，启动参数:', options)
    
    // 初始化全局配置
    this.initGlobalData()
    
    // 初始化服务器配置
    try {
      console.log('正在获取服务器配置...')
      console.log('环境信息:', getEnvInfo())
      await fetchServerConfig()
      console.log('服务器配置获取成功')
      console.log('最终使用的配置:', getEnvInfo())
    } catch (error) {
      console.warn('服务器配置获取失败，将使用默认配置:', error)
    }
    
    // 初始化认证状态
    await this.initAuth()
  },

  /**
   * 小程序显示
   */
  onShow(options) {
    console.log('小程序显示，显示参数:', options)
  },

  /**
   * 小程序隐藏
   */
  onHide() {
    console.log('小程序隐藏')
  },

  /**
   * 小程序错误
   */
  onError(msg) {
    console.error('小程序错误:', msg)
  },

  /**
   * 页面未找到
   */
  onPageNotFound(res) {
    console.error('页面未找到:', res)
    // 跳转到首页
    wx.redirectTo({
      url: '/pages/index/index'
    })
  },

  /**
   * 内存警告
   */
  onMemoryWarning() {
    console.warn('内存警告')
  },

  /**
   * 未处理的Promise拒绝
   */
  onUnhandledRejection(res) {
    console.error('未处理的Promise拒绝:', res)
  },

  // 检查小程序更新
  checkForUpdate() {
    if (wx.canIUse('getUpdateManager')) {
      const updateManager = wx.getUpdateManager()
      
      updateManager.onCheckForUpdate((res) => {
        if (res.hasUpdate) {
          console.log('发现新版本')
        }
      })

      updateManager.onUpdateReady(() => {
        wx.showModal({
          title: '更新提示',
          content: '新版本已经准备好，是否重启应用？',
          success: (res) => {
            if (res.confirm) {
              updateManager.applyUpdate()
            }
          }
        })
      })

      updateManager.onUpdateFailed(() => {
        wx.showModal({
          title: '更新失败',
          content: '新版本下载失败，请检查网络后重试',
          showCancel: false
        })
      })
    }
  },

  // 初始化全局数据
  initGlobalData() {
    // 获取系统信息，使用新的API替代弃用的wx.getSystemInfoSync
    try {
      // 尝试使用新的API组合获取系统信息
      const windowInfo = wx.getWindowInfo()
      const deviceInfo = wx.getDeviceInfo()
      const appBaseInfo = wx.getAppBaseInfo()
      
      // 组合成类似旧API的格式
      const systemInfo = {
        ...windowInfo,
        ...deviceInfo,
        ...appBaseInfo
      }
      
      this.globalData.systemInfo = systemInfo
    } catch (e) {
      // 如果新API不支持，提供默认值
      console.warn('获取系统信息失败，使用默认值:', e)
      this.globalData.systemInfo = {
        platform: 'unknown',
        system: 'unknown',
        version: '1.0.0',
        SDKVersion: '3.0.0',
        pixelRatio: 2,
        windowWidth: 375,
        windowHeight: 667
      }
    }
    
    // 计算导航栏高度
    try {
      const menuButtonInfo = wx.getMenuButtonBoundingClientRect()
      this.globalData.navBarHeight = menuButtonInfo.top + menuButtonInfo.height + 10
    } catch (e) {
      console.error('获取菜单按钮信息失败:', e)
      this.globalData.navBarHeight = 44 // 默认高度
    }

    // 设置API基础URL
    // 设置初始baseURL（同步获取）
    this.globalData.baseURL = getBaseURLSync()
    
    // 异步获取最新配置并更新globalData
    getBaseURL().then(baseURL => {
      this.globalData.baseURL = baseURL
      console.log('globalData.baseURL 已更新为:', baseURL)
    }).catch(err => {
      console.warn('获取baseURL失败，保持默认配置:', err)
    })
  },

  // 初始化认证状态
  async initAuth() {
    try {
      console.log('🔐 初始化认证状态...')
      
      const auth = require('./utils/auth.js')
      
      // 使用新的认证初始化方法
      const authSuccess = await auth.initAuth()
      
      if (authSuccess) {
        console.log('✅ 认证状态初始化成功')
        this.globalData.isLoggedIn = true
        this.globalData.userInfo = auth.getUserInfo()
        
        // 获取token用于全局使用
        this.globalData.token = auth.getToken()
        
        console.log('用户已登录:', this.globalData.userInfo)
      } else {
        console.log('❌ 认证状态初始化失败或用户未登录')
        this.globalData.isLoggedIn = false
        this.globalData.userInfo = null
        this.globalData.token = null
      }
    } catch (error) {
      console.error('初始化认证状态失败:', error)
      this.globalData.isLoggedIn = false
      this.globalData.userInfo = null
      this.globalData.token = null
    }
  },

  // 初始化错误处理
  initErrorHandler() {
    // 监听未捕获的Promise rejection
    wx.onUnhandledRejection((res) => {
      console.error('未处理的Promise拒绝:', res)
      // 在开发环境显示错误信息
      if (this.globalData.isDev) {
        wx.showModal({
          title: '错误',
          content: `未处理的Promise拒绝: ${res.reason}`,
          showCancel: false
        })
      }
    })

    // 监听内存警告
    wx.onMemoryWarning(() => {
      console.warn('内存警告，尝试清理缓存')
      // 清理一些缓存数据
      this.cleanupCache()
    })
  },

  // 清理缓存
  cleanupCache() {
    try {
      // 清理过期的缓存数据
      const keys = wx.getStorageInfoSync().keys
      keys.forEach(key => {
        if (key.startsWith('cache_')) {
          const data = wx.getStorageSync(key)
          if (data && data.expireTime && Date.now() > data.expireTime) {
            wx.removeStorageSync(key)
          }
        }
      })
      console.log('缓存清理完成')
    } catch (error) {
      console.error('清理缓存失败:', error)
    }
  },

  // 用户登录
  login(userInfo) {
    this.globalData.isLoggedIn = true
    this.globalData.userInfo = userInfo
    
    // 保存到本地存储
    wx.setStorageSync('userInfo', userInfo)
    
    // 触发登录事件
    this.onUserLogin(userInfo)
  },

  // 用户登出
  logout() {
    this.globalData.isLoggedIn = false
    this.globalData.userInfo = null
    this.globalData.token = null
    
    // 清除本地存储
    wx.removeStorageSync('token')
    wx.removeStorageSync('userInfo')
    
    // 触发登出事件
    this.onUserLogout()
  },

  // 设置Token
  setToken(token) {
    this.globalData.token = token
    wx.setStorageSync('token', token)
  },

  // 用户登录回调
  onUserLogin(userInfo) {
    console.log('用户登录成功:', userInfo.nickname)
    
    // 这里可以添加登录后的逻辑
    // 比如：上报用户行为、初始化用户数据等
  },

  // 用户登出回调
  onUserLogout() {
    console.log('用户已登出')
    
    // 这里可以添加登出后的逻辑
    // 比如：清理缓存、重置应用状态等
  },

  // 检查网络状态
  checkNetworkStatus() {
    return new Promise((resolve) => {
      wx.getNetworkType({
        success: (res) => {
          if (res.networkType === 'none') {
            wx.showToast({
              title: '网络连接异常',
              icon: 'none'
            })
            resolve(false)
          } else {
            resolve(true)
          }
        },
        fail: () => {
          resolve(false)
        }
      })
    })
  },

  // 显示加载提示
  showLoading(title = '加载中...') {
    wx.showLoading({
      title: title,
      mask: true
    })
  },

  // 隐藏加载提示
  hideLoading() {
    wx.hideLoading()
  },

  // 显示成功提示
  showSuccess(title) {
    wx.showToast({
      title: title,
      icon: 'success',
      duration: 2000
    })
  },

  // 显示错误提示
  showError(title) {
    wx.showToast({
      title: title,
      icon: 'none',
      duration: 3000
    })
  },

  // 全局数据
  globalData: {
    // 基础信息
    appName: 'AI课堂',
    version: '1.0.0',
    isDev: false,
    
    // 用户信息
    isLoggedIn: false,
    userInfo: null,
    token: null,
    
    // 系统信息
    systemInfo: null,
    navBarHeight: 0,
    
    // API配置
    baseURL: '',
    
    // WebSocket
    socketUrl: '',
    socket: null,
    
    // 缓存配置
    cacheExpire: 30 * 60 * 1000, // 30分钟
    
    // 业务配置
    maxFileSize: 10 * 1024 * 1024, // 10MB
    supportedFileTypes: ['pdf', 'doc', 'docx', 'txt'],
    
    // 颜色主题
    colors: {
      primary: '#4A90E2',
      secondary: '#357ABD',
      success: '#28A745',
      warning: '#FFC107',
      danger: '#DC3545',
      info: '#17A2B8',
      light: '#F8F9FA',
      dark: '#343A40',
      muted: '#999999'
    }
  }
})