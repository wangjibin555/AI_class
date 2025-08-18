const { request } = require('../utils/request.js')
const { getBaseURLSync } = require('../utils/config.js')

const API_BASE = getBaseURLSync()

const userAPI = {
  /**
   * 获取用户资料
   */
  getProfile() {
    return request({
      url: `${API_BASE}/users/profile`,
      method: 'GET',
      needAuth: true
    })
  },

  /**
   * 更新用户资料
   * @param {Object} profile 用户资料
   */
  updateProfile(profile) {
    return request({
      url: `${API_BASE}/users/profile`,
      method: 'PUT',
      data: profile,
      needAuth: true
    })
  },

  /**
   * 获取用户统计信息
   */
  getStats() {
    return request({
      url: `${API_BASE}/users/stats`,
      method: 'GET',
      needAuth: true
    })
  },

  /**
   * 获取用户使用记录
   * @param {Object} params 查询参数
   */
  getUsage(params = {}) {
    const queryString = new URLSearchParams(params).toString()
    return request({
      url: `${API_BASE}/users/usage${queryString ? '?' + queryString : ''}`,
      method: 'GET',
      needAuth: true
    })
  },

  /**
   * 消费积分
   * @param {Number} amount 积分数量
   */
  consumeCredits(amount) {
    return request({
      url: `${API_BASE}/users/credits/consume`,
      method: 'POST',
      data: { amount },
      needAuth: true
    })
  },

  /**
   * 增加积分
   * @param {Number} amount 积分数量
   */
  addCredits(amount) {
    return request({
      url: `${API_BASE}/users/credits/add`,
      method: 'POST',
      data: { amount },
      needAuth: true
    })
  },

  /**
   * 获取指定用户信息
   * @param {Number} userId 用户ID
   */
  getUserInfo(userId) {
    return request({
      url: `${API_BASE}/users/${userId}`,
      method: 'GET',
      needAuth: true
    })
  },

  /**
   * 更新用户VIP状态
   * @param {Number} userId 用户ID
   * @param {Object} vipData VIP数据
   */
  updateUserVIP(userId, vipData) {
    return request({
      url: `${API_BASE}/users/${userId}/vip`,
      method: 'PUT',
      data: vipData,
      needAuth: true
    })
  }
}

module.exports = userAPI