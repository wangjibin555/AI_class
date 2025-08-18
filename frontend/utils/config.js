/**
 * 配置工具函数
 */

// 开发环境配置
const DEV_CONFIG = {
  baseURL: 'http://wangjibin-sc.wepie.com:8088/api/v1',  // HTTP服务使用8088端口
  wsURL: 'ws://wangjibin-sc.wepie.com:8088/ws',           // WebSocket使用8088端口
  fileURL: 'http://wangjibin-sc.wepie.com:8088'           // 文件服务使用8088端口
}

// 生产环境配置
const PROD_CONFIG = {
  baseURL: 'https://wangjibin-sc.wepie.com:9000/api/v1',
  wsURL: 'wss://wangjibin-sc.wepie.com:9000/ws',
  fileURL: 'https://wangjibin-sc.wepie.com:9000'
}

// 动态配置缓存
let dynamicConfig = null
let configPromise = null

/**
 * 检测当前运行环境
 */
function getEnvironment() {
  try {
    // 获取小程序账号信息
    const accountInfo = wx.getAccountInfoSync()
    
    // 检查是否在开发者工具中
    const systemInfo = wx.getDeviceInfo()
    if (systemInfo.platform === 'devtools') {
      return 'development'
    }
    
    // 根据小程序版本类型判断环境
    if (accountInfo.miniProgram.envVersion === 'develop') {
      return 'development'
    } else if (accountInfo.miniProgram.envVersion === 'trial') {
      return 'staging'
    } else if (accountInfo.miniProgram.envVersion === 'release') {
      return 'production'
    }
    
    // 默认为开发环境
    return 'development'
  } catch (error) {
    console.warn('环境检测失败，默认为开发环境:', error)
    return 'development'
  }
}

/**
 * 获取当前环境的默认配置
 */
function getDefaultConfig() {
  const env = getEnvironment()
  console.log('当前检测到的环境:', env)
  
  if (env === 'production' || env === 'staging') {
    return PROD_CONFIG
  }
  return DEV_CONFIG
}

/**
 * 从服务器获取配置
 */
function fetchServerConfig() {
  if (configPromise) {
    return configPromise
  }

  const defaultConfig = getDefaultConfig()
  
  configPromise = new Promise((resolve, reject) => {
    const configUrl = defaultConfig.baseURL.replace('/api/v1', '') + '/api/v1/config/client'
    console.log('正在从以下地址获取配置:', configUrl)
    
    wx.request({
      url: configUrl,
      method: 'GET',
      timeout: 10000,
      success: (res) => {
        console.log('服务器配置获取成功:', res.data)
        if (res.statusCode === 200 && res.data.success) {
          dynamicConfig = res.data.data.server
          console.log('使用服务器动态配置:', dynamicConfig)
          resolve(dynamicConfig)
        } else {
          console.warn('服务器配置获取失败，使用默认配置:', res.data)
          resolve(defaultConfig)
        }
      },
      fail: (err) => {
        console.error('服务器配置获取失败，使用默认配置:', err)
        console.log('使用的默认配置:', defaultConfig)
        resolve(defaultConfig)
      }
    })
  })

  return configPromise
}

/**
 * 获取基础URL
 */
function getBaseURL() {
  return new Promise((resolve) => {
    if (dynamicConfig) {
      resolve(dynamicConfig.base_url)
      return
    }

    fetchServerConfig().then((config) => {
      resolve(config.base_url || getDefaultConfig().baseURL)
    })
  })
}

/**
 * 获取WebSocket URL
 */
function getWSURL() {
  return new Promise((resolve) => {
    if (dynamicConfig) {
      resolve(dynamicConfig.ws_url)
      return
    }

    fetchServerConfig().then((config) => {
      resolve(config.ws_url || getDefaultConfig().wsURL)
    })
  })
}

/**
 * 获取文件服务URL
 */
function getFileURL() {
  return new Promise((resolve) => {
    if (dynamicConfig) {
      resolve(dynamicConfig.file_url)
      return
    }

    fetchServerConfig().then((config) => {
      resolve(config.file_url || getDefaultConfig().fileURL)
    })
  })
}

/**
 * 同步方式获取基础URL（用于需要立即获取的场景）
 */
function getBaseURLSync() {
  if (dynamicConfig) {
    return dynamicConfig.base_url
  }
  return getDefaultConfig().baseURL
}

/**
 * 同步方式获取WebSocket URL
 */
function getWSURLSync() {
  if (dynamicConfig) {
    return dynamicConfig.ws_url
  }
  return getDefaultConfig().wsURL
}

/**
 * 同步方式获取文件服务URL
 */
function getFileURLSync() {
  if (dynamicConfig) {
    return dynamicConfig.file_url
  }
  return getDefaultConfig().fileURL
}

/**
 * 获取当前环境信息
 */
function getEnvInfo() {
  return {
    environment: getEnvironment(),
    defaultConfig: getDefaultConfig(),
    dynamicConfig: dynamicConfig
  }
}

module.exports = {
  getBaseURL,
  getWSURL,
  getFileURL,
  getBaseURLSync,
  getWSURLSync,
  getFileURLSync,
  fetchServerConfig,
  getEnvironment,
  getEnvInfo
}


