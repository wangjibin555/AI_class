// apis/content.js
const request = require('../utils/request.js')

// 内容API模块
const contentAPI = {
  // 处理内容
  processContent: (data) => {
    return request.post('/content/process', data, { needAuth: true })
  },

  // 生成AI总结
  generateAISummary: (data) => {
    return request.post('/content/ai-summary', data, { needAuth: true })
  },

  // 生成增强PPT
  generateEnhancedPPT: (data) => {
    return request.post('/content/enhanced-ppt', data, { 
      needAuth: true, 
      timeout: 300000 // 5分钟超时，适应PPT生成的耗时操作
    })
  },

  // 生成增强PPT（带页数限制）
  generateEnhancedPPTWithLimits: (data) => {
    return request.post('/content/enhanced-ppt-limits', data, { 
      needAuth: true, 
      timeout: 300000 // 5分钟超时，适应PPT生成的耗时操作
    })
  },

  // 获取页数限制信息
  getSlideCountLimits: (userType = '') => {
    const params = userType ? { user_type: userType } : {}
    const options = { 
      needAuth: true,
      data: params
    }
    return request.get('/content/slide-limits', options)
  },

  // 处理文件内容
  processFile: (data) => {
    return request.post('/content/process-file', data, { needAuth: true })
  },

  // 上传文档文件
  uploadDocument: (fileData) => {
    return request.post('/ppt/upload', fileData, { 
      needAuth: true,
      isFormData: true
    })
  },

  // 生成PPT
  generatePPT: (data) => {
    return request.post('/ppt/generate', data, { 
      needAuth: true, 
      timeout: 300000 // 5分钟超时，适应PPT生成的耗时操作
    })
  },

  // 获取PPT生成状态
  getPPTGenerationStatus: (id) => {
    return request.get(`/ppt/status/${id}`, { needAuth: true })
  },

  // 预览PPT
  previewPPT: (filename) => {
    return request.get(`/ppt/preview/${filename}`, { needAuth: true })
  },

  // 下载PPT
  downloadPPT: (filename) => {
    return request.get(`/ppt/download/${filename}`, { needAuth: true })
  },

  // 获取PPT模板
  getPPTTemplates: () => {
    return request.get('/ppt/templates')
  },

  // 获取支持的文件类型
  getSupportedFileTypes: () => {
    return request.get('/content/supported-types')
  },

  // 验证URL（原方法名）
  validateURL: (url) => {
    return request.post('/ppt/validate-url', { url })
  },

  // 验证URL（别名，保持向后兼容）
  validateUrl: (url) => {
    return request.post('/ppt/validate-url', { url })
  },

  // 获取URL预览
  getURLPreview: (url) => {
    return request.post('/content/url-preview', { url })
  }
}

module.exports = contentAPI