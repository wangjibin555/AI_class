const aiAssistantAPI = require('../../apis/ai-assistant.js')

Page({
  data: {
    messages: [],
    inputMessage: '',
    isLoading: false,
    isEmpty: true,
    scrollTop: 0,
    autoScroll: true
  },

  onLoad() {
    this.loadHistory()
  },

  onShow() {
    // 页面显示时滚动到底部
    this.scrollToBottom()
  },

  /**
   * 加载对话历史
   */
  async loadHistory() {
    try {
      const response = await aiAssistantAPI.getHistory(1, 50)
      const history = response.history || []
      
      // 转换历史记录格式
      const messages = []
      history.reverse().forEach(record => {
        // 添加用户消息
        messages.push({
          id: `user_${record.id}`,
          type: 'user',
          content: record.message,
          timestamp: record.created_at
        })
        
        // 添加AI回复
        messages.push({
          id: `ai_${record.id}`,
          type: 'ai',
          content: record.response,
          timestamp: record.created_at,
          hasCode: this.detectCodeBlock(record.response),
          codeContent: this.extractCodeContent(record.response),
          codeLanguage: this.detectCodeLanguage(record.response)
        })
      })
      
      this.setData({
        messages,
        isEmpty: messages.length === 0
      })
      
      // 滚动到底部
      this.scrollToBottom()
      
    } catch (error) {
      console.error('加载对话历史失败:', error)
      wx.showToast({
        title: '加载历史失败',
        icon: 'none'
      })
    }
  },

  /**
   * 发送消息
   */
  async sendMessage() {
    const message = this.data.inputMessage.trim()
    if (!message) {
      wx.showToast({
        title: '请输入消息',
        icon: 'none'
      })
      return
    }

    // 添加用户消息到列表
    const userMessage = {
      id: `user_${Date.now()}`,
      type: 'user',
      content: message,
      timestamp: new Date().toISOString()
    }

    this.setData({
      messages: [...this.data.messages, userMessage],
      inputMessage: '',
      isLoading: true,
      isEmpty: false
    })

    // 滚动到底部
    this.scrollToBottom()

    try {
      // 发送消息给AI助手
      const response = await aiAssistantAPI.chat(message)
      const aiResponse = response.response

      // 添加AI回复到列表
      const aiMessage = {
        id: `ai_${Date.now()}`,
        type: 'ai',
        content: aiResponse,
        timestamp: new Date().toISOString(),
        hasCode: this.detectCodeBlock(aiResponse),
        codeContent: this.extractCodeContent(aiResponse),
        codeLanguage: this.detectCodeLanguage(aiResponse)
      }

      this.setData({
        messages: [...this.data.messages, aiMessage],
        isLoading: false
      })

      // 滚动到底部
      this.scrollToBottom()

    } catch (error) {
      console.error('发送消息失败:', error)
      this.setData({
        isLoading: false
      })
      
      wx.showToast({
        title: '发送失败: ' + error.message,
        icon: 'none'
      })
    }
  },

  /**
   * 输入框内容变化
   */
  onInputChange(e) {
    this.setData({
      inputMessage: e.detail.value
    })
  },

  /**
   * 清除对话历史
   */
  async clearHistory() {
    wx.showModal({
      title: '确认清除',
      content: '确定要清除所有对话历史吗？',
      success: async (res) => {
        if (res.confirm) {
          try {
            await aiAssistantAPI.clearHistory()
            this.setData({
              messages: [],
              isEmpty: true
            })
            
            wx.showToast({
              title: '历史已清除',
              icon: 'success'
            })
          } catch (error) {
            console.error('清除历史失败:', error)
            wx.showToast({
              title: '清除失败',
              icon: 'none'
            })
          }
        }
      }
    })
  },

  /**
   * 滚动到底部
   */
  scrollToBottom() {
    // 延迟执行，确保DOM更新完成
    setTimeout(() => {
      this.setData({
        scrollTop: 99999,
        autoScroll: true
      })
    }, 100)
  },

  /**
   * 复制消息内容
   */
  copyMessage(e) {
    const content = e.currentTarget.dataset.content
    wx.setClipboardData({
      data: content,
      success: () => {
        wx.showToast({
          title: '已复制',
          icon: 'success'
        })
      }
    })
  },

  /**
   * 返回上一页
   */
  goBack() {
    wx.navigateBack()
  },

  /**
   * 显示菜单
   */
  showMenu() {
    wx.showActionSheet({
      itemList: ['清除历史', '设置', '帮助'],
      success: (res) => {
        if (res.tapIndex === 0) {
          this.clearHistory()
        } else if (res.tapIndex === 1) {
          // 设置功能
          wx.showToast({
            title: '设置功能开发中',
            icon: 'none'
          })
        } else if (res.tapIndex === 2) {
          // 帮助功能
          wx.showToast({
            title: '帮助功能开发中',
            icon: 'none'
          })
        }
      }
    })
  },

  /**
   * 新对话
   */
  newChat() {
    wx.showModal({
      title: '新对话',
      content: '确定要开始新的对话吗？当前对话将被清除。',
      success: (res) => {
        if (res.confirm) {
          this.clearHistory()
        }
      }
    })
  },

  /**
   * 复制代码
   */
  copyCode(e) {
    const content = e.currentTarget.dataset.content
    wx.setClipboardData({
      data: content,
      success: () => {
        wx.showToast({
          title: '代码已复制',
          icon: 'success'
        })
      }
    })
  },



  /**
   * 点赞消息
   */
  likeMessage(e) {
    const index = e.currentTarget.dataset.index
    wx.showToast({
      title: '感谢您的反馈',
      icon: 'success'
    })
  },

  /**
   * 点踩消息
   */
  dislikeMessage(e) {
    const index = e.currentTarget.dataset.index
    wx.showToast({
      title: '感谢您的反馈',
      icon: 'success'
    })
  },



  /**
   * 检测代码块
   */
  detectCodeBlock(content) {
    return /```[\s\S]*?```/.test(content)
  },

  /**
   * 提取代码内容
   */
  extractCodeContent(content) {
    const match = content.match(/```(?:(\w+)\n)?([\s\S]*?)```/)
    return match ? match[2].trim() : ''
  },

  /**
   * 检测代码语言
   */
  detectCodeLanguage(content) {
    const match = content.match(/```(\w+)/)
    return match ? match[1] : 'code'
  },

  /**
   * 分享功能
   */
  onShareAppMessage() {
    return {
      title: 'AI课堂智能助手',
      path: '/pages/ai-assistant/ai-assistant',
      imageUrl: '/images/ai-assistant-share.png'
    }
  }
}) 