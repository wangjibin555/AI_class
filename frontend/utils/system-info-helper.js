/**
 * 系统信息获取工具
 * 使用新的API替代废弃的wx.getSystemInfoSync
 */

/**
 * 获取完整的系统信息
 * @returns {Object} 系统信息对象
 */
function getSystemInfo() {
  try {
    // 使用新的API组合获取系统信息
    const windowInfo = wx.getWindowInfo()
    const deviceInfo = wx.getDeviceInfo()
    const appBaseInfo = wx.getAppBaseInfo()
    
    // 组合成类似旧API的格式
    const systemInfo = {
      ...windowInfo,
      ...deviceInfo,
      ...appBaseInfo,
      // 添加一些兼容性字段
      screenWidth: windowInfo.screenWidth,
      screenHeight: windowInfo.screenHeight,
      windowWidth: windowInfo.windowWidth,
      windowHeight: windowInfo.windowHeight,
      pixelRatio: windowInfo.pixelRatio,
      platform: deviceInfo.platform,
      system: deviceInfo.system,
      version: appBaseInfo.version,
      SDKVersion: appBaseInfo.SDKVersion
    }
    
    console.log('✅ 成功获取系统信息 (新API):', systemInfo)
    return systemInfo
  } catch (error) {
    console.warn('❌ 新API获取系统信息失败:', error)
    
    // 提供默认值
    const defaultSystemInfo = {
      platform: 'unknown',
      system: 'unknown',
      version: '1.0.0',
      SDKVersion: '3.0.0',
      pixelRatio: 2,
      windowWidth: 375,
      windowHeight: 667,
      screenWidth: 375,
      screenHeight: 667
    }
    
    console.log('🔄 使用默认系统信息:', defaultSystemInfo)
    return defaultSystemInfo
  }
}

/**
 * 检查是否在开发者工具中运行
 * @returns {Boolean} 是否在开发者工具中
 */
function isDevTools() {
  try {
    const deviceInfo = wx.getDeviceInfo()
    return deviceInfo.platform === 'devtools'
  } catch (error) {
    console.warn('检查开发者工具状态失败:', error)
    return false
  }
}

/**
 * 获取设备平台
 * @returns {String} 平台名称 ('ios', 'android', 'devtools', 'unknown')
 */
function getPlatform() {
  try {
    const deviceInfo = wx.getDeviceInfo()
    return deviceInfo.platform || 'unknown'
  } catch (error) {
    console.warn('获取平台信息失败:', error)
    return 'unknown'
  }
}

/**
 * 获取窗口尺寸信息
 * @returns {Object} 窗口尺寸对象
 */
function getWindowSize() {
  try {
    const windowInfo = wx.getWindowInfo()
    return {
      windowWidth: windowInfo.windowWidth,
      windowHeight: windowInfo.windowHeight,
      screenWidth: windowInfo.screenWidth,
      screenHeight: windowInfo.screenHeight,
      pixelRatio: windowInfo.pixelRatio
    }
  } catch (error) {
    console.warn('获取窗口尺寸失败:', error)
    return {
      windowWidth: 375,
      windowHeight: 667,
      screenWidth: 375,
      screenHeight: 667,
      pixelRatio: 2
    }
  }
}

/**
 * 获取应用基础信息
 * @returns {Object} 应用信息对象
 */
function getAppInfo() {
  try {
    const appBaseInfo = wx.getAppBaseInfo()
    return {
      version: appBaseInfo.version,
      SDKVersion: appBaseInfo.SDKVersion,
      language: appBaseInfo.language,
      theme: appBaseInfo.theme
    }
  } catch (error) {
    console.warn('获取应用信息失败:', error)
    return {
      version: '1.0.0',
      SDKVersion: '3.0.0',
      language: 'zh_CN',
      theme: 'light'
    }
  }
}

module.exports = {
  getSystemInfo,
  isDevTools,
  getPlatform,
  getWindowSize,
  getAppInfo
}