// pages/profile/profile.js
const userAPI = require('../../apis/user.js')
const authAPI = require('../../apis/auth.js')
const auth = require('../../utils/auth.js')

Page({
  data: {
    userInfo: {},
    userStats: {},
    isLoading: false,
    loadingText: '',
    showCreditsModal: false,
    showEditModal: false,
    editForm: {
      nickname: '',
      phone: '',
      email: ''
    }
  },

  onLoad() {
    console.log('个人中心页面加载成功')
    
    // 检查登录状态
    if (!auth.isLoggedIn()) {
      auth.requireLogin()
      return
    }

    this.loadUserData()
  },

  onShow() {
    console.log('个人中心页面显示')
    
    // 每次显示时检查登录状态
    if (!auth.isLoggedIn()) {
      auth.requireLogin()
      return
    }
    
    // 刷新用户数据
    this.loadUserData()
  },

  /**
   * 下拉刷新
   */
  onPullDownRefresh() {
    this.loadUserData()
    setTimeout(() => {
      wx.stopPullDownRefresh()
    }, 1000)
  },

  /**
   * 加载用户数据
   */
  async loadUserData() {
    this.setData({ 
      isLoading: true,
      loadingText: '加载用户信息...'
    })

    try {
      // 并行加载用户信息和统计数据
      const [userInfo, userStats] = await Promise.all([
        userAPI.getProfile(),
        userAPI.getStats()
      ])

      // 处理响应数据结构
      const userData = userInfo.data || userInfo
      const statsData = userStats.data || userStats

      // 添加默认头像
      if (!userData.avatar_url) {
        userData.avatar_url = '/images/default-avatar.svg'
      }

      // 确保学习时长数据存在
      if (!statsData.total_study_time) {
        statsData.total_study_time = userData.total_study_time || 0
      }

      this.setData({
        userInfo: userData,
        userStats: statsData
      })
      
      // 同步更新本地存储
      wx.setStorageSync('userInfo', userData)
      
    } catch (error) {
      console.error('加载用户数据失败:', error)
      wx.showToast({
        title: '加载失败: ' + error.message,
        icon: 'none',
        duration: 2000
      })
    } finally {
      this.setData({ isLoading: false })
    }
  },

  /**
   * 获取VIP等级名称
   */
  getVipLevelName(vipLevel) {
    switch (vipLevel) {
      case 1:
        return '月度会员'
      case 2:
        return '年度会员'
      default:
        return '普通用户'
    }
  },

  /**
   * 格式化日期
   */
  formatDate(dateString) {
    if (!dateString) return ''
    const date = new Date(dateString)
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
  },

  /**
   * 格式化学习时长
   */
  formatStudyTime(seconds) {
    if (!seconds || seconds === 0) return '0分钟'
    
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    
    if (hours > 0) {
      return `${hours}小时${minutes}分钟`
    } else {
      return `${minutes}分钟`
    }
  },

  /**
   * 修改头像
   */
  changeAvatar() {
    wx.chooseImage({
      count: 1,
      sizeType: ['compressed'],
      sourceType: ['album', 'camera'],
      success: (res) => {
        const tempFilePath = res.tempFilePaths[0]
        console.log('选择头像:', tempFilePath)
        
        // TODO: 实现头像上传
        wx.showToast({
          title: '头像上传功能开发中...',
          icon: 'none',
          duration: 2000
        })
      }
    })
  },

  /**
   * 编辑资料
   */
  editProfile() {
    const { userInfo } = this.data
    this.setData({
      showEditModal: true,
      editForm: {
        nickname: userInfo.nickname || '',
        phone: userInfo.phone || '',
        email: userInfo.email || ''
      }
    })
  },

  /**
   * 保存资料
   */
  async saveProfile() {
    const { editForm } = this.data
    
    // 简单验证
    if (!editForm.nickname.trim()) {
      wx.showToast({
        title: '请输入昵称',
        icon: 'none',
        duration: 2000
      })
      return
    }

    this.setData({ 
      isLoading: true,
      loadingText: '保存中...'
    })

    try {
      // 构建更新数据
      const updateData = {}
      if (editForm.nickname !== this.data.userInfo.nickname) {
        updateData.nickname = editForm.nickname
      }
      if (editForm.phone !== this.data.userInfo.phone) {
        updateData.phone = editForm.phone
      }
      if (editForm.email !== this.data.userInfo.email) {
        updateData.email = editForm.email
      }
      
      // 如果有更新的数据，调用API
      if (Object.keys(updateData).length > 0) {
        const updatedUserInfo = await userAPI.updateProfile(updateData)
        this.setData({
          userInfo: updatedUserInfo,
          showEditModal: false
        })
        
        // 同步更新本地存储
        wx.setStorageSync('userInfo', updatedUserInfo)
        
        wx.showToast({
          title: '保存成功',
          icon: 'success',
          duration: 2000
        })
      } else {
        this.setData({ showEditModal: false })
      }

    } catch (error) {
      console.error('保存资料失败:', error)
      wx.showToast({
        title: '保存失败: ' + error.message,
        icon: 'none',
        duration: 2000
      })
    } finally {
      this.setData({ isLoading: false })
    }
  },

  /**
   * 表单输入处理
   */
  onNicknameInput(e) {
    this.setData({
      'editForm.nickname': e.detail.value
    })
  },

  onPhoneInput(e) {
    this.setData({
      'editForm.phone': e.detail.value
    })
  },

  onEmailInput(e) {
    this.setData({
      'editForm.email': e.detail.value
    })
  },

  /**
   * 显示积分说明
   */
  showCreditsInfo() {
    this.setData({ showCreditsModal: true })
  },

  /**
   * 隐藏积分弹窗
   */
  hideCreditsModal() {
    this.setData({ showCreditsModal: false })
  },

  /**
   * 隐藏编辑弹窗
   */
  hideEditModal() {
    this.setData({ showEditModal: false })
  },

  /**
   * 阻止弹窗关闭
   */
  preventClose() {
    // 空函数，阻止事件冒泡
  },

  /**
   * 查看积分历史
   */
  async viewCreditsHistory() {
    try {
      const usage = await userAPI.getUsage({
        type: 'course_generation',
        page: 1,
        limit: 10
      })
      
      wx.showModal({
        title: '积分使用记录',
        content: `最近使用记录:\n${usage.records.map(record => 
          `${record.resource_title} - ${record.consumed_credits}积分`
        ).join('\n') || '暂无记录'}`,
        showCancel: false
      })
    } catch (error) {
      console.error('获取积分历史失败:', error)
      wx.showToast({
        title: '获取失败: ' + error.message,
        icon: 'none'
      })
    }
  },

  /**
   * 购买积分
   */
  buyCredits() {
    this.setData({ showCreditsModal: false })
    wx.showToast({
      title: '购买积分功能尚未开放',
      icon: 'none',
      duration: 2000
    })
  },

  /**
   * 导航功能
   */
  goToMyCourses() {
    wx.navigateTo({
      url: '/pages/course/list/list'
    })
  },

  goToLearningHistory() {
    wx.navigateTo({
      url: '/pages/profile/learning/learning'
    })
  },

  goToFavorites() {
    wx.showToast({
      title: '我的收藏功能尚未开放',
      icon: 'none',
      duration: 2000
    })
  },

  goToVIP() {
    wx.showToast({
      title: '会员中心功能尚未开放',
      icon: 'none',
      duration: 2000
    })
  },

  goToSettings() {
    wx.showToast({
      title: '设置功能开发中...',
      icon: 'none',
      duration: 2000
    })
  },

  goToHelp() {
    wx.showModal({
      title: '帮助中心',
      content: '如果您在使用过程中遇到问题，可以：\n\n1. 查看常见问题\n2. 联系客服\n3. 提交意见反馈\n\n我们会及时为您解答。',
      showCancel: false
    })
  },

  aboutApp() {
    wx.showModal({
      title: 'AI课堂',
      content: 'AI课堂 v1.0.0\n\n一款基于AI技术的智能学习平台，帮助用户快速生成学习课件，提升学习效率。\n\n技术支持：AI课堂团队',
      showCancel: false,
      confirmText: '确定'
    })
  },

  /**
   * 退出登录
   */
  logout() {
    wx.showModal({
      title: '确认退出',
      content: '确定要退出登录吗？',
      success: (res) => {
        if (res.confirm) {
          this.performLogout()
        }
      }
    })
  },

  /**
   * 执行退出登录
   */
  async performLogout() {
    this.setData({ 
      isLoading: true,
      loadingText: '退出中...'
    })

    try {
      // 调用后端退出登录API
      await authAPI.logout()
    } catch (error) {
      console.error('退出登录API调用失败:', error)
      // 即使API调用失败，也继续清除本地数据
    }

    try {
      // 清除本地存储
      auth.clearAuth()

      wx.showToast({
        title: '已退出登录',
        icon: 'success',
        duration: 1500
      })

      // 跳转到登录页
      setTimeout(() => {
        wx.redirectTo({
          url: '/pages/login/login'
        })
      }, 1500)

    } catch (error) {
      console.error('清除本地数据失败:', error)
      wx.redirectTo({
        url: '/pages/login/login'
      })
    } finally {
      this.setData({ isLoading: false })
    }
  },

  /**
   * 分享功能
   */
  onShareAppMessage() {
    return {
      title: `${this.data.userInfo.nickname || '我'}正在使用AI课堂学习`,
      path: '/pages/index/index'
    }
  }
})