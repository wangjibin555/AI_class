const { request } = require('../utils/request.js')
const { getBaseURLSync } = require('../utils/config.js')

const API_BASE = getBaseURLSync() + '/ai-assistant'

/**
 * AI助手API
 */
const aiAssistantAPI = {
  /**
   * 发送消息给AI助手
   * @param {String} message 用户消息
   * @param {String} context 可选的上下文信息
   */
  chat(message, context = '') {
    return request({
      url: `${API_BASE}/chat`,
      method: 'POST',
      data: {
        message,
        context
      },
      needAuth: true,
      timeout: 120000 // 2分钟超时，因为AI响应可能需要更长时间
    })
  },

  /**
   * 获取对话历史
   * @param {Number} page 页码
   * @param {Number} limit 每页数量
   */
  getHistory(page = 1, limit = 50) {
    return request({
      url: `${API_BASE}/history`,
      method: 'GET',
      data: {
        page,
        limit
      },
      needAuth: true
    })
  },

  /**
   * 清空对话历史
   */
  clearHistory() {
    return request({
      url: `${API_BASE}/history`,
      method: 'DELETE',
      needAuth: true
    })
  }
}

module.exports = aiAssistantAPI 