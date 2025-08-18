const placeholder = require('../../../images/placeholder.png')

Component({
  properties: {
    show: {
      type: Boolean,
      value: true
    },
    title: {
      type: String,
      value: '请先登录'
    },
    desc: {
      type: String,
      value: '登录后即可使用完整功能'
    },
    btnText: {
      type: String,
      value: '立即登录'
    },
    iconUrl: {
      type: String,
      value: placeholder // TODO: 替换为登录提示图标
    }
  },

  data: {
    
  },

  methods: {
    /**
     * 跳转到登录页
     */
    goToLogin() {
      // 获取当前页面路径作为登录后的重定向地址
      const pages = getCurrentPages()
      const currentPage = pages[pages.length - 1]
      const currentRoute = currentPage.route
      const currentOptions = currentPage.options
      
      // 构建重定向URL
      let redirectUrl = `/${currentRoute}`
      const query = Object.keys(currentOptions).map(key => 
        `${key}=${encodeURIComponent(currentOptions[key])}`
      ).join('&')
      
      if (query) {
        redirectUrl += `?${query}`
      }

      // 跳转到登录页，携带重定向参数
      wx.navigateTo({
        url: `/pages/login/login?redirect=${encodeURIComponent(redirectUrl)}`,
        fail: () => {
          // 如果导航失败，使用重定向
          wx.redirectTo({
            url: `/pages/login/login?redirect=${encodeURIComponent(redirectUrl)}`
          })
        }
      })

      // 触发父组件事件
      this.triggerEvent('login')
    }
  }
}) 