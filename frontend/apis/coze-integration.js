const { request } = require('../utils/request.js')
const { getBaseURLSync } = require('../utils/config.js')

const cozeIntegrationAPI = {
  /**
   * 将Coze生成的HTML内容集成到课程系统
   * @param {string} htmlContent - Coze生成的HTML字符串
   * @param {object} params - 其他集成参数，如title, description, source_url
   * @returns {Promise<object>} - 集成结果，包含course_id等
   */
  integrateHTMLContent: async (htmlContent, params = {}) => {
    const data = {
      html_content: htmlContent,
      title: params.title,
      description: params.description,
      source_url: params.source_url
    }
    return request({
      url: `${getBaseURLSync()}/coze-integration/html-content`,
      method: 'POST',
      data: data,
      needAuth: true,
      timeout: 300000 // 适当延长超时时间
    })
  },

  /**
   * 从HTML文件路径集成到课程系统（备用方法）
   * @param {Object} data - 集成参数
   * @param {string} data.html_file_path - HTML文件路径
   * @param {string} data.source_url - 源URL（可选）
   * @param {string} data.title - 自定义标题（可选）
   * @param {string} data.description - 自定义描述（可选）
   * @returns {Promise}
   */
  integrateFromFile: (data) => {
    return request({
      url: `${getBaseURLSync()}/coze-integration/html-file`,
      method: 'POST',
      data: data,
      needAuth: true,
      timeout: 60000 // 1分钟超时
    })
  },

  /**
   * 从上传的HTML文件集成到课程系统
   * @param {File} file - HTML文件
   * @param {Object} options - 选项
   * @param {string} options.source_url - 源URL（可选）
   * @param {string} options.title - 自定义标题（可选）
   * @param {string} options.description - 自定义描述（可选）
   * @returns {Promise}
   */
  integrateFromUpload: (file, options = {}) => {
    const formData = new FormData()
    formData.append('file', file)
    
    if (options.source_url) {
      formData.append('source_url', options.source_url)
    }
    if (options.title) {
      formData.append('title', options.title)
    }
    if (options.description) {
      formData.append('description', options.description)
    }

    return request({
      url: `${getBaseURLSync()}/coze/integrate/upload`,
      method: 'POST',
      data: formData,
      needAuth: true,
      timeout: 60000,
      header: {
        'Content-Type': 'multipart/form-data'
      }
    })
  },

  /**
   * 获取课程详情
   * @param {number} courseId - 课程ID
   * @returns {Promise}
   */
  getCourse: (courseId) => {
    return request({
      url: `${getBaseURLSync()}/coze/courses/${courseId}`,
      method: 'GET',
      needAuth: true
    })
  },

  /**
   * 获取Coze集成的课程列表
   * @param {Object} params - 查询参数
   * @param {number} params.page - 页码（默认1）
   * @param {number} params.page_size - 每页数量（默认10）
   * @returns {Promise}
   */
  getCourses: (params = {}) => {
    const queryParams = {
      page: params.page || 1,
      page_size: params.page_size || 10
    }

    return request({
      url: `${getBaseURLSync()}/coze/courses`,
      method: 'GET',
      data: queryParams,
      needAuth: true
    })
  },

  /**
   * 更新课程信息
   * @param {number} courseId - 课程ID
   * @param {Object} data - 更新数据
   * @param {string} data.title - 新标题（可选）
   * @param {string} data.description - 新描述（可选）
   * @returns {Promise}
   */
  updateCourse: (courseId, data) => {
    return request({
      url: `${getBaseURLSync()}/coze/courses/${courseId}`,
      method: 'PUT',
      data: data,
      needAuth: true
    })
  },

  /**
   * 删除课程
   * @param {number} courseId - 课程ID
   * @returns {Promise}
   */
  deleteCourse: (courseId) => {
    return request({
      url: `${getBaseURLSync()}/coze/courses/${courseId}`,
      method: 'DELETE',
      needAuth: true
    })
  },

  /**
   * 获取课程统计信息
   * @returns {Promise}
   */
  getStats: () => {
    return request({
      url: `${getBaseURLSync()}/coze/stats`,
      method: 'GET',
      needAuth: true
    })
  }
}

module.exports = cozeIntegrationAPI