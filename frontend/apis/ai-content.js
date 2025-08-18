const { request } = require('../utils/request.js')
const { getBaseURLSync } = require('../utils/config.js')

const aiContentAPI = {
  /**
   * AI URL分析
   * @param {Object} data - 分析参数
   * @param {string} data.url - 要分析的URL
   * @param {string} data.analysis_type - 分析类型 (comprehensive, summary, technical)
   * @param {string} data.engine_type - 引擎类型 (dashscope, coze, hybrid)
   * @param {string} data.language - 输出语言 (zh-CN, en-US)
   * @returns {Promise}
   */
  analyzeURL: (data) => {
    return request({
      url: `${getBaseURLSync()}/ai-content/url-analysis`,
      method: 'POST',
      data: data,
      needAuth: true,
      timeout: 120000 // AI分析可能需要更长时间（2分钟）
    })
  },

  /**
   * 基于AI分析结果生成PPT
   * @param {Object} data - 生成参数
   * @param {Object} data.ai_content - AI分析结果
   * @param {Object} data.generation_params - 生成参数
   * @returns {Promise}
   */
  generatePPT: (data) => {
    return request({
      url: `${getBaseURLSync()}/ai-content/generate-ppt`,
      method: 'POST',
      data: data,
      needAuth: true,
      timeout: 300000 // PPT生成需要更长时间（5分钟）
    })
  },

  /**
   * 获取分析历史记录
   * @param {Object} params - 查询参数
   * @param {number} params.page - 页码
   * @param {number} params.page_size - 每页数量
   * @param {string} params.engine_type - 引擎类型过滤
   * @returns {Promise}
   */
  getAnalysisHistory: (params = {}) => {
    return request({
      url: `${getBaseURLSync()}/ai-content/analysis-history`,
      method: 'GET',
      data: params,
      needAuth: true
    })
  },

  /**
   * 删除分析记录
   * @param {string|number} analysisId - 分析记录ID
   * @returns {Promise}
   */
  deleteAnalysis: (analysisId) => {
    return request({
      url: `${getBaseURLSync()}/ai-content/analysis/${analysisId}`,
      method: 'DELETE',
      needAuth: true
    })
  },

  /**
   * 获取AI引擎状态
   * @returns {Promise}
   */
  getEngineStatus: () => {
    return request({
      url: `${getBaseURLSync()}/ai-content/engine-status`,
      method: 'GET',
      needAuth: false
    })
  },

  /**
   * 一步完成：从URL到PPT生成
   * @param {Object} data - 完整生成参数
   * @param {string} data.url - 要分析的URL
   * @param {string} data.analysis_type - 分析类型
   * @param {string} data.engine_type - 引擎类型
   * @param {Object} data.generation_params - PPT生成参数
   * @returns {Promise}
   */
  async createCourseFromURL(data) {
    try {
      // 第一步：AI分析URL
      const analysisResult = await this.analyzeURL({
        url: data.url,
        analysis_type: data.analysis_type || 'comprehensive',
        engine_type: data.engine_type || 'dashscope',
        language: data.language || 'zh-CN'
      })

      // 第二步：基于分析结果生成PPT
      const pptResult = await this.generatePPT({
        ai_content: analysisResult,
        generation_params: data.generation_params || {
          slide_count: 15,
          style: 'professional',
          include_code_examples: true,
          include_best_practices: true,
          template: 'technical',
          language: 'zh-CN'
        }
      })

      return {
        success: true,
        data: {
          analysis: analysisResult,
          course: pptResult
        }
      }
    } catch (error) {
      throw error
    }
  }
}

module.exports = aiContentAPI 