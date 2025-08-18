// 调试页面：用于查看配置信息
const { getEnvInfo, getBaseURLSync, getWSURLSync, getFileURLSync } = require('../../utils/config.js')

Page({
  data: {
    envInfo: {},
    currentConfig: {},
    logs: []
  },

  onLoad() {
    this.refreshInfo()
  },

  refreshInfo() {
    const envInfo = getEnvInfo()
    const currentConfig = {
      baseURL: getBaseURLSync(),
      wsURL: getWSURLSync(),
      fileURL: getFileURLSync()
    }

    this.setData({
      envInfo: envInfo,
      currentConfig: currentConfig,
      logs: [
        ...this.data.logs,
        `[${new Date().toLocaleTimeString()}] 配置刷新完成`
      ]
    })

    console.log('调试信息:', {
      envInfo,
      currentConfig
    })
  },

  async testConfigAPI() {
    try {
      this.addLog('开始测试配置API...')
      
      const { fetchServerConfig } = require('../../utils/config.js')
      await fetchServerConfig()
      
      this.addLog('配置API测试成功')
      this.refreshInfo()
    } catch (error) {
      this.addLog(`配置API测试失败: ${error.message}`)
    }
  },

  addLog(message) {
    this.setData({
      logs: [
        ...this.data.logs,
        `[${new Date().toLocaleTimeString()}] ${message}`
      ]
    })
  },

  clearLogs() {
    this.setData({
      logs: []
    })
  },

  copyInfo() {
    const info = JSON.stringify({
      envInfo: this.data.envInfo,
      currentConfig: this.data.currentConfig,
      logs: this.data.logs
    }, null, 2)

    wx.setClipboardData({
      data: info,
      success: () => {
        wx.showToast({
          title: '已复制到剪贴板',
          icon: 'success'
        })
      }
    })
  }
})