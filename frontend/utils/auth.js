/**
 * 认证相关工具函数
 */

/**
 * 获取访问令牌
 */
function getToken() {
  try {
    const token = wx.getStorageSync('access_token')
    console.log('getToken called, token exists:', !!token)
    return token
  } catch (e) {
    console.error('获取token失败:', e)
    return null
  }
}

/**
 * 移除认证信息
 */
function removeToken() {
  try {
    clearAuth()
  } catch (e) {
    console.error('移除token失败:', e)
  }
}

/**
 * 检查是否已登录
 */
function isLoggedIn() {
  try {
    const token = wx.getStorageSync('access_token')
    const expiresAt = wx.getStorageSync('token_expires')
    
    console.log('isLoggedIn check:', {
      hasToken: !!token,
      expiresAt: expiresAt,
      currentTime: Date.now(),
      expired: expiresAt ? Date.now() >= expiresAt : false
    })
    
    if (!token) {
      console.log('isLoggedIn: false - no token')
      return false
    }
    
    // 如果没有过期时间，假设token有效（向后兼容）
    if (!expiresAt) {
      console.log('isLoggedIn: true - no expiry time, assuming valid')
      return true
    }
    
    // 检查token是否过期，给予5分钟的缓冲时间
    const bufferTime = 5 * 60 * 1000 // 5分钟
    if (Date.now() >= (expiresAt - bufferTime)) {
      console.log('isLoggedIn: false - token expired or expiring soon')
      // Token已过期或即将过期，尝试刷新
      return false
    }
    
    console.log('isLoggedIn: true - token valid')
    return true
  } catch (e) {
    console.error('检查登录状态失败:', e)
    return false
  }
}

/**
 * 保存登录信息
 */
function saveLoginInfo(loginResponse) {
  try {
    console.log('保存登录信息:', loginResponse)
    
    if (loginResponse.token) {
      const token = loginResponse.token
      
      // 保存token
      wx.setStorageSync('access_token', token.access_token)
      if (token.refresh_token) {
        wx.setStorageSync('refresh_token', token.refresh_token)
      }
      
      // 计算过期时间，如果没有expires_in，默认1小时
      const expiresIn = token.expires_in || (60 * 60) // 默认1小时
      const expiresAt = Date.now() + (expiresIn * 1000)
      wx.setStorageSync('token_expires', expiresAt)
      
      console.log('Token保存成功，过期时间:', new Date(expiresAt))
    }
    
    if (loginResponse.user) {
      wx.setStorageSync('user_info', loginResponse.user)
    }
    
    return true
  } catch (e) {
    console.error('保存登录信息失败:', e)
    return false
  }
}

/**
 * 刷新token
 */
async function refreshToken() {
  try {
    const refreshToken = wx.getStorageSync('refresh_token')
    if (!refreshToken) {
      console.log('没有refresh_token，无法刷新')
      return false
    }
    
    console.log('尝试刷新token...')
    
    // 这里需要调用后端的refresh token API
    // 暂时返回false，表示需要重新登录
    console.log('refresh token API未实现，需要重新登录')
    return false
  } catch (e) {
    console.error('刷新token失败:', e)
    return false
  }
}

/**
 * 验证token有效性
 */
async function validateToken() {
  try {
    if (!isLoggedIn()) {
      return false
    }
    
    // 尝试调用一个简单的API来验证token
    const authAPI = require('../apis/auth.js')
    const userInfo = await authAPI.getProfile()
    
    console.log('Token验证成功:', userInfo)
    return true
  } catch (error) {
    console.error('Token验证失败:', error)
    
    // 如果是认证错误，清除token
    if (error.message && error.message.includes('认证失败')) {
      clearAuth()
    }
    
    return false
  }
}

/**
 * 获取用户信息
 */
function getUserInfo() {
  try {
    return wx.getStorageSync('user_info')
  } catch (e) {
    console.error('获取用户信息失败:', e)
    return null
  }
}

/**
 * 清除认证信息
 */
function clearAuth() {
  try {
    console.log('clearAuth called')
    wx.removeStorageSync('access_token')
    wx.removeStorageSync('refresh_token')
    wx.removeStorageSync('user_info')
    wx.removeStorageSync('token_expires')
  } catch (e) {
    console.error('清除认证信息失败:', e)
  }
}

/**
 * 要求登录（如果未登录则跳转到登录页）
 */
function requireLogin() {
  console.log('requireLogin called')
  if (!isLoggedIn()) {
    wx.showModal({
      title: '提示',
      content: '请先登录',
      showCancel: false,
      success: () => {
        wx.navigateTo({
          url: '/pages/login/login'
        })
      }
    })
    return false
  }
  return true
}

/**
 * 初始化认证状态
 */
async function initAuth() {
  try {
    console.log('初始化认证状态...')
    
    if (isLoggedIn()) {
      // 验证token有效性
      const isValid = await validateToken()
      if (!isValid) {
        console.log('Token无效，清除认证信息')
        clearAuth()
        return false
      }
      
      console.log('认证状态初始化成功')
      return true
    }
    
    console.log('用户未登录')
    return false
  } catch (e) {
    console.error('初始化认证状态失败:', e)
    return false
  }
}

module.exports = {
  getToken,
  removeToken,
  isLoggedIn,
  saveLoginInfo,
  refreshToken,
  validateToken,
  getUserInfo,
  clearAuth,
  requireLogin,
  initAuth
} 