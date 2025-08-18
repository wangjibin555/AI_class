const { getToken, removeToken, clearAuth } = require('./auth.js')
const { getBaseURL } = require('./config.js')

/**
 * 网络请求封装
 * @param {Object} options 请求选项
 */
function request(options) {
  return new Promise((resolve, reject) => {
    const {
      url,
      method = 'GET',
      data = {},
      needAuth = false,
      header = {},
      timeout = 120000,
      isFormData = false
    } = options

    console.log('Request:', method, url, 'needAuth:', needAuth, 'isFormData:', isFormData)

    // 如果需要认证，添加Authorization头
    if (needAuth) {
      const token = getToken()
      console.log('Token for request:', token ? token.substring(0, 20) + '...' : 'null')
      
      if (!token) {
        console.log('No token, redirecting to login')
        // 没有token，跳转到登录页
        wx.redirectTo({
          url: '/pages/login/login'
        })
        reject(new Error('未登录'))
        return
      }
      header['Authorization'] = `Bearer ${token}`
    }

    // 设置默认Content-Type
    if (method === 'POST' || method === 'PUT') {
      // 如果是FormData，不设置Content-Type，让微信自动处理
      if (!isFormData) {
        header['Content-Type'] = 'application/json'
      }
    }

    console.log('Request headers:', header)

    // 如果是FormData，使用uploadFile
    if (isFormData && (method === 'POST' || method === 'PUT')) {
      const { filePath, name, formData, ...otherData } = data
      
      wx.uploadFile({
        url: url,
        filePath: filePath,
        name: name || 'file',
        formData: formData || otherData,
        header: header,
        timeout: timeout,
        success: (res) => {
          console.log('Upload请求成功:', url, res.statusCode, res.data)
          
          if (res.statusCode === 200) {
            try {
              const responseData = typeof res.data === 'string' ? JSON.parse(res.data) : res.data
              const { code, data: responseBody, message } = responseData
              
              if (code === 200) {
                resolve(responseBody || responseData)
              } else {
                console.error('业务错误:', code, message)
                wx.showToast({
                  title: message || '请求失败',
                  icon: 'none',
                  duration: 2000
                })
                reject(new Error(message || '请求失败'))
              }
            } catch (parseError) {
              console.error('解析响应数据失败:', parseError)
              reject(new Error('响应数据格式错误'))
            }
          } else if (res.statusCode === 401) {
            console.error('认证失败，清除token并跳转登录')
            clearAuth()
            wx.showToast({
              title: '登录已过期，请重新登录',
              icon: 'none',
              duration: 2000
            })
            setTimeout(() => {
              wx.redirectTo({
                url: '/pages/login/login'
              })
            }, 2000)
            reject(new Error('认证失败'))
          } else {
            console.error('HTTP错误:', res.statusCode, res.data)
            wx.showToast({
              title: `请求失败(${res.statusCode})`,
              icon: 'none',
              duration: 2000
            })
            reject(new Error(`HTTP ${res.statusCode}`))
          }
        },
        fail: (err) => {
          console.error('Upload请求失败:', url, err)
          wx.showToast({
            title: '文件上传失败',
            icon: 'none',
            duration: 2000
          })
          reject(err)
        }
      })
      return
    }

    // 普通请求
    wx.request({
      url: url,
      method: method,
      data: data,
      header: header,
      timeout: timeout, // 使用传入的超时时间
      success: (res) => {
        console.log('API请求成功:', url, res.statusCode, res.data)
        
        if (res.statusCode === 200) {
          const { code, data: responseData, message } = res.data
          
          if (code === 200) {
            resolve(responseData || res.data)
          } else {
            // 业务错误
            console.error('业务错误:', code, message)
            wx.showToast({
              title: message || '请求失败',
              icon: 'none',
              duration: 2000
            })
            reject(new Error(message || '请求失败'))
          }
        } else if (res.statusCode === 401) {
          console.error('认证失败，清除token并跳转登录')
          // 认证失败，清除token并跳转登录
          clearAuth()
          wx.showToast({
            title: '登录已过期，请重新登录',
            icon: 'none',
            duration: 2000
          })
          setTimeout(() => {
            wx.redirectTo({
              url: '/pages/login/login'
            })
          }, 2000)
          reject(new Error('认证失败'))
        } else {
          // HTTP错误
          console.error('HTTP错误:', res.statusCode, res.data)
          
          // 创建包含响应数据的错误对象
          const error = new Error(`HTTP ${res.statusCode}`)
          error.statusCode = res.statusCode
          error.responseData = res.data  // 保留响应数据，包含quiz_id等信息
          
          wx.showToast({
            title: `请求失败(${res.statusCode})`,
            icon: 'none',
            duration: 2000
          })
          reject(error)
        }
      },
      fail: (err) => {
        console.error('API请求失败:', url, err)
        wx.showToast({
          title: '网络请求失败',
          icon: 'none',
          duration: 2000
        })
        reject(err)
      }
    })
  })
}

// 便捷方法
const get = (url, options = {}) => {
  if (url.startsWith('http')) {
    return request({
      url: url,
      method: 'GET',
      ...options
    })
  }
  
  // 对于相对URL，需要先获取baseURL
  return getBaseURL().then(baseURL => {
    return request({
      url: `${baseURL}${url}`,
      method: 'GET',
      ...options
    })
  })
}

const post = (url, data = {}, options = {}) => {
  if (url.startsWith('http')) {
    return request({
      url: url,
      method: 'POST',
      data,
      ...options
    })
  }
  
  // 对于相对URL，需要先获取baseURL
  return getBaseURL().then(baseURL => {
    return request({
      url: `${baseURL}${url}`,
      method: 'POST',
      data,
      ...options
    })
  })
}

const put = (url, data = {}, options = {}) => {
  if (url.startsWith('http')) {
    return request({
      url: url,
      method: 'PUT',
      data,
      ...options
    })
  }
  
  return getBaseURL().then(baseURL => {
    return request({
      url: `${baseURL}${url}`,
      method: 'PUT',
      data,
      ...options
    })
  })
}

const del = (url, options = {}) => {
  if (url.startsWith('http')) {
    return request({
      url: url,
      method: 'DELETE',
      ...options
    })
  }
  
  return getBaseURL().then(baseURL => {
    return request({
      url: `${baseURL}${url}`,
      method: 'DELETE',
      ...options
    })
  })
}

module.exports = {
  request,
  get,
  post,
  put,
  delete: del
} 