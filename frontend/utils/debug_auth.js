/**
 * 认证调试工具
 * 用于检查和调试认证状态
 */

/**
 * 检查当前认证状态
 */
function checkAuthStatus() {
  console.log('🔐 检查认证状态...')
  
  try {
    // 检查token
    const token = wx.getStorageSync('access_token')
    const refreshToken = wx.getStorageSync('refresh_token')
    const expiresAt = wx.getStorageSync('token_expires')
    const userInfo = wx.getStorageSync('user_info')
    
    console.log('📋 认证信息：')
    console.log('  - access_token:', token ? `${token.substring(0, 20)}...` : 'null')
    console.log('  - refresh_token:', refreshToken ? `${refreshToken.substring(0, 20)}...` : 'null')
    console.log('  - expires_at:', expiresAt ? new Date(expiresAt).toLocaleString() : 'null')
    console.log('  - user_info:', userInfo ? '存在' : 'null')
    
    // 检查是否过期
    const now = Date.now()
    const isExpired = expiresAt ? now >= expiresAt : true
    
    console.log('⏰ 时间检查：')
    console.log('  - 当前时间:', new Date(now).toLocaleString())
    console.log('  - 是否过期:', isExpired)
    
    if (token && !isExpired) {
      console.log('✅ 认证状态：已登录且token有效')
      return {
        isLoggedIn: true,
        token: token,
        userInfo: userInfo
      }
    } else if (token && isExpired) {
      console.log('⚠️ 认证状态：token已过期')
      return {
        isLoggedIn: false,
        reason: 'token_expired',
        token: token
      }
    } else {
      console.log('❌ 认证状态：未登录')
      return {
        isLoggedIn: false,
        reason: 'no_token'
      }
    }
  } catch (error) {
    console.error('❌ 检查认证状态失败:', error)
    return {
      isLoggedIn: false,
      reason: 'error',
      error: error.message
    }
  }
}

/**
 * 清除认证信息
 */
function clearAuthData() {
  console.log('🧹 清除认证信息...')
  
  try {
    wx.removeStorageSync('access_token')
    wx.removeStorageSync('refresh_token')
    wx.removeStorageSync('user_info')
    wx.removeStorageSync('token_expires')
    
    console.log('✅ 认证信息已清除')
    
    wx.showToast({
      title: '认证信息已清除',
      icon: 'success',
      duration: 2000
    })
  } catch (error) {
    console.error('❌ 清除认证信息失败:', error)
    
    wx.showToast({
      title: '清除失败',
      icon: 'error',
      duration: 2000
    })
  }
}

/**
 * 跳转到登录页
 */
function goToLogin() {
  console.log('🔗 跳转到登录页...')
  
  wx.navigateTo({
    url: '/pages/login/login',
    success: () => {
      console.log('✅ 跳转到登录页成功')
    },
    fail: (error) => {
      console.error('❌ 跳转到登录页失败:', error)
      
      wx.showToast({
        title: '跳转失败',
        icon: 'error',
        duration: 2000
      })
    }
  })
}

/**
 * 测试API调用
 */
function testAPICall() {
  console.log('🧪 测试API调用...')
  
  const authStatus = checkAuthStatus()
  
  if (!authStatus.isLoggedIn) {
    console.log('❌ 未登录，无法测试API调用')
    wx.showToast({
      title: '请先登录',
      icon: 'none',
      duration: 2000
    })
    return
  }
  
  // 测试一个简单的API调用
  const contentAPI = require('../apis/content.js')
  
  contentAPI.getSlideCountLimits()
    .then(response => {
      console.log('✅ API调用成功:', response)
      wx.showToast({
        title: 'API调用成功',
        icon: 'success',
        duration: 2000
      })
    })
    .catch(error => {
      console.error('❌ API调用失败:', error)
      wx.showToast({
        title: 'API调用失败',
        icon: 'error',
        duration: 2000
      })
    })
}

/**
 * 显示认证状态对话框
 */
function showAuthStatusDialog() {
  const authStatus = checkAuthStatus()
  
  let message = ''
  let buttons = []
  
  if (authStatus.isLoggedIn) {
    message = '当前已登录，token有效'
    buttons = [
      { text: '测试API', handler: testAPICall },
      { text: '清除认证', handler: clearAuthData },
      { text: '取消' }
    ]
  } else {
    message = `未登录状态\n原因: ${authStatus.reason || '未知'}`
    buttons = [
      { text: '去登录', handler: goToLogin },
      { text: '清除认证', handler: clearAuthData },
      { text: '取消' }
    ]
  }
  
  wx.showModal({
    title: '认证状态',
    content: message,
    showCancel: true,
    cancelText: '取消',
    confirmText: buttons[0].text,
    success: (res) => {
      if (res.confirm) {
        buttons[0].handler()
      }
    }
  })
}

/**
 * 自动修复认证问题
 */
function autoFixAuth() {
  console.log('🔧 自动修复认证问题...')
  
  const authStatus = checkAuthStatus()
  
  if (authStatus.isLoggedIn) {
    console.log('✅ 认证状态正常，无需修复')
    wx.showToast({
      title: '认证状态正常',
      icon: 'success',
      duration: 2000
    })
    return
  }
  
  // 清除可能损坏的认证信息
  clearAuthData()
  
  // 延迟跳转到登录页
  setTimeout(() => {
    goToLogin()
  }, 1000)
}

module.exports = {
  checkAuthStatus,
  clearAuthData,
  goToLogin,
  testAPICall,
  showAuthStatusDialog,
  autoFixAuth
} 