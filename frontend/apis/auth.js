const { request } = require('../utils/request.js')
const { getBaseURLSync } = require('../utils/config.js')

const API_BASE = getBaseURLSync()

const authAPI = {
  /**
   * 微信登录
   * @param {string} code 微信授权码
   * @param {object} userInfo 用户信息（可选）
   */
  async wechatLogin(code, userInfo = {}) {
    try {
      const response = await request({
        url: `${API_BASE}/auth/wechat`,
        method: 'POST',
        data: {
          code: code,
          encrypted_data: userInfo.encryptedData || '',
          iv: userInfo.iv || ''
        }
      });

      // 验证登录结果
      if (response.data && response.data.user) {
        console.log('🎉 微信登录成功，用户信息:', {
          id: response.data.user.id,
          openid: response.data.user.openid,
          nickname: response.data.user.nickname,
          isNewUser: response.data.is_new_user
        });
        
        // 检查是否为测试用户
        if (response.data.user.openid.includes('mock_openid')) {
          console.warn('⚠️ 当前使用测试环境，用户数据可能不是真实的');
          console.warn('🔍 检测到Mock用户，OpenID:', response.data.user.openid);
        } else {
          console.log('✅ 真实微信用户登录，OpenID:', response.data.user.openid);
        }

        // 新用户提示
        if (response.data.is_new_user) {
          console.log('🆕 检测到新用户，获得初始积分:', response.data.user.credits);
        }
      }

      return response;
    } catch (error) {
      console.error('❌ 微信登录失败:', error);
      throw error;
    }
  },

  /**
   * 验证Token
   * @param {string} token JWT token
   */
  validateToken(token) {
    return request({
      url: `${API_BASE}/auth/validate`,
      method: 'GET',
      data: { token }
    })
  },

  /**
   * 获取用户资料
   */
  getProfile() {
    return request({
      url: `${API_BASE}/auth/profile`,
      method: 'GET',
      needAuth: true
    })
  },

  /**
   * 更新用户资料
   * @param {object} profile 用户资料
   */
  updateProfile(profile) {
    return request({
      url: `${API_BASE}/auth/profile`,
      method: 'PUT',
      data: profile,
      needAuth: true
    })
  },

  /**
   * 获取用户积分
   */
  getCredits() {
    return request({
      url: `${API_BASE}/auth/credits`,
      method: 'GET',
      needAuth: true
    })
  },

  /**
   * 消费积分
   * @param {number} amount 积分数量
   */
  consumeCredits(amount) {
    return request({
      url: `${API_BASE}/auth/credits/consume`,
      method: 'POST',
      data: { amount },
      needAuth: true
    })
  },

  /**
   * 获取VIP状态
   */
  getVIPStatus() {
    return request({
      url: `${API_BASE}/auth/vip`,
      method: 'GET',
      needAuth: true
    })
  },

  /**
   * 刷新Token
   * @param {string} refreshToken 刷新token
   */
  refreshToken(refreshToken) {
    return request({
      url: `${API_BASE}/auth/refresh`,
      method: 'POST',
      data: { refresh_token: refreshToken }
    })
  },

  /**
   * 用户登出
   */
  logout() {
    return request({
      url: `${API_BASE}/auth/logout`,
      method: 'POST',
      needAuth: true
    })
  }
}

module.exports = authAPI 