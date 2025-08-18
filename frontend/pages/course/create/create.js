// pages/course/create/create.js
const courseAPI = require('../../../apis/course.js')
const contentAPI = require('../../../apis/content.js')
const aiContentAPI = require('../../../apis/ai-content.js') // 🆕 引入AI内容分析API
const cozeAPI = require('../../../apis/coze.js') // 🆕 引入Coze智能体API
const cozeIntegrationAPI = require('../../../apis/coze-integration.js') // 🆕 引入Coze集成API
const authAPI = require('../../../apis/auth.js')
const { getWSURLSync } = require('../../../utils/config.js') // 🔧 修复路径导入问题
const urlPlatformDetector = require('../../../utils/url-platform-detector.js') // 🆕 引入URL平台识别工具


Page({
  /**
   * 检查用户登录状态
   */
  checkAuthStatus() {
    const token = wx.getStorageSync('access_token')
    const expiresAt = wx.getStorageSync('token_expires')
    return token && (!expiresAt || Date.now() < (expiresAt - 5 * 60 * 1000))
  },

  /**
   * 清除认证信息
   */
  clearAuthData() {
    wx.removeStorageSync('access_token')
    wx.removeStorageSync('refresh_token')
    wx.removeStorageSync('user_info')
    wx.removeStorageSync('token_expires')
  },

  /**
   * 要求用户登录
   */
  requireUserLogin() {
    wx.showModal({
      title: '提示',
      content: '请先登录',
      showCancel: false,
      success: () => {
        wx.navigateTo({
          url: '/pages/login/login'
        })
      }
    })
  },

  /**
   * 页面的初始数据
   */
  data: {
    // 输入方式: url, file, text
    inputType: 'url',
    
    // 🆕 AI生成方式选择
    generationMode: 'ai_enhanced', // ai_enhanced, traditional, ai_only
    engineType: 'dashscope', // dashscope, coze (预留), hybrid (预留)
    
    // 🆕 Coze智能体相关状态
    cozeEngineStatus: 'unknown', // unknown, available, unavailable
    cozeConfig: null, // Coze引擎配置信息
    cozeTaskId: null, // 当前Coze任务ID
    cozeProgress: 0, // Coze任务进度
    availableEngines: [], // 可用的AI引擎列表
    engineLoading: false, // 引擎状态加载中
    
    // AI总结功能
    useAISummary: true,
    showSummaryOptions: false,
    summaryLevel: 'detailed',
    summaryTargetAudience: 'general',
    summaryResult: null,
    generatingSummary: false,
    
    // 表单数据
    formData: {
      title: '',
      description: '',
      category: 'general',
      tags: [],
      		voice_type: 'zhixiaobai',
      is_public: false,
      template: 'business',
      slide_count: 10,
      audience: 'general',
      difficulty: 'intermediate',
      use_ai_summary: true,
      include_notes: true,
      generate_audio: true
    },
    
    // 状态管理
    submitting: false,
    generating: false,
    generationCompleted: false, // 新增：标记生成是否完成
    
    // 生成进度
    generationProgress: 0,
    generationStatus: '',
    progressTimer: null, // 进度定时器
    pollTimer: null, // 轮询定时器
    
    // WebSocket连接
    wsConnection: null,
    
    // 页数限制相关
    slideCountLimits: {
      min: 3,
      max: 50,
      default: 10,
      user_max: 20
    },
    userType: 'regular', // 用户类型
    showSlideCountWarning: false, // 是否显示页数警告
    slideCountWarningText: '', // 页数警告文本
    
    // 文件处理相关
    supportedFileTypes: ['.txt', '.pdf', '.docx'], // 支持的文件类型
    fileProcessing: false, // 文件处理状态
    fileProcessResult: null, // 文件处理结果
    
    // URL输入
    urlInput: '',
    urlValidated: false,
    urlPreview: null,
    // 🆕 URL平台识别相关
    detectedPlatform: null, // 识别到的平台信息
    urlValidation: null, // URL验证结果
    showPlatformInfo: false, // 是否显示平台信息
    
    // 文件上传
    selectedFile: null,
    filePreview: null,
    fileList: [],
    uploadProgress: 0,
    uploading: false,
    fileUploaded: false,
    uploadedFileInfo: null,
    
    // 文本输入
    textInput: '',
    textPreview: null,
    
    // 分类选项
    categoryOptions: [
      { value: 'general', label: '通用' },
      { value: 'programming', label: '编程' },
      { value: 'business', label: '商业' },
      { value: 'education', label: '教育' },
      { value: 'science', label: '科学' },
      { value: 'literature', label: '文学' },
      { value: 'history', label: '历史' },
      { value: 'art', label: '艺术' }
    ],
    
    // 语音类型选项
    voiceTypeOptions: [
      		{ value: 'zhixiaobai', label: '知小白 - 智能女声' },
      { value: 'xiaoyun', label: '小云 - 温柔女声' },
      { value: 'xiaogang', label: '小刚 - 专业男声' },
      { value: 'xiaomei', label: '小美 - 清新女声' },
      { value: 'xiaofeng', label: '小峰 - 磁性男声' }
    ],
    
    // 模板选项
    templateOptions: [
      { value: 'business', label: '商务风格', desc: '专业的商务演示模板' },
      { value: 'education', label: '教育风格', desc: '清新的教育演示模板' },
      { value: 'tech', label: '科技风格', desc: '现代科技风格模板' },
      { value: 'minimal', label: '简约风格', desc: '极简风格模板' }
    ],
    
    // 幻灯片数量选项
    slideCountOptions: [
      { value: 5, label: '5页' },
      { value: 10, label: '10页' },
      { value: 15, label: '15页' },
      { value: 20, label: '20页' }
    ],
    
    // 目标受众选项
    audienceOptions: [
      { value: 'general', label: '通用' },
      { value: 'student', label: '学生' },
      { value: 'professional', label: '专业人士' },
      { value: 'beginner', label: '初学者' }
    ],
    
    // 难度等级选项
    difficultyOptions: [
      { value: 'beginner', label: '初级' },
      { value: 'intermediate', label: '中级' },
      { value: 'advanced', label: '高级' }
    ],
    
    // 总结级别选项
    summaryLevelOptions: [
      { value: 'brief', label: '简要总结', desc: '快速概括主要内容' },
      { value: 'detailed', label: '详细总结', desc: '深入分析内容结构' },
      { value: 'comprehensive', label: '全面总结', desc: '全方位深度解析' }
    ],
    
    // 总结目标受众选项
    summaryAudienceOptions: [
      { value: 'general', label: '通用', desc: '适合所有人群' },
      { value: 'student', label: '学生', desc: '注重基础概念' },
      { value: 'professional', label: '专业人士', desc: '关注深度应用' }
    ],
    
    // 界面状态
    loading: false,
    supportedPlatforms: [],
    
    // 用户信息
    userInfo: null,
    credits: 0,
    
    // Picker索引
    templateIndex: 0,
    slideCountIndex: 1,
    audienceIndex: 0,
    difficultyIndex: 1,
    categoryIndex: 0,
    voiceTypeIndex: 0,
    summaryLevelIndex: 1,
    summaryAudienceIndex: 0
  },

  /**
   * 生命周期函数--监听页面加载
   */
  onLoad(options) {
    // 检查登录状态
    this.checkLoginStatus()
    
    // 🆕 初始化AI引擎状态
    this.initializeAIEngines()
    
    // 从页面参数获取数据
    if (options.url) {
      this.setData({
        urlInput: decodeURIComponent(options.url)
      })
    }
    
    this.updatePickerIndexes()
  },

  /**
   * 检查登录状态并加载相应数据
   */
  async checkLoginStatus() {
    console.log('🔐 开始检查登录状态...')
    
    // 获取token进行详细检查
    const token = wx.getStorageSync('access_token')
    const loggedIn = this.checkAuthStatus()
    
    console.log('认证状态详情:', {
      hasToken: !!token,
      tokenPreview: token ? token.substring(0, 20) + '...' : 'null',
      isLoggedIn: loggedIn
    })
    
    if (loggedIn && token) {
      console.log('✅ 用户已登录，加载完整数据')
      try {
        // 用户已登录，加载所有数据
        await this.loadUserInfo()
        await this.loadSlideCountLimits()
        await this.loadSupportedFileTypes()
      } catch (error) {
        console.error('❌ 加载登录用户数据失败:', error)
        if (error.message && error.message.includes('认证失败')) {
          console.log('🔄 认证失败，切换到游客模式')
          // 认证失败，切换到游客模式
          await this.switchToGuestMode()
        } else {
          throw error
        }
      }
    } else {
      console.log('👤 用户未登录，切换到游客模式')
      await this.switchToGuestMode()
    }
  },

  /**
   * 切换到游客模式
   */
  async switchToGuestMode() {
    console.log('🔄 切换到游客模式...')
    
    try {
      // 只加载不需要认证的数据
      await this.loadSupportedFileTypes()
    } catch (error) {
      console.error('加载支持文件类型失败:', error)
    }
    
    // 直接设置默认的页数限制，不调用需要认证的API
    this.setDefaultLimits()
    
    // 显示登录提示
    this.showLoginPrompt()
    
    console.log('✅ 游客模式设置完成')
  },

  /**
   * 设置默认的页数限制
   */
  setDefaultLimits() {
    this.setData({
      slideCountLimits: {
        min: 3,
        max: 50,
        default: 10,
        user_max: 10 // 未登录用户限制更少
      },
      userType: 'guest'
    })
    console.log('设置默认页数限制完成')
  },

  /**
   * 显示登录提示
   */
  showLoginPrompt() {
    wx.showModal({
      title: '需要登录',
      content: '登录后可以使用更多功能，包括自定义页数限制等',
      confirmText: '去登录',
      cancelText: '继续使用',
      success: (res) => {
        if (res.confirm) {
          // 跳转到登录页
          wx.navigateTo({
            url: '/pages/login/login?redirect=' + encodeURIComponent('/pages/course/create/create')
          })
        }
      }
    })
  },

  /**
   * 生命周期函数--监听页面卸载
   */
  onUnload() {
    // 清理所有定时器和连接
    this.cleanupGeneration()
  },

  /**
   * 生命周期函数--监听页面隐藏
   */
  onHide() {
    // 页面隐藏时暂停进度动画但保留状态
    if (this.data.progressTimer) {
      clearTimeout(this.data.progressTimer)
      this.setData({ progressTimer: null })
    }
  },

  /**
   * 生命周期函数--监听页面显示
   */
  onShow() {
    // 每次显示页面时检查登录状态
    this.checkLoginStatus()
    
    // 如果正在生成且有目标进度，恢复进度动画
    if (this.data.generating && !this.data.generationCompleted) {
      this.resumeProgressAnimation()
    }
  },

  /**
   * 加载用户信息
   */
  async loadUserInfo() {
    try {
      const userInfo = await authAPI.getProfile()
      console.log('用户信息:', userInfo)
      
      // 处理响应数据结构
      const userData = userInfo.data || userInfo
      
      this.setData({
        userInfo: userData,
        credits: userData.credits || 0
      })
    } catch (error) {
      console.error('加载用户信息失败:', error)
      wx.showToast({
        title: '加载用户信息失败',
        icon: 'none'
      })
    }
  },

  /**
   * 更新Picker索引
   */
  updatePickerIndexes() {
    const { 
      formData, templateOptions, slideCountOptions, audienceOptions, 
      difficultyOptions, categoryOptions, voiceTypeOptions,
      summaryLevel, summaryTargetAudience, summaryLevelOptions, summaryAudienceOptions
    } = this.data
    
    // 更新模板索引
    const templateIndex = templateOptions.findIndex(item => item.value === formData.template)
    const slideCountIndex = slideCountOptions.findIndex(item => item.value === formData.slide_count)
    const audienceIndex = audienceOptions.findIndex(item => item.value === formData.audience)
    const difficultyIndex = difficultyOptions.findIndex(item => item.value === formData.difficulty)
    const categoryIndex = categoryOptions.findIndex(item => item.value === formData.category)
    const voiceTypeIndex = voiceTypeOptions.findIndex(item => item.value === formData.voice_type)
    const summaryLevelIndex = summaryLevelOptions.findIndex(item => item.value === summaryLevel)
    const summaryAudienceIndex = summaryAudienceOptions.findIndex(item => item.value === summaryTargetAudience)
    
    this.setData({
      templateIndex: templateIndex >= 0 ? templateIndex : 0,
      slideCountIndex: slideCountIndex >= 0 ? slideCountIndex : 1,
      audienceIndex: audienceIndex >= 0 ? audienceIndex : 0,
      difficultyIndex: difficultyIndex >= 0 ? difficultyIndex : 1,
      categoryIndex: categoryIndex >= 0 ? categoryIndex : 0,
      voiceTypeIndex: voiceTypeIndex >= 0 ? voiceTypeIndex : 0,
      summaryLevelIndex: summaryLevelIndex >= 0 ? summaryLevelIndex : 1,
      summaryAudienceIndex: summaryAudienceIndex >= 0 ? summaryAudienceIndex : 0
    })
  },

  /**
   * 加载支持的平台
   */
  async loadSupportedPlatforms() {
    // 设置默认支持的平台
    const defaultPlatforms = [
      { name: 'CSDN博客', url: 'https://blog.csdn.net', icon: '📝' },
      { name: '微信公众号', url: 'https://mp.weixin.qq.com', icon: '💬' },
      { name: '知乎', url: 'https://www.zhihu.com', icon: '🤔' },
      { name: '微博', url: 'https://weibo.com', icon: '📱' },
      { name: '哔哩哔哩', url: 'https://www.bilibili.com', icon: '📺' }
    ]
    
    this.setData({
      supportedPlatforms: defaultPlatforms
    })
    
    // 尝试从后端获取支持的平台
    try {
      const platforms = await contentAPI.getSupportedPlatforms()
      if (platforms && platforms.length > 0) {
        this.setData({
          supportedPlatforms: platforms
        })
      }
    } catch (error) {
      console.error('加载支持平台失败:', error)
      // 使用默认平台，不显示错误
    }
  },

  /**
   * 识别URL平台 - 优化版本
   */
  identifyPlatform(url) {
    if (!url) return '未知平台'
    
    // 标准化URL，移除协议前缀进行匹配
    const normalizedUrl = url.toLowerCase().replace(/^https?:\/\//, '')
    
    const platformMap = {
      // CSDN相关
      'blog.csdn.net': 'CSDN博客',
      'www.csdn.net': 'CSDN博客',
      'csdn.net': 'CSDN博客',
      
      // 哔哩哔哩相关
      'www.bilibili.com': '哔哩哔哩',
      'bilibili.com': '哔哩哔哩',
      'b23.tv': '哔哩哔哩',
      
      // 知乎相关
      'www.zhihu.com': '知乎',
      'zhihu.com': '知乎',
      'zhuanlan.zhihu.com': '知乎专栏',
      
      // 微信公众号
      'mp.weixin.qq.com': '微信公众号',
      
      // 微博相关
      'weibo.com': '微博',
      'www.weibo.com': '微博',
      'm.weibo.com': '微博',
      
      // 简书
      'www.jianshu.com': '简书',
      'jianshu.com': '简书',
      
      // 掘金
      'juejin.cn': '掘金',
      'juejin.im': '掘金',
      
      // 博客园
      'www.cnblogs.com': '博客园',
      'cnblogs.com': '博客园',
      
      // GitHub
      'github.com': 'GitHub',
      'www.github.com': 'GitHub',
      
      // 其他技术平台
      'segmentfault.com': 'SegmentFault',
      'www.segmentfault.com': 'SegmentFault',
      'oschina.net': '开源中国',
      'www.oschina.net': '开源中国'
    }
    
    // 精确匹配
    if (platformMap[normalizedUrl]) {
      return platformMap[normalizedUrl]
    }
    
    // 前缀匹配
    for (const [domain, platform] of Object.entries(platformMap)) {
      if (normalizedUrl.startsWith(domain)) {
        return platform
      }
    }
    
    return '其他网站'
  },

  /**
   * 切换输入方式
   */
  switchInputType(e) {
    const type = e.currentTarget.dataset.type
    this.setData({
      inputType: type,
      urlInput: '',
      selectedFile: null,
      textInput: '',
      urlValidated: false,
      urlPreview: null
    })
  },

  /**
   * 🆕 切换生成模式
   */
  switchGenerationMode(e) {
    const mode = e.currentTarget.dataset.mode
    this.setData({
      generationMode: mode
    })
    
    // 根据模式给出提示
    if (mode === 'ai_enhanced') {
      wx.showToast({
        title: 'AI智能模式：内容更专业，结构更清晰',
        icon: 'none',
        duration: 2000
      })
    } else if (mode === 'traditional') {
      wx.showToast({
        title: '传统模式：速度快，兼容性好',
        icon: 'none',
        duration: 2000
      })
    }
  },

  /**
   * 🆕 切换AI引擎
   */
  switchEngine(e) {
    const engine = e.currentTarget.dataset.engine
    
    // 支持DashScope和Coze引擎
    if (engine === 'dashscope') {
      this.setData({
        engineType: engine
      })
      wx.showToast({
        title: 'DashScope引擎：通用AI分析，质量稳定',
        icon: 'none',
        duration: 2000
      })
    } else if (engine === 'coze') {
      this.setData({
        engineType: engine
      })
      wx.showToast({
        title: 'Coze智能体：专业PPT生成，深度分析',
        icon: 'none',
        duration: 2000
      })
    } else {
      wx.showToast({
        title: '该引擎暂未开放，敬请期待',
        icon: 'none'
      })
    }
  },

  /**
   * URL输入变化
   */
  onUrlInput(e) {
    const urlInput = e.detail.value
    
    // 实时检测平台
    const detectedPlatform = urlPlatformDetector.detectPlatform(urlInput)
    const urlValidation = urlPlatformDetector.validateURL(urlInput)
    
    this.setData({
      urlInput: urlInput,
      urlValidated: false,
      urlPreview: null,
      detectedPlatform: detectedPlatform,
      urlValidation: urlValidation,
      showPlatformInfo: !!detectedPlatform && urlInput.trim().length > 0
    })
    
    // 如果URL格式正确且识别到平台，显示提示
    if (detectedPlatform && urlValidation.isValid && detectedPlatform.name !== '未知网站') {
      console.log('检测到平台:', detectedPlatform.name)
    }
  },

  /**
   * 验证URL
   */
  async validateUrl() {
    const { urlInput, urlValidation } = this.data
    
    if (!urlInput.trim()) {
      wx.showToast({
        title: '请输入URL',
        icon: 'none'
      })
      return
    }

    // 首先进行本地验证
    if (!urlValidation.isValid) {
      const errorMsg = urlValidation.errors.length > 0 
        ? urlValidation.errors[0] 
        : 'URL格式不正确'
      wx.showToast({
        title: errorMsg,
        icon: 'none',
        duration: 2000
      })
      return
    }

    // 显示警告信息（如果有）
    if (urlValidation.warnings.length > 0) {
      wx.showModal({
        title: '提醒',
        content: urlValidation.warnings.join('\n'),
        showCancel: true,
        confirmText: '继续验证',
        cancelText: '取消',
        success: (res) => {
          if (res.confirm) {
            this.performUrlValidation(urlValidation.normalizedURL)
          }
        }
      })
    } else {
      this.performUrlValidation(urlValidation.normalizedURL)
    }
  },

  /**
   * 执行URL验证
   */
  async performUrlValidation(normalizedURL) {
    this.setData({ loading: true })

    try {
      const result = await contentAPI.validateUrl(normalizedURL)
      console.log('URL验证结果:', result)
      
      this.setData({
        urlValidated: true,
        urlPreview: result,
        loading: false
      })
      
      // 根据检测到的平台显示不同的成功消息
      const { detectedPlatform } = this.data
      const successMsg = detectedPlatform && detectedPlatform.name !== '未知网站' 
        ? `${detectedPlatform.name} 验证成功` 
        : 'URL验证成功'
      
      wx.showToast({
        title: successMsg,
        icon: 'success'
      })
    } catch (error) {
      console.error('URL验证失败:', error)
      this.setData({
        urlValidated: false,
        urlPreview: null,
        loading: false
      })
      
      // 提供更详细的错误信息
      let errorMsg = 'URL验证失败'
      
      // 尝试从错误响应中提取具体错误信息
      if (error.data && error.data.message) {
        errorMsg = error.data.message
      } else if (error.message) {
        if (error.message.includes('网络')) {
          errorMsg = '网络连接失败，请检查网络'
        } else if (error.message.includes('404')) {
          errorMsg = '页面不存在(404)'
        } else if (error.message.includes('403')) {
          errorMsg = '页面访问被拒绝(403)'
        } else if (error.message.includes('timeout')) {
          errorMsg = '访问超时，请稍后重试'
        } else if (error.message.includes('URL格式无效')) {
          errorMsg = 'URL格式不正确，请检查输入'
        } else if (error.message.includes('URL验证失败')) {
          errorMsg = 'URL验证失败，请检查URL是否正确'
        } else {
          errorMsg = error.message
        }
      }
      
      wx.showToast({
        title: errorMsg,
        icon: 'none',
        duration: 3000
      })
    }
  },

  /**
   * 选择文件
   */
  selectFile() {
    wx.chooseMessageFile({
      count: 1,
      type: 'file',
      extension: ['pdf', 'doc', 'docx', 'txt', 'md'],
      success: (res) => {
        const file = res.tempFiles[0]
        console.log('选择的文件:', file)
        
        // 检查文件大小（10MB限制）
        if (file.size > 10 * 1024 * 1024) {
          wx.showToast({
            title: '文件大小不能超过10MB',
            icon: 'none'
          })
          return
        }
        
        this.setData({
          selectedFile: file,
          filePreview: {
            name: file.name,
            size: this.formatFileSize(file.size),
            type: this.getFileExtension(file.name)
          }
        })
      },
      fail: (error) => {
        console.error('选择文件失败:', error)
        wx.showToast({
          title: '选择文件失败',
          icon: 'none'
        })
      }
    })
  },

  /**
   * 上传文件
   */
  async uploadFile() {
    const { selectedFile } = this.data
    
    if (!selectedFile) {
      wx.showToast({
        title: '请先选择文件',
        icon: 'none'
      })
      return
    }

    this.setData({ 
      uploading: true,
      uploadProgress: 0
    })

    try {
      // 模拟上传进度
      const progressInterval = setInterval(() => {
        this.setData({
          uploadProgress: Math.min(this.data.uploadProgress + 10, 90)
        })
      }, 200)

      // 准备上传数据
      const uploadData = {
        filePath: selectedFile.path,
        name: 'file',
        formData: {
          filename: selectedFile.name,
          filesize: selectedFile.size
        }
      }

      const result = await contentAPI.uploadDocument(uploadData)
      
      clearInterval(progressInterval)
      
      this.setData({
        uploadProgress: 100,
        uploading: false,
        fileUploaded: true,
        uploadedFileInfo: result
      })
      
      wx.showToast({
        title: '文件上传成功',
        icon: 'success'
      })
    } catch (error) {
      console.error('文件上传失败:', error)
      this.setData({ 
        uploading: false,
        uploadProgress: 0
      })
      
      wx.showToast({
        title: '文件上传失败',
        icon: 'none'
      })
    }
  },

  /**
   * 移除文件
   */
  removeFile() {
    this.setData({
      selectedFile: null,
      filePreview: null,
      uploadProgress: 0,
      uploading: false,
      fileUploaded: false,
      uploadedFileInfo: null
    })
  },

  /**
   * 文本输入变化
   */
  onTextInput(e) {
    const text = e.detail.value
    this.setData({
      textInput: text,
      textPreview: text.length > 0 ? {
        wordCount: text.length,
        estimatedSlides: Math.ceil(text.length / 500) + 1
      } : null
    })
  },

  /**
   * 表单输入变化
   */
  onFormInput(e) {
    const { field } = e.currentTarget.dataset
    const value = e.detail.value
    
    this.setData({
      [`formData.${field}`]: value
    })
  },

  /**
   * 分类选择
   */
  onCategoryChange(e) {
    const index = e.detail.value
    const category = this.data.categoryOptions[index].value
    this.setData({
      'formData.category': category,
      categoryIndex: index
    })
  },

  /**
   * 语音类型选择
   */
  onVoiceTypeChange(e) {
    const index = e.detail.value
    const voiceType = this.data.voiceTypeOptions[index].value
    this.setData({
      'formData.voice_type': voiceType,
      voiceTypeIndex: index
    })
  },

  /**
   * 模板选择
   */
  onTemplateChange(e) {
    const index = e.detail.value
    const template = this.data.templateOptions[index].value
    this.setData({
      'formData.template': template,
      templateIndex: index
    })
  },

  /**
   * 幻灯片数量选择
   */
  onSlideCountChange(e) {
    const index = e.detail.value
    const slideCount = this.data.slideCountOptions[index].value
    
    // 验证页数
    const isValid = this.validateSlideCount(slideCount)
    
    this.setData({
      'formData.slide_count': slideCount,
      slideCountIndex: index
    })

    // 如果超出限制，显示提示
    if (!isValid) {
      wx.showToast({
        title: this.data.slideCountWarningText,
        icon: 'none',
        duration: 3000
      })
    }
  },

  /**
   * 目标受众选择
   */
  onAudienceChange(e) {
    const index = e.detail.value
    const audience = this.data.audienceOptions[index].value
    this.setData({
      'formData.audience': audience,
      audienceIndex: index
    })
  },

  /**
   * 难度等级选择
   */
  onDifficultyChange(e) {
    const index = e.detail.value
    const difficulty = this.data.difficultyOptions[index].value
    this.setData({
      'formData.difficulty': difficulty,
      difficultyIndex: index
    })
  },

  /**
   * 切换AI总结模式
   */
  toggleAISummary(e) {
    const useAISummary = e.detail.value
    this.setData({
      useAISummary: useAISummary,
      'formData.use_ai_summary': useAISummary,
      showSummaryOptions: useAISummary,
      summaryResult: null
    })
  },

  /**
   * 总结级别选择
   */
  onSummaryLevelChange(e) {
    const index = e.detail.value
    const summaryLevel = this.data.summaryLevelOptions[index].value
    this.setData({
      summaryLevel: summaryLevel,
      summaryLevelIndex: index
    })
  },

  /**
   * 总结目标受众选择
   */
  onSummaryAudienceChange(e) {
    const index = e.detail.value
    const summaryAudience = this.data.summaryAudienceOptions[index].value
    this.setData({
      summaryTargetAudience: summaryAudience,
      summaryAudienceIndex: index
    })
  },

  /**
   * 开始AI分析
   */
  async startAISummary() {
    if (!this.validateContentForSummary()) {
      return
    }

    this.setData({ generatingSummary: true })

    try {
      // 获取要分析的内容
      let content = ''
      let sourceType = ''

      switch (this.data.inputType) {
        case 'url':
          content = this.data.urlInput
          sourceType = 'url'
          break
        case 'file':
          if (this.data.filePreview && this.data.filePreview.content) {
            content = this.data.filePreview.content
          } else if (this.data.selectedFile) {
            content = this.data.selectedFile.path || this.data.selectedFile.name
          }
          sourceType = 'file'
          break
        case 'text':
          content = this.data.textInput
          sourceType = 'text'
          break
      }

      // 调用AI总结API
      const summaryResult = await contentAPI.generateAISummary({
        content: content,
        summary_level: this.data.summaryLevel,
        target_audience: this.data.summaryTargetAudience,
        source_type: sourceType
      })

      this.setData({
        summaryResult: summaryResult,
        generatingSummary: false
      })

      // 根据总结结果自动调整幻灯片数量
      if (summaryResult && summaryResult.suggested_slides) {
        const suggestedCount = Math.min(20, Math.max(5, summaryResult.suggested_slides))
        const slideCountIndex = this.data.slideCountOptions.findIndex(option => option.value === suggestedCount)
        
        if (slideCountIndex >= 0) {
          this.setData({
            'formData.slide_count': suggestedCount,
            slideCountIndex: slideCountIndex
          })
        }
      }

      wx.showToast({
        title: 'AI分析完成',
        icon: 'success'
      })

    } catch (error) {
      console.error('AI总结失败:', error)
      this.setData({ generatingSummary: false })
      
      wx.showToast({
        title: 'AI分析失败',
        icon: 'none'
      })
    }
  },

  /**
   * 确认增强生成
   */
  async confirmEnhancedGeneration() {
    if (!this.data.summaryResult) {
      wx.showToast({
        title: '请先进行AI分析',
        icon: 'none'
      })
      return
    }

    if (!this.validateForm()) {
      return
    }

    // 防止重复提交
    if (this.data.submitting || this.data.generating) {
      return
    }

    this.setData({ 
      submitting: true,
      generating: false, // 重置生成状态
      generationCompleted: false,
      generationProgress: 0,
      generationStatus: '准备生成...'
    })

    try {
      // 显示开始进度
      this.smoothUpdateProgress(5, '正在提交请求...')

      // 构建增强PPT生成请求数据 - 使用与submitCourse相同的格式
      const requestData = {
        source_type: this.data.inputType,
        content: this.getContentByType(),
        title: this.data.formData.title.trim() || '', // 如果标题为空，让后端自动生成
        slide_count: this.data.formData.slide_count,
        template: this.data.formData.template,
        style: 'professional',
        audience: this.data.formData.audience,
        difficulty: this.data.formData.difficulty,
        language: 'zh-CN',
        include_images: true,
        include_notes: this.data.formData.include_notes,
        auto_optimize: true,
        create_course: true,
        category: this.data.formData.category,
        tags: this.data.formData.tags.join(','),
        is_public: this.data.formData.is_public,
        generate_audio: this.data.formData.generate_audio,
        voice_type: this.data.formData.voice_type,
        // 附加AI总结结果
        summary_result: this.data.summaryResult,
        summary_level: this.data.summaryLevel,
        target_audience: this.data.summaryTargetAudience
      }

      console.log('增强PPT生成请求:', requestData)

      // 调用增强PPT生成API（带页数限制）- 与submitCourse使用相同接口
      const result = await contentAPI.generateEnhancedPPTWithLimits(requestData)

      console.log('增强PPT生成结果:', result)

      // 检查是否成功 - 与submitCourse使用相同逻辑
      const isSuccess = result && (
        result.success || 
        (result.data && result.data.success) ||
        result.course_id ||
        (result.data && result.data.course_id) ||
        result.ppt_file_path ||
        (result.data && result.data.ppt_file_path)
      )

      if (isSuccess) {
        // 提取课程ID - 与submitCourse使用相同逻辑
        let courseId = result.course_id;
        if (!courseId && result.data) {
          courseId = result.data.course_id;
        }
        if (!courseId) {
          courseId = result.courseId || result.id || result.course_id;
        }

        console.log('增强生成提取的课程ID:', courseId)

        if (courseId) {
          console.log('✅ 增强生成成功获取课程ID，开始生成过程:', courseId)
          this.startGeneration(courseId)
        } else {
          // 如果没有课程ID，尝试获取最新课程
          console.log('⚠️ 增强生成未获取到课程ID，尝试获取最新课程...')
          
          try {
            const latestCourseResponse = await courseAPI.getLatestCourse()
            if (latestCourseResponse && latestCourseResponse.data) {
              const latestCourse = latestCourseResponse.data
              console.log('🔄 增强生成获取到最新课程:', latestCourse.id, '状态:', latestCourse.status)
              
              courseId = latestCourse.id
              console.log('✅ 增强生成使用最新课程ID开始生成过程:', courseId)
              this.startGeneration(courseId)
              return
            }
          } catch (courseError) {
            console.error('增强生成获取最新课程失败:', courseError)
          }
          
          throw new Error('增强生成成功但课程ID获取失败')
        }
      } else {
        // 获取错误信息
        const errorMsg = result.error || 
                        (result.data && result.data.error) || 
                        result.message || 
                        (result.data && result.data.message) || 
                        '增强PPT生成失败'
        throw new Error(errorMsg)
      }

    } catch (error) {
      console.error('增强PPT生成失败:', error)
      this.setData({ 
        submitting: false,
        generating: false,
        generationCompleted: false 
      })
      
      wx.showModal({
        title: '生成失败',
        content: error.message || '增强PPT生成失败，请重试',
        showCancel: false,
        confirmText: '确定'
      })
    }
  },

  /**
   * 验证内容是否可以进行总结
   */
  validateContentForSummary() {
    const { inputType, urlInput, selectedFile, textInput } = this.data
    
    switch (inputType) {
      case 'url':
        if (!urlInput.trim()) {
          wx.showToast({
            title: '请输入URL',
            icon: 'none'
          })
          return false
        }
        break
      case 'file':
        if (!selectedFile) {
          wx.showToast({
            title: '请选择文件',
            icon: 'none'
          })
          return false
        }
        break
      case 'text':
        if (!textInput.trim()) {
          wx.showToast({
            title: '请输入文本内容',
            icon: 'none'
          })
          return false
        }
        if (textInput.length < 100) {
          wx.showToast({
            title: '文本内容至少100个字符',
            icon: 'none'
          })
          return false
        }
        break
    }
    
    return true
  },

  /**
   * 切换音频生成
   */
  onAudioToggle(e) {
    this.setData({
      'formData.generate_audio': e.detail.value
    })
  },

  /**
   * 选择语音类型
   */
  onVoiceTypeChange(e) {
    const index = parseInt(e.detail.value)
    this.setData({
      voiceTypeIndex: index,
      'formData.voice_type': this.data.voiceTypeOptions[index].value
    })
  },

  /**
   * 公开切换
   */
  onPublicToggle(e) {
    this.setData({
      'formData.is_public': e.detail.value
    })
  },

  /**
   * 延时工具方法
   */
  delay: function(ms) {
    return new Promise(function(resolve) {
      setTimeout(resolve, ms)
    })
  },

  /**
   * 🆕 AI增强课程生成（基于新的AI内容分析API）
   */
  async generateCourseWithAI() {
    // 防止重复提交
    if (this.data.submitting || this.data.generating) {
      return
    }

    // 检查登录状态
    if (!this.checkAuthStatus()) {
      console.log('用户未登录或token已过期，跳转到登录页面')
      this.requireUserLogin()
      return
    }

    // 验证表单
    if (!this.validateForm()) {
      return
    }

    // 目前只支持URL输入的AI分析
    if (this.data.inputType !== 'url') {
      wx.showToast({
        title: 'AI增强模式目前只支持URL输入',
        icon: 'none'
      })
      return
    }

    // 验证URL
    if (!this.data.urlInput.trim()) {
      wx.showToast({
        title: '请输入有效的URL',
        icon: 'none'
      })
      return
    }

    this.setData({ 
      submitting: true,
      generating: false,
      generationCompleted: false,
      generationProgress: 0,
      generationStatus: 'AI正在分析URL内容...'
    })

    try {
      // 🆕 根据选择的引擎使用不同的生成策略
      if (this.data.engineType === 'coze') {
        // Coze智能体一站式生成流程 - 🆕 优化进度更新
        this.smoothUpdateProgress(15, 'Coze智能体正在分析内容...')
        console.log('🎯 使用Coze智能体生成PPT，URL:', this.data.urlInput)
        
        // 模拟分析阶段
        await this.delay(1000)
        this.smoothUpdateProgress(30, '深度解析内容结构...')
        
        const cozeParams = {
          url: this.data.urlInput,
          topic: this.data.formData.title || '', // 不传递默认标题，让系统从内容中提取
          slides_count: this.data.formData.slide_count,
          template: this.data.formData.template || 'professional',
          options: {
            language: 'zh-CN',
            style: 'professional'
          }
        }

        this.smoothUpdateProgress(50, 'Coze工作流正在生成专业PPT...')
        const cozeResult = await cozeAPI.createCourseWithWorkflow(cozeParams)
        
        // 模拟生成过程
        await this.delay(1200)
        this.smoothUpdateProgress(75, '优化PPT结构和样式...')
        
        if (cozeResult.success) {
          console.log('✅ Coze PPT生成完成:', cozeResult.result)
          
          // 🆕 集成到课程系统
          this.smoothUpdateProgress(85, '正在集成到课程系统...')
          await this.delay(500)
          
          try {
            // 调用HTML集成API，将Coze生成的HTML转换为Course实体
            const courseResult = await this.integrateCozeHTMLToCourse(cozeResult.result)
            
            if (courseResult && courseResult.course_id) {
              console.log('✅ Coze HTML成功集成到课程系统，课程ID:', courseResult.course_id)
              
              // 最终完成
              await this.delay(300)
              this.setProgressComplete() // 直接设置为完成
              
              // 🆕 跳转到课程详情页面（图二界面）
              setTimeout(() => {
                wx.navigateTo({
                  url: `/pages/course/detail/detail?id=${courseResult.course_id}`
                })
              }, 1000)
            } else {
              throw new Error('课程集成失败：无法获取课程ID')
            }
          } catch (integrationError) {
            console.error('❌ Coze HTML集成失败:', integrationError)
            
            // 集成失败时降级到预览模式
            this.setProgressComplete()
            setTimeout(() => {
              if (cozeResult.result && cozeResult.result.id) {
                console.log('🔄 降级到PPT预览模式，任务ID:', cozeResult.result.id)
                wx.navigateTo({
                  url: `/pages/ppt-preview/ppt-preview?taskId=${cozeResult.result.id}`
                })
              } else {
                wx.showToast({
                  title: 'PPT生成完成，但集成失败',
                  icon: 'none'
                })
              }
            }, 1000)
          }
        } else {
          throw new Error(cozeResult.error || 'Coze智能体生成失败')
        }
        
      } else {
        // 原有的DashScope等引擎处理逻辑 - 🆕 优化进度更新
      // 第一步：AI分析URL内容
              this.smoothUpdateProgress(12, 'AI正在深度分析内容结构...')
      console.log('🤖 开始AI分析，URL:', this.data.urlInput)
      
      // 模拟分析阶段
      await this.delay(800)
              this.smoothUpdateProgress(25, '提取关键信息和结构...')
      
      const analysisParams = {
        url: this.data.urlInput,
        analysis_type: 'comprehensive',
        engine_type: this.data.engineType,
        language: 'zh-CN'
      }

      const analysisResult = await aiContentAPI.analyzeURL(analysisParams)
      console.log('🎯 AI分析完成:', analysisResult)

      // 第二步：基于分析结果生成PPT
              this.smoothUpdateProgress(45, '正在生成高质量PPT内容...')
      console.log('📊 开始生成PPT，分析结果:', analysisResult)
      
      // 模拟生成过程
      await this.delay(1000)
              this.smoothUpdateProgress(70, '优化内容布局和设计...')

      const pptParams = {
        ai_content: analysisResult,
        generation_params: {
          slide_count: this.data.formData.slide_count,
          style: 'professional',
          include_code_examples: true,
          include_best_practices: true,
          template: this.data.formData.template,
          language: 'zh-CN'
        }
      }

      const pptResult = await aiContentAPI.generatePPT(pptParams)
      console.log('✅ PPT生成完成:', pptResult)

      // 最终优化阶段
      await this.delay(600)
              this.smoothUpdateProgress(90, '正在保存和优化...')
      
      // 完成生成
      await this.delay(500)
      this.setProgressComplete() // 直接设置为完成

      // 成功后跳转到课程详情页
      setTimeout(() => {
        wx.navigateTo({
          url: `/pages/course/detail/detail?id=${pptResult.course_id}`
        })
      }, 1500)
      }

      this.setData({
        generating: false,
        generationCompleted: true,
        submitting: false
      })

    } catch (error) {
      console.error('❌ AI增强生成失败:', error)
      this.setData({ 
        submitting: false,
        generating: false,
        generationCompleted: false 
      })
      
      wx.showModal({
        title: 'AI生成失败',
        content: error.message || 'AI增强生成失败，请重试或切换到传统模式',
        showCancel: true,
        cancelText: '切换传统模式',
        confirmText: '重试',
        success: (res) => {
          if (res.cancel) {
            // 切换到传统模式
            this.setData({ generationMode: 'traditional' })
          } else {
            // 重试AI生成
            setTimeout(() => {
              this.generateCourseWithAI()
            }, 1000)
          }
        }
      })
    }
  },

  /**
   * 🆕 智能提交课程（根据生成模式选择不同的生成方法）
   */
  async submitCourse() {
    const { generationMode } = this.data
    
    console.log('📝 提交课程，生成模式:', generationMode)
    
    switch (generationMode) {
      case 'ai_enhanced':
      case 'ai_only':
        // 使用新的AI增强生成
        await this.generateCourseWithAI()
        break
      case 'traditional':
        // 使用传统生成方法
        await this.submitCourseTraditional()
        break
      default:
        // 默认使用AI增强
        await this.generateCourseWithAI()
        break
    }
  },

  /**
   * 🔄 传统课程生成方法（重命名原有的submitCourse）
   */
  async submitCourseTraditional() {
    // 防止重复提交
    if (this.data.submitting || this.data.generating) {
      return
    }

    // 检查登录状态
    if (!this.checkAuthStatus()) {
      console.log('用户未登录或token已过期，跳转到登录页面')
      this.requireUserLogin()
      return
    }

    // 验证表单
    if (!this.validateForm()) {
      return
    }

    // 验证文件上传（如果输入类型是文件）
    if (this.data.inputType === 'file') {
      if (!this.data.uploadedFileInfo || !this.data.uploadedFileInfo.file_path) {
        wx.showToast({
          title: '请先上传文件',
          icon: 'none'
        })
        return
      }
    }

    // 验证幻灯片数量
    if (!this.validateSlideCount(this.data.formData.slide_count)) {
      wx.showToast({
        title: this.data.slideCountWarningText,
        icon: 'none'
      })
      return
    }

    this.setData({ 
      submitting: true,
      generating: false,
      generationCompleted: false,
      generationProgress: 0,
      generationStatus: '准备生成...'
    })

    try {
      // 显示开始进度
      this.smoothUpdateProgress(5, '正在提交请求...')

      // 构建请求数据
      const requestData = {
        source_type: this.data.inputType,
        content: this.getContentByType(),
        title: this.data.formData.title.trim() || '', // 如果标题为空，让后端自动生成
        slide_count: this.data.formData.slide_count,
        template: this.data.formData.template,
        style: 'professional',
        audience: this.data.formData.audience,
        difficulty: this.data.formData.difficulty,
        language: 'zh-CN',
        include_images: true,
        include_notes: this.data.formData.include_notes,
        auto_optimize: true,
        create_course: true,
        category: this.data.formData.category,
        tags: this.data.formData.tags.join(','),
        is_public: this.data.formData.is_public,
        generate_audio: this.data.formData.generate_audio,
        voice_type: this.data.formData.voice_type
      }

      console.log('提交课程请求:', requestData)

      // 调用增强PPT生成API
      console.log('🚀 准备调用API，请求数据:', requestData)
      
      let result;
      try {
        result = await contentAPI.generateEnhancedPPTWithLimits(requestData)
      } catch (apiError) {
        console.error('❌ API调用异常:', apiError)
        throw new Error('API调用失败: ' + (apiError.message || '网络错误'))
      }
      
      console.log('🎯 API调用完成，原始结果:', result)
      console.log('🔍 结果类型检查:', {
        type: typeof result,
        isNull: result === null,
        isUndefined: result === undefined,
        isObject: typeof result === 'object',
        keys: result ? Object.keys(result) : 'N/A'
      })
      
      // 如果result为null或undefined，说明请求失败
      if (!result) {
        throw new Error('API调用失败，请检查网络连接或重新登录')
      }
      
      // 确保result是对象且不为空
      if (typeof result !== 'object' || Object.keys(result).length === 0) {
        console.error('❌ API返回的数据格式不正确:', result)
        throw new Error('API返回的数据格式不正确，请重试')
      }

      // 检查是否成功 - 根据API返回格式进行判断
      const isSuccess = result && (
        // 直接返回数据的情况
        result.success || 
        // 嵌套在data中的情况
        (result.data && result.data.success) ||
        // 有course_id就认为成功
        result.course_id ||
        (result.data && result.data.course_id) ||
        // 有ppt_file_path就认为成功
        result.ppt_file_path ||
        (result.data && result.data.ppt_file_path)
      )

      console.log('API响应解析:', {
        result: result,
        isSuccess: isSuccess,
        courseId: result.course_id,
        dataCourseId: result.data && result.data.course_id
      })

      if (isSuccess) {
        // 获取课程ID和结果数据 - 详细调试
        console.log('🔍 详细课程ID提取过程:')
        console.log('  - result:', result)
        console.log('  - result.course_id:', result.course_id)
        console.log('  - result.data:', result.data)
        console.log('  - result.data?.course_id:', result.data && result.data.course_id)
        
        // 多种方式尝试获取课程ID
        let courseId = result.course_id;
        if (!courseId && result.data) {
          courseId = result.data.course_id;
        }
        
        // 如果还是没有，尝试其他可能的字段名
        if (!courseId) {
          courseId = result.courseId || result.id || result.course_id;
        }
        
        // 最后的安全检查
        if (!courseId && typeof result === 'object') {
          const keys = Object.keys(result);
          console.log('🔍 所有可用字段:', keys);
          // 寻找任何包含id的字段
          const idFields = keys.filter(key => key.toLowerCase().includes('id'));
          console.log('🔍 包含ID的字段:', idFields);
          if (idFields.length > 0) {
            courseId = result[idFields[0]]; // 使用第一个ID字段
            console.log('🔄 使用备用ID字段:', idFields[0], '=', courseId);
          }
        }
        
        const resultData = result.data || result

        console.log('🎯 最终提取的课程ID:', courseId)
        console.log('🎯 最终结果数据:', resultData)

        // 开始生成过程或直接处理完成结果
        if (courseId) {
          console.log('✅ 成功获取课程ID，开始生成过程:', courseId)
          this.startGeneration(courseId)
        } else {
          // 如果没有获取到课程ID，尝试从数据库获取最新课程ID
          console.log('⚠️ 未能从API响应获取课程ID，尝试获取最新课程...')
          
          try {
            const latestCourseResponse = await courseAPI.getLatestCourse()
            if (latestCourseResponse && latestCourseResponse.data) {
              const latestCourse = latestCourseResponse.data
              console.log('🔄 获取到最新课程:', latestCourse.id, '状态:', latestCourse.status)
              
              // 使用最新课程的ID
              courseId = latestCourse.id
              console.log('✅ 使用最新课程ID开始生成过程:', courseId)
              this.startGeneration(courseId)
              return
            }
          } catch (courseError) {
            console.error('获取最新课程失败:', courseError)
            
            // 如果专用API失败，尝试通用课程列表API
            try {
              const coursesResponse = await courseAPI.getCourses({ page: 1, page_size: 1 })
              if (coursesResponse && coursesResponse.courses && coursesResponse.courses.length > 0) {
                const latestCourse = coursesResponse.courses[0]
                console.log('🔄 从课程列表获取到最新课程:', latestCourse.id, '状态:', latestCourse.status)
                
                courseId = latestCourse.id
                console.log('✅ 使用课程列表中的最新ID开始生成过程:', courseId)
                this.startGeneration(courseId)
                return
              }
            } catch (fallbackError) {
              console.error('备用获取课程列表也失败:', fallbackError)
            }
          }
          
          // 所有方法都失败了，抛出错误
          console.error('课程ID获取失败详情:', {
            result: result,
            isSuccess: isSuccess,
            courseId: result.course_id,
            dataCourseId: result.data && result.data.course_id,
            resultKeys: Object.keys(result || {}),
            hasSuccess: 'success' in result,
            successValue: result.success,
            hasPptFile: 'ppt_file_path' in result,
            pptFileValue: result.ppt_file_path
          })
          
          // 检查是否有PPT文件但没有课程ID的情况
          if (result.ppt_file_path && !courseId) {
            console.log('⚠️ 检测到PPT生成成功但课程ID缺失，尝试从文件路径推导...')
            throw new Error('PPT生成成功但课程ID获取异常。请前往"我的课程"查看生成的PPT，或重新尝试生成。')
          }
          
          throw new Error('未能获取到有效的课程ID。可能的原因：\n1. 登录状态已过期，请重新登录\n2. 网络连接问题\n3. 服务器错误\n\n请尝试重新登录后再试')
        }
      } else {
        // 获取错误信息
        const errorMsg = result.error || 
                        (result.data && result.data.error) || 
                        result.message || 
                        (result.data && result.data.message) || 
                        '生成PPT失败'
        throw new Error(errorMsg)
      }

    } catch (error) {
      console.error('提交课程失败:', error)
      this.setData({ 
        submitting: false,
        generating: false,
        generationCompleted: false 
      })
      
      // 处理页数限制错误
      if (error.message && error.message.includes('配额限制')) {
        wx.showModal({
          title: '页数限制',
          content: error.message,
          showCancel: false,
          confirmText: '了解'
        })
      } else if (error.message && error.message.includes('认证失败')) {
        // 认证失败，提示用户重新登录
        wx.showModal({
          title: '登录已过期',
          content: '您的登录状态已过期，请重新登录后再试',
          showCancel: true,
          cancelText: '取消',
          confirmText: '重新登录',
          success: (res) => {
            if (res.confirm) {
              wx.redirectTo({
                url: '/pages/login/login'
              })
            }
          }
        })
      } else {
        wx.showModal({
          title: '生成失败',
          content: error.message || '课程生成失败，请重试',
          showCancel: false,
          confirmText: '确定'
        })
      }
    }
  },

  /**
   * 根据输入类型获取内容
   */
  getContentByType() {
    switch (this.data.inputType) {
      case 'url':
        return this.data.urlInput
      case 'file':
        // 优先使用上传后的服务器文件路径
        if (this.data.uploadedFileInfo && this.data.uploadedFileInfo.file_path) {
          return this.data.uploadedFileInfo.file_path
        }
        // 降级使用本地文件路径（不推荐）
        return this.data.selectedFile ? this.data.selectedFile.path : ''
      case 'text':
        return this.data.textInput
      default:
        return ''
    }
  },

  /**
   * 开始生成过程
   */
  startGeneration(courseId) {
    this.setData({
      generating: true,
      generationProgress: 0,
      generationStatus: '开始生成...',
      generationCompleted: false,
      submitting: false // 开始生成后取消提交状态
    })
    
    // 开始初始进度动画 (0% -> 15%)
    this.smoothUpdateProgress(15, '正在初始化...')
    
    // 建立WebSocket连接
    this.connectWebSocket(courseId)
    
    // 开始轮询生成状态
    this.pollGenerationStatus(courseId)
    
    // 模拟初始阶段进度
    setTimeout(() => {
      if (this.data.generating && !this.data.generationCompleted) {
        this.smoothUpdateProgress(25, '解析文档...')
      }
    }, 2000)
    
    setTimeout(() => {
      if (this.data.generating && !this.data.generationCompleted) {
        this.smoothUpdateProgress(40, '分析内容...')
      }
    }, 5000)
  },

  /**
   * 连接WebSocket
   */
  connectWebSocket(courseId) {
    const wsUrl = `${getWSURLSync()}/generation/${courseId}`
    
    this.data.wsConnection = wx.connectSocket({
      url: wsUrl,
      success: () => {
        console.log('WebSocket连接成功')
      },
      fail: (error) => {
        console.error('WebSocket连接失败:', error)
      }
    })
    
    this.data.wsConnection.onMessage((res) => {
      try {
        const data = JSON.parse(res.data)
        this.handleWebSocketMessage(data)
      } catch (error) {
        console.error('解析WebSocket消息失败:', error)
      }
    })
    
    this.data.wsConnection.onError((error) => {
      console.error('WebSocket错误:', error)
    })
    
    this.data.wsConnection.onClose(() => {
      console.log('WebSocket连接关闭')
    })
  },

  /**
   * 处理WebSocket消息
   */
  handleWebSocketMessage(data) {
    switch (data.type) {
      case 'progress':
        this.updateGenerationProgress(data.progress, data.status)
        break
      case 'step_complete':
        this.updateGenerationStep(data.step, 'completed')
        break
      case 'generation_complete':
        this.onGenerationComplete(data.result)
        break
      case 'error':
        this.onGenerationError(data.error)
        break
    }
  },

  /**
   * 更新生成进度
   */
  updateGenerationProgress(progress, status) {
    // 确保进度值合理
    if (progress < 0) progress = 0
    if (progress > 100) progress = 100
    
    // 简化进度更新，只更新基本状态
    this.setData({
      generationProgress: progress,
      generationStatus: status || '生成中...'
    })
  },

  /**
   * 更新生成步骤
   */
  updateGenerationStep(stepIndex, status) {
    const steps = [...this.data.generationSteps]
    steps[stepIndex].status = status
    
    this.setData({
      generationSteps: steps
    })
  },

  /**
   * 生成完成
   */
  onGenerationComplete(result) {
    console.log('生成完成回调，接收到的result:', result)
    
    // 标记生成完成，防止进度被覆盖
    this.setData({
      generationCompleted: true
    })
    
    // 平滑进度到100%
    this.smoothUpdateProgress(100, '生成完成！')
    
    // 等待进度动画完成后处理跳转
    setTimeout(() => {
    this.setData({
      generating: false,
        submitting: false
    })
    
    this.closeWebSocket()
    
    // 提取课程ID
    let courseId = null
    if (result && result.courseId) {
      courseId = result.courseId
    } else if (result && result.status && result.status.id) {
      courseId = result.status.id
    } else if (result && result.id) {
      courseId = result.id
    }
    
    console.log('提取的课程ID:', courseId)
    
    // 验证课程ID
    if (!courseId || courseId === 'undefined' || isNaN(courseId)) {
      console.error('生成完成但课程ID无效:', courseId)
      wx.showModal({
        title: '生成完成',
        content: '课件生成完成，但无法跳转到详情页。请到课程列表查看。',
        showCancel: true,
        confirmText: '去课程列表',
        cancelText: '留在此页',
        success: (res) => {
          if (res.confirm) {
            wx.navigateTo({
              url: '/pages/course/list/list'
            })
            } else {
              // 重置状态，允许用户重新创建
              this.resetGenerationState()
          }
        }
      })
      return
    }
    
    // 确保ID是数字
    const numericId = parseInt(courseId, 10)
    if (isNaN(numericId) || numericId <= 0) {
      console.error('课程ID不是有效数字:', courseId)
      wx.showModal({
        title: '生成完成',
        content: '课件生成完成，但课程ID格式错误。请到课程列表查看。',
        showCancel: false,
        confirmText: '去课程列表',
        success: () => {
          wx.navigateTo({
            url: '/pages/course/list/list'
          })
        }
      })
      return
    }
    
    wx.showToast({
      title: '课件生成完成',
      icon: 'success'
    })
    
    // 跳转到课件详情页
    console.log('跳转到详情页，课程ID:', numericId)
    wx.navigateTo({
      url: `/pages/course/detail/detail?id=${numericId}`
    })
    }, 1000) // 等待1秒让用户看到100%进度
  },

  /**
   * 生成错误
   */
  onGenerationError(error) {
    console.error('生成错误:', error)
    
    this.setData({
      generating: false,
      generationStatus: '生成失败',
      generationCompleted: false,
      submitting: false
    })
    
    this.closeWebSocket()
    
    wx.showModal({
      title: '生成失败',
      content: error || '生成过程中发生错误，请重试',
      showCancel: true,
      confirmText: '重试',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          // 重置状态，允许重新生成
          this.resetGenerationState()
        } else {
          // 用户选择取消，也重置状态
          this.resetGenerationState()
        }
      }
    })
  },

  /**
   * 关闭WebSocket连接
   */
  closeWebSocket() {
    if (this.data.wsConnection) {
      this.data.wsConnection.close()
      this.data.wsConnection = null
    }
  },

  /**
   * 轮询生成状态
   */
  pollGenerationStatus(courseId) {
    console.log('开始轮询生成状态，课程ID:', courseId)
    
    // 验证courseId
    if (!courseId || courseId === 'undefined' || isNaN(courseId)) {
      console.error('轮询时课程ID无效:', courseId)
      this.onGenerationError('课程ID无效，无法查询生成状态')
      return
    }
    
    // 清理之前的轮询定时器
    if (this.data.pollTimer) {
      clearInterval(this.data.pollTimer)
    }
    
    const pollInterval = setInterval(async () => {
      // 检查是否已经完成或页面已销毁
      if (this.data.generationCompleted || !this.data.generating) {
        clearInterval(pollInterval)
        this.setData({ pollTimer: null })
        return
      }
      
      try {
        console.log('轮询生成状态，课程ID:', courseId)
        const response = await courseAPI.getGenerationStatus(courseId)
        
        // 增强响应处理
        if (!response) {
          console.error('生成状态响应为空')
          return
        }
        
        let status
        if (response.data) {
          status = response.data
        } else if (typeof response === 'object' && response.id) {
          status = response
        } else {
          console.error('生成状态响应格式错误:', response)
          return
        }
        
        console.log('生成状态:', status)
        
        if (status.status === 'completed') {
          // 确保返回的courseId正确
          const resultCourseId = status.id || status.course_id || courseId
          console.log('生成完成，使用课程ID:', resultCourseId)
          
          clearInterval(pollInterval)
          this.setData({ pollTimer: null })
          
          this.onGenerationComplete({
            courseId: resultCourseId,
            status: status
          })
        } else if (status.status === 'generating') {
          // 检查是否已经轮询了很长时间但仍在生成状态
          // 这可能意味着前端获取到了错误的课程ID
          if (!this.data.pollStartTime) {
            this.setData({ pollStartTime: Date.now() })
          }
          
          const pollDuration = Date.now() - this.data.pollStartTime
          if (pollDuration > 60000) { // 超过1分钟仍在生成状态
            console.log('⚠️ 轮询超时，可能获取了错误的课程ID，尝试获取最新课程')
            
            // 尝试获取用户最新的课程列表
            try {
              const coursesResponse = await courseAPI.getCourses({ page: 1, page_size: 1 })
              if (coursesResponse && coursesResponse.courses && coursesResponse.courses.length > 0) {
                const latestCourse = coursesResponse.courses[0]
                if (latestCourse.status === 'completed' && latestCourse.id !== courseId) {
                  console.log('🔄 发现最新完成的课程:', latestCourse.id)
                  
                  clearInterval(pollInterval)
                  this.setData({ pollTimer: null })
                  
                  this.onGenerationComplete({
                    courseId: latestCourse.id,
                    status: latestCourse
                  })
                  return
                }
              }
            } catch (error) {
              console.error('获取最新课程失败:', error)
            }
            
            // 如果没有找到更新的课程，继续轮询但提示用户
            console.log('⏰ 生成时间较长，请耐心等待...')
            this.setData({
              generationStatus: '生成时间较长，请耐心等待...'
            })
          }
        } else if (status.status === 'failed') {
          clearInterval(pollInterval)
          this.setData({ pollTimer: null })
          
          this.onGenerationError(status.error_message || status.error || '生成失败')
        } else {
          // 更新进度，确保不超过95%（给完成留余量）
          const progress = Math.min(status.progress || 0, 95)
          this.updateGenerationProgress(progress, status.message || status.status || '')
        }
      } catch (error) {
        console.error('轮询生成状态失败:', error)
        // 不立即停止轮询，给网络恢复的机会
      }
    }, 3000) // 每3秒查询一次
    
    // 保存定时器引用
    this.setData({ pollTimer: pollInterval })
    
    // 设置轮询超时（5分钟）
    setTimeout(() => {
      if (this.data.pollTimer === pollInterval) {
          clearInterval(pollInterval)
        this.setData({ pollTimer: null })
        
        if (this.data.generating && !this.data.generationCompleted) {
          this.onGenerationError('生成超时，请重试')
        }
      }
    }, 300000) // 5分钟超时
  },

  /**
   * 清理生成相关的资源
   */
  cleanupGeneration() {
    // 关闭WebSocket连接
    this.closeWebSocket()
    
    // 清理所有定时器
    if (this.data.progressTimer) {
      clearTimeout(this.data.progressTimer)
    }
    if (this.data.pollTimer) {
      clearInterval(this.data.pollTimer)
    }
    
    // 重置状态
    this.setData({
      progressTimer: null,
      pollTimer: null,
      generating: false,
      generationCompleted: false,
      submitting: false
    })
  },

  /**
   * 恢复进度动画
   */
  resumeProgressAnimation() {
    if (this.data.generating && !this.data.progressTimer) {
      this.startProgressAnimation(this.data.generationProgress)
    }
  },

  /**
   * 开始平滑进度动画
   */
  startProgressAnimation(targetProgress = 20) {
    if (this.data.progressTimer) {
      clearTimeout(this.data.progressTimer)
    }

    const currentProgress = this.data.generationProgress
    const increment = targetProgress > currentProgress ? 1 : 0
    
    if (increment === 0) return

    const timer = setInterval(() => {
      const newProgress = Math.min(this.data.generationProgress + increment, targetProgress)
      
      this.setData({
        generationProgress: newProgress
      })

      if (newProgress >= targetProgress) {
        clearInterval(timer)
        this.setData({ progressTimer: null })
      }
    }, 100) // 每100ms增加1%

    this.setData({ progressTimer: timer })
  },

  /**
   * 平滑更新进度到指定值 - 🆕 简化版本，避免兼容性问题
   */
  smoothUpdateProgress(targetProgress, status = '') {
    console.log('📊 更新进度:', targetProgress, status)
    
    if (targetProgress > 100) targetProgress = 100
    if (targetProgress < 0) targetProgress = 0

    // 清除之前的定时器
    if (this.data.progressTimer) {
      clearTimeout(this.data.progressTimer)
    }

    // 直接设置进度，简化逻辑
    this.setData({ 
      generationProgress: targetProgress,
      generationStatus: status || '生成中...',
      progressTimer: null
    })
  },



  /**
   * 🆕 改进的平滑进度更新 - 设置目标并启动持续更新
   */


  /**
   * 🆕 设置为完成状态
   */
  setProgressComplete() {
    this.setData({ 
      generationProgress: 100,
      generationStatus: '生成完成！'
    })
  },

  /**
   * 验证表单
   */
  validateForm() {
    const { inputType, urlInput, selectedFile, textInput, formData, generationMode } = this.data
    
    // 检查输入内容
    switch (inputType) {
      case 'url':
        if (!urlInput.trim()) {
          wx.showToast({
            title: '请输入URL',
            icon: 'none'
          })
          return false
        }
        break
      case 'file':
        if (!selectedFile) {
          wx.showToast({
            title: '请选择文件',
            icon: 'none'
          })
          return false
        }
        break
      case 'text':
        if (!textInput.trim()) {
          wx.showToast({
            title: '请输入文本内容',
            icon: 'none'
          })
          return false
        }
        if (textInput.length < 50) {
          wx.showToast({
            title: '文本内容至少50个字符',
            icon: 'none'
          })
          return false
        }
        break
    }
    
    // 检查表单数据 - 所有模式下标题都为可选
    // AI增强模式：AI会自动生成标题，无需用户填写
    // 传统快速模式：用户可选择填写标题，如不填写系统会根据内容自动生成
    console.log(`${generationMode}模式：标题为可选字段，系统将自动生成或使用用户输入的标题`)
    
    return true
  },

  /**
   * 格式化文件大小
   */
  formatFileSize(bytes) {
    if (bytes === 0) return '0 B'
    
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  },

  /**
   * 获取文件扩展名
   */
  getFileExtension(filename) {
    return filename.split('.').pop().toLowerCase()
  },

  /**
   * 分享小程序
   */
  onShareAppMessage() {
    return {
      title: 'AI课堂 - 智能课件生成',
      path: '/pages/index/index'
    }
  },

  /**
   * 重置生成状态
   */
  resetGenerationState() {
    this.cleanupGeneration()
    this.setData({
      generationProgress: 0,
      generationStatus: '',
      generating: false,
      generationCompleted: false,
      submitting: false
    })
  },

  /**
   * 加载页数限制信息
   */
  async loadSlideCountLimits() {
    try {
      // 检查登录状态
      const token = wx.getStorageSync('access_token')
      const loggedIn = this.checkAuthStatus()
      
      console.log('📊 加载页数限制 - 认证状态:', {
        hasToken: !!token,
        isLoggedIn: loggedIn
      })
      
      if (!loggedIn || !token) {
        console.log('❌ 用户未登录或无token，使用默认页数限制')
        this.setDefaultLimits()
        return
      }

      console.log('✅ 用户已登录，获取页数限制信息')
      const response = await contentAPI.getSlideCountLimits()
      console.log('📊 页数限制信息:', response)
      
      if (response && response.data) {
        const limits = response.data.limits
        const userType = response.data.user_type || 'regular'
        
        this.setData({
          slideCountLimits: limits,
          userType: userType,
          'formData.slide_count': Math.min(this.data.formData.slide_count, limits.user_max)
        })
        
        console.log('✅ 页数限制加载完成:', limits)
      } else {
        console.warn('⚠️ 页数限制响应格式异常，使用默认值')
        this.setDefaultLimits()
      }
    } catch (error) {
      console.error('❌ 加载页数限制失败:', error)
      
      // 检查是否是认证错误
      if (error.message && (error.message.includes('认证失败') || error.message.includes('401'))) {
        console.log('🔄 认证失败，清除认证信息并切换到游客模式')
        // 清除认证信息
        this.clearAuthData()
        
        // 切换到游客模式
        await this.switchToGuestMode()
      } else {
        console.warn('⚠️ 其他错误，使用默认限制')
        // 其他错误，使用默认限制
        this.setDefaultLimits()
      }
    }
  },

  /**
   * 加载支持的文件类型
   */
  async loadSupportedFileTypes() {
    try {
      const response = await contentAPI.getSupportedFileTypes()
      console.log('支持的文件类型:', response)
      
      if (response && response.data) {
        this.setData({
          supportedFileTypes: response.data.supported_types || ['.txt', '.pdf', '.docx']
        })
      }
    } catch (error) {
      console.error('加载支持文件类型失败:', error)
    }
  },

  /**
   * 验证幻灯片数量
   */
  validateSlideCount(slideCount) {
    const limits = this.data.slideCountLimits
    let isValid = true
    let warningText = ''

    if (slideCount < limits.min) {
      isValid = false
      warningText = `幻灯片数量不能少于${limits.min}页`
    } else if (slideCount > limits.user_max) {
      isValid = false
      warningText = `${this.data.userType}用户最多可生成${limits.user_max}页PPT`
    } else if (slideCount > limits.max) {
      isValid = false
      warningText = `幻灯片数量不能超过${limits.max}页`
    }

    this.setData({
      showSlideCountWarning: !isValid,
      slideCountWarningText: warningText
    })

    return isValid
  },

  /**
   * 🆕 初始化AI引擎状态
   */
  async initializeAIEngines() {
    this.setData({ engineLoading: true })
    
    try {
      console.log('🔧 初始化AI引擎状态...')
      
      // 获取可用的AI引擎列表
      const engines = await cozeAPI.getAIEngines()
      console.log('可用AI引擎:', engines)
      
      // 检查Coze引擎状态
      let cozeStatus = 'unavailable'
      const cozeEngine = engines.find(engine => engine.id === 'coze')
      
      if (cozeEngine && cozeEngine.status === 'available') {
        cozeStatus = 'available'
        
        // 获取Coze配置信息
        try {
          const cozeConfig = await cozeAPI.getCozeConfig()
          console.log('Coze配置信息:', cozeConfig)
          
          this.setData({
            cozeConfig: cozeConfig,
            cozeEngineStatus: 'available'
          })
        } catch (configError) {
          console.warn('获取Coze配置失败:', configError)
          cozeStatus = 'unavailable'
        }
      }
      
      this.setData({
        availableEngines: engines,
        cozeEngineStatus: cozeStatus,
        engineLoading: false
      })
      
      console.log('✅ AI引擎初始化完成，Coze状态:', cozeStatus)
      
    } catch (error) {
      console.error('❌ AI引擎初始化失败:', error)
      this.setData({
        engineLoading: false,
        cozeEngineStatus: 'unavailable'
      })
      
      // 静默失败，不影响用户体验
      console.warn('使用默认AI引擎配置')
    }
  },

  /**
   * 🆕 切换AI引擎
   */
  switchEngine(e) {
    const engine = e.currentTarget.dataset.engine
    
    // 支持DashScope和Coze引擎
    if (engine === 'dashscope') {
      this.setData({
        engineType: engine
      })
      wx.showToast({
        title: 'DashScope引擎：通用AI分析，质量稳定',
        icon: 'none',
        duration: 2000
      })
    } else if (engine === 'coze') {
      this.setData({
        engineType: engine
      })
      wx.showToast({
        title: 'Coze智能体：专业PPT生成，深度分析',
        icon: 'none',
        duration: 2000
      })
    } else {
      wx.showToast({
        title: '该引擎暂未开放，敬请期待',
        icon: 'none'
      })
    }
  },

  /**
   * 🆕 启动进度条动画
   */
  startProgressAnimation(targetProgress = 0) {
    // 清除之前的动画定时器
    if (this.data.progressAnimationTimer) {
      clearInterval(this.data.progressAnimationTimer)
    }

    const startProgress = this.data.progress
    const progressDiff = targetProgress - startProgress
    const animationDuration = Math.abs(progressDiff) * 20 // 每1%进度需要20ms
    const frameRate = 16 // 约60fps
    const totalFrames = Math.max(1, Math.floor(animationDuration / frameRate))
    const progressPerFrame = progressDiff / totalFrames
    
    let currentFrame = 0
    
    // 更新进度描述
    this.updateProgressDescription(targetProgress)

    const timer = setInterval(() => {
      currentFrame++
      const newProgress = startProgress + (progressPerFrame * currentFrame)
      
      this.setData({
        progress: Math.round(newProgress * 10) / 10 // 保留1位小数
      })
      
      if (currentFrame >= totalFrames) {
        clearInterval(timer)
        this.setData({
          progress: targetProgress,
          progressAnimationTimer: null
        })
      }
    }, frameRate)
    
    this.setData({
      progressAnimationTimer: timer,
      targetProgress: targetProgress
    })
  },

  /**
   * 🆕 更新进度描述
   */
  updateProgressDescription(progress) {
    const { progressSteps } = this.data
    let description = '准备生成...'
    
    // 找到最接近的进度步骤
    for (let i = progressSteps.length - 1; i >= 0; i--) {
      if (progress >= progressSteps[i].progress) {
        description = progressSteps[i].description
        break
      }
    }
    
    this.setData({
      progressDescription: description
    })
  },

  /**
   * 🆕 模拟进度更新（用于演示）
   */
  simulateProgress() {
    const { progressSteps } = this.data
    let currentStep = 0
    
    const updateStep = () => {
      if (currentStep < progressSteps.length) {
        const step = progressSteps[currentStep]
        this.startProgressAnimation(step.progress)
        currentStep++
        
        // 根据进度设置不同的延迟时间
        let delay = 1000
        if (step.progress <= 10) delay = 800
        else if (step.progress <= 40) delay = 1500
        else if (step.progress <= 80) delay = 2000
        else delay = 1000
        
        setTimeout(updateStep, delay)
      }
    }
    
    updateStep()
  },



  /**
   * 🆕 再次生成PPT
   */
  regeneratePPT() {
    console.log('🔄 用户点击再次生成')
    
    wx.showModal({
      title: '再次生成确认',
      content: '将清除当前内容，重新开始生成新的PPT，是否继续？',
      confirmText: '开始生成',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          this.resetGenerationState()
        }
      }
    })
  },

  /**
   * 🆕 重置生成状态，准备新一轮生成
   */
  resetGenerationState() {
    console.log('🔄 重置生成状态')
    
    // 清理定时器和WebSocket连接
    this.cleanupGeneration()
    
    // 重置所有生成相关状态
    this.setData({
      // 生成状态
      generating: false,
      generationCompleted: false,
      submitting: false,
      
      // 进度相关
      generationProgress: 0,
      generationStatus: '',
      
      // 重置验证状态
      urlValidation: {
        isValid: false,
        errors: [],
        warnings: []
      }
    })
    
    // 显示成功提示
    wx.showToast({
      title: '已重置，可以开始新的生成',
      icon: 'success',
      duration: 2000
    })
    
    console.log('✅ 生成状态已重置，用户可以开始新的生成流程')
  },

  /**
   * 🆕 快速重新生成（使用相同URL和设置）
   */
  quickRegenerate() {
    console.log('⚡ 快速重新生成')
    
    wx.showModal({
      title: '快速重新生成',
      content: '将使用当前的URL和设置重新生成PPT，是否继续？',
      confirmText: '立即生成',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          // 重置状态但保留当前输入
          this.setData({
            generating: false,
            generationCompleted: false,
            submitting: false,
            generationProgress: 0,
            generationStatus: '',
            progress: 0,
            targetProgress: 0,
            progressDescription: '准备生成...'
          })
          
          // 清理定时器
          this.cleanupGeneration()
          
          // 直接开始生成
          setTimeout(() => {
            this.submitCourse()
          }, 500)
        }
      }
    })
  },

  /**
   * 🆕 将Coze生成的HTML集成到课程系统
   * @param {Object} cozeResult - Coze生成结果
   * @returns {Promise}
   */
  async integrateCozeHTMLToCourse(cozeResult) {
    try {
      console.log('🔄 开始集成Coze HTML到课程系统:', cozeResult)
      
      // 获取任务ID
      const taskId = cozeResult.id || cozeResult.taskId
      if (!taskId) {
        throw new Error('无法获取任务ID')
      }

      console.log('🌐 通过HTML预览API获取HTML内容，任务ID:', taskId)
      
      // 获取HTML预览信息 - 使用已引入的cozeAPI
      const previewResponse = await cozeAPI.getTaskHTMLPreview(taskId)
      console.log('🌐 预览响应:', previewResponse)
      
      if (previewResponse.html_preview_url) {
        console.log('📝 获取到HTML预览URL:', previewResponse.html_preview_url)
        
        // 通过预览URL获取HTML内容
        const htmlContent = await this.fetchHTMLContentFromURL(previewResponse.html_preview_url)
        if (htmlContent) {
          console.log('📄 成功获取HTML内容，长度:', htmlContent.length)
          
          // 调用集成API（使用HTML内容）
          const integrationResult = await cozeIntegrationAPI.integrateHTMLContent(htmlContent, {
            source_url: this.data.urlInput,
            title: this.data.formData.title || '通过Coze智能体生成的课程',
            description: `基于${this.data.urlInput}的内容，通过Coze智能体生成的PPT课程`
          })

          console.log('✅ 集成成功:', integrationResult)
          return integrationResult
        }
      }

      throw new Error('无法获取HTML内容进行集成')
      
    } catch (error) {
      console.error('❌ 集成Coze HTML到课程系统失败:', error)
      throw error
    }
  },

  /**
   * 🆕 从URL获取HTML内容
   * @param {string} url - HTML预览URL
   * @returns {Promise<string>} - HTML内容
   */
  async fetchHTMLContentFromURL(url) {
    try {
      console.log('📡 开始从URL获取HTML内容:', url)
      
      // 确保URL是完整的
      let fullURL = url
      if (url.startsWith('/')) {
        // 相对路径，需要加上基础URL
        const app = getApp()
        let baseURL = app.globalData.baseURL || 'https://wangjibin-sc.wepie.com:9000'
        
        // 如果baseURL已包含/api/v1且相对路径也以/api/v1开头，需要去除重复
        if (baseURL.endsWith('/api/v1') && url.startsWith('/api/v1')) {
          baseURL = baseURL.replace('/api/v1', '')
        }
        
        fullURL = baseURL + url
        console.log('🔗 转换为完整URL:', fullURL)
      }
      
      // 在小程序环境中，通过wx.request获取内容
      return new Promise((resolve, reject) => {
        wx.request({
          url: fullURL,
          method: 'GET',
          header: {
            'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8'
          },
          success: (res) => {
            if (res.statusCode === 200) {
              console.log('✅ 成功获取HTML内容，状态码:', res.statusCode)
              console.log('📄 内容类型:', res.header['content-type'] || res.header['Content-Type'])
              resolve(res.data)
            } else {
              console.error('❌ 获取HTML内容失败，状态码:', res.statusCode)
              reject(new Error(`HTTP ${res.statusCode}: ${res.errMsg || '未知错误'}`))
            }
          },
          fail: (err) => {
            console.error('❌ 网络请求失败:', err)
            reject(new Error(`网络请求失败: ${err.errMsg || '未知网络错误'}`))
          }
        })
      })
      
    } catch (error) {
      console.error('❌ 获取HTML内容异常:', error)
      throw error
    }
  }
})