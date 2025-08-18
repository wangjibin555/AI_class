const { request, get, post, put, delete: del } = require('../utils/request.js')

const courseAPI = {
  // 获取课程列表
  getCourseList: async (page = 1, pageSize = 10, category = '') => {
    try {
      const params = {
        page: page,
        page_size: pageSize
      }
      if (category) {
        params.category = category
      }
      
      const response = await get('/courses', { params })
      return response
    } catch (error) {
      console.error('获取课程列表失败:', error)
      throw error
    }
  },

  // getCourses是getCourseList的别名，用于兼容性
  getCourses: async (params = {}) => {
    try {
      console.log('getCourses调用参数:', params)
      const response = await get('/courses', { params, needAuth: true })
      console.log('getCourses响应:', response)
      return response
    } catch (error) {
      console.error('获取课程列表失败:', error)
      throw error
    }
  },

  // 获取公开课件列表
  getPublicCourses: async (params = {}) => {
    try {
      console.log('getPublicCourses调用参数:', params)
      const response = await get('/courses/public', { params, needAuth: false })
      console.log('getPublicCourses响应:', response)
      return response
    } catch (error) {
      console.error('获取公开课件列表失败:', error)
      throw error
    }
  },

  // 获取课程详情
  getCourseDetail: async (courseId) => {
    try {
      const response = await get(`/courses/${courseId}`, { needAuth: true })
      return response
    } catch (error) {
      console.error('获取课程详情失败:', error)
      throw error
    }
  },

  // 创建课程
  createCourse: async (courseData) => {
    try {
      console.log('创建课程数据:', courseData)
      
      // 转换前端数据格式为后端期望的格式
      const requestData = {
        title: courseData.title,
        description: courseData.description || '',
        category: courseData.category || 'general',
        tags: courseData.tags || [],
        source_type: courseData.source_type || (courseData.type === 'file' ? 'document' : courseData.type),
        source_content: courseData.source_content || courseData.content,
        source_url: courseData.source_url || courseData.url || (courseData.type === 'url' ? courseData.content : ''),
        is_public: courseData.is_public || false,
        		voice_type: courseData.voice_type || 'zhixiaobai',
        generation_params: courseData.generation_params || {
          template: courseData.template || 'business',
          slide_count: courseData.slide_count || 10,
          audience: courseData.audience || 'general',
          difficulty: courseData.difficulty || 'intermediate'
        }
      }
      
      console.log('发送的请求数据:', requestData)
      const response = await post('/courses', requestData, { needAuth: true })
      return response
    } catch (error) {
      console.error('创建课程失败:', error)
      throw error
    }
  },

  // 从URL创建课程
  createCourseFromUrl: async (url, formData) => {
    try {
      console.log('从URL创建课程:', url, formData)
      
      const requestData = {
        title: formData.title,
        description: formData.description || '',
        category: formData.category || 'general',
        tags: formData.tags || [],
        source_type: 'url',
        source_content: url,
        source_url: url,
        is_public: formData.is_public || false,
        voice_type: formData.voice_type || 'zhixiaobai',
        generation_params: {
          template: formData.template || 'business',
          slide_count: formData.slide_count || 10,
          audience: formData.audience || 'general',
          difficulty: formData.difficulty || 'intermediate'
        }
      }
      
      console.log('发送的URL课程数据:', requestData)
      const response = await post('/courses', requestData, { needAuth: true })
      return response
    } catch (error) {
      console.error('从URL创建课程失败:', error)
      throw error
    }
  },

  // 从文件创建课程
  createCourseFromFile: async (fileData, formData) => {
    try {
      console.log('从文件创建课程:', fileData, formData)
      
      const requestData = {
        title: formData.title,
        description: formData.description || '',
        category: formData.category || 'general',
        tags: formData.tags || [],
        source_type: 'document',
        source_content: fileData.content || fileData.path || '',
        source_url: fileData.url || '',
        is_public: formData.is_public || false,
        voice_type: formData.voice_type || 'zhixiaobai',
        generation_params: {
          template: formData.template || 'business',
          slide_count: formData.slide_count || 10,
          audience: formData.audience || 'general',
          difficulty: formData.difficulty || 'intermediate'
        }
      }
      
      console.log('发送的文件课程数据:', requestData)
      const response = await post('/courses', requestData, { needAuth: true })
      return response
    } catch (error) {
      console.error('从文件创建课程失败:', error)
      throw error
    }
  },

  // 从文本创建课程
  createCourseFromText: async (text, formData) => {
    try {
      console.log('从文本创建课程:', text, formData)
      
      const requestData = {
        title: formData.title,
        description: formData.description || '',
        category: formData.category || 'general',
        tags: formData.tags || [],
        source_type: 'text',
        source_content: text,
        source_url: '',
        is_public: formData.is_public || false,
        voice_type: formData.voice_type || 'zhixiaobai',
        generation_params: {
          template: formData.template || 'business',
          slide_count: formData.slide_count || 10,
          audience: formData.audience || 'general',
          difficulty: formData.difficulty || 'intermediate'
        }
      }
      
      console.log('发送的文本课程数据:', requestData)
      const response = await post('/courses', requestData, { needAuth: true })
      return response
    } catch (error) {
      console.error('从文本创建课程失败:', error)
      throw error
    }
  },

  // 更新课程
  updateCourse: async (courseId, courseData) => {
    try {
      const response = await put(`/courses/${courseId}`, courseData, { needAuth: true })
      return response
    } catch (error) {
      console.error('更新课程失败:', error)
      throw error
    }
  },

  // 删除课程
  deleteCourse: async (courseId) => {
    try {
      const response = await del(`/courses/${courseId}`, { needAuth: true })
      return response
    } catch (error) {
      console.error('删除课程失败:', error)
      throw error
    }
  },

  // 获取我的课程
  getMyCourses: async (page = 1, pageSize = 10) => {
    try {
      const response = await get('/courses/my', {
        params: {
          page: page,
          page_size: pageSize
        },
        needAuth: true
      })
      return response
    } catch (error) {
      console.error('获取我的课程失败:', error)
      throw error
    }
  },

  // 获取最新课程
  getLatestCourse: async () => {
    try {
      const response = await get('/courses/latest', { needAuth: true })
      return response
    } catch (error) {
      console.error('获取最新课程失败:', error)
      throw error
    }
  },

  // 获取公开课程
  getPublicCourses: async (page = 1, pageSize = 10) => {
    try {
      const response = await get('/courses/public', {
        params: {
          page: page,
          page_size: pageSize
        }
      })
      return response
    } catch (error) {
      console.error('获取公开课程失败:', error)
      throw error
    }
  },

  // 搜索课程
  searchCourses: async (keyword, page = 1, pageSize = 10) => {
    try {
      const response = await get('/courses/search', {
        params: {
          keyword: keyword,
          page: page,
          page_size: pageSize
        }
      })
      return response
    } catch (error) {
      console.error('搜索课程失败:', error)
      throw error
    }
  },

  // 获取课程分类
  getCourseCategories: async () => {
    try {
      const response = await get('/courses/categories')
      return response
    } catch (error) {
      console.error('获取课程分类失败:', error)
      // 返回默认分类
      return [
        { value: 'general', label: '通用' },
        { value: 'programming', label: '编程' },
        { value: 'business', label: '商业' },
        { value: 'education', label: '教育' },
        { value: 'science', label: '科学' },
        { value: 'literature', label: '文学' },
        { value: 'history', label: '历史' },
        { value: 'art', label: '艺术' }
      ]
    }
  },

  // 获取生成状态
  getGenerationStatus: async (courseId) => {
    try {
      const response = await get(`/courses/${courseId}/generation-status`, { needAuth: true })
      return response
    } catch (error) {
      console.error('获取生成状态失败:', error)
      throw error
    }
  },

  // 重新生成课程
  regenerateCourse: async (courseId, params = {}) => {
    try {
      const response = await post(`/courses/${courseId}/regenerate`, params, { needAuth: true })
      return response
    } catch (error) {
      console.error('重新生成课程失败:', error)
      throw error
    }
  },

  // 下载课程
  downloadCourse: async (courseId) => {
    try {
      const response = await get(`/courses/${courseId}/download`, {
        responseType: 'blob',
        needAuth: true
      })
      return response
    } catch (error) {
      console.error('下载课程失败:', error)
      throw error
    }
  },

  // 分享课程
  shareCourse: async (courseId, shareData) => {
    try {
      const response = await post(`/courses/${courseId}/share`, shareData, { needAuth: true })
      return response
    } catch (error) {
      console.error('分享课程失败:', error)
      throw error
    }
  },

  // 收藏课程
  favoriteCourse: async (courseId) => {
    try {
      const response = await post(`/courses/${courseId}/favorite`, {}, { needAuth: true })
      return response
    } catch (error) {
      console.error('收藏课程失败:', error)
      throw error
    }
  },

  // 取消收藏课程
  unfavoriteCourse: async (courseId) => {
    try {
      const response = await del(`/courses/${courseId}/favorite`, { needAuth: true })
      return response
    } catch (error) {
      console.error('取消收藏课程失败:', error)
      throw error
    }
  },

  // 获取收藏的课程
  getFavoriteCourses: async (page = 1, pageSize = 10) => {
    try {
      const response = await get('/courses/favorites', {
        params: {
          page: page,
          page_size: pageSize
        },
        needAuth: true
      })
      return response
    } catch (error) {
      console.error('获取收藏课程失败:', error)
      throw error
    }
  },

  // 获取课程统计信息
  getCourseStats: async () => {
    try {
      const response = await get('/courses/stats', { needAuth: true })
      return response
    } catch (error) {
      console.error('获取课程统计失败:', error)
      throw error
    }
  },

  // 获取课程历史记录
  getCourseHistory: async (page = 1, pageSize = 10) => {
    try {
      const response = await get('/courses/history', {
        params: {
          page: page,
          page_size: pageSize
        },
        needAuth: true
      })
      return response
    } catch (error) {
      console.error('获取课程历史失败:', error)
      throw error
    }
  },

  // 复制课程
  copyCourse: async (courseId) => {
    try {
      const response = await post(`/courses/${courseId}/copy`, {}, { needAuth: true })
      return response
    } catch (error) {
      console.error('复制课程失败:', error)
      throw error
    }
  },

  // 获取课程评论
  getCourseComments: async (courseId, page = 1, pageSize = 10) => {
    try {
      const response = await get(`/courses/${courseId}/comments`, {
        params: {
          page: page,
          page_size: pageSize
        }
      })
      return response
    } catch (error) {
      console.error('获取课程评论失败:', error)
      throw error
    }
  },

  // 添加课程评论
  addCourseComment: async (courseId, comment) => {
    try {
      const response = await post(`/courses/${courseId}/comments`, {
        content: comment
      })
      return response
    } catch (error) {
      console.error('添加课程评论失败:', error)
      throw error
    }
  },

  // 删除课程评论
  deleteCourseComment: async (courseId, commentId) => {
    try {
      const response = await del(`/courses/${courseId}/comments/${commentId}`)
      return response
    } catch (error) {
      console.error('删除课程评论失败:', error)
      throw error
    }
  },

  // 获取所有标签
  getTags: async () => {
    try {
      const response = await get('/courses/tags')
      return response
    } catch (error) {
      console.error('获取标签失败:', error)
      throw error
    }
  },

  // 根据标签获取课程
  getCoursesByTag: async (tag) => {
    try {
      const response = await get(`/courses/tag/${tag}`, {
        needAuth: true
      })
      return response
    } catch (error) {
      console.error('根据标签获取课程失败:', error)
      throw error
    }
  },

  // 获取学习记录
  getLearningRecords: async (params = {}) => {
    try {
      console.log('获取学习记录参数:', params)
      const response = await get('/learning/records', {
        params: params,
        needAuth: true
      })
      console.log('学习记录响应:', response)
      
      // 处理后端响应数据结构
      const responseData = response.data || response || {}
      const result = {
        records: responseData.records || [],
        total: responseData.total || 0,
        page: responseData.page || 1,
        page_size: responseData.page_size || 10
      }
      
      // 处理学习记录数据，确保字段格式正确
      result.records = result.records.map(record => ({
        ...record,
        course_title: record.Course?.title || record.course_title || '未知课程',
        course_category: record.Course?.category || record.course_category || '通用',
        // 后端已转换为0-100范围，直接使用complete_rate
        progress: record.complete_rate || 0,
        progress_percentage: Math.round(record.complete_rate || 0),
        study_duration: record.study_duration || 0,
        last_study_time: record.updated_at || record.last_study_time,
        quiz_attempts: record.quiz_attempts || 0,
        highest_score: record.highest_score || 0,
        average_score: record.average_score || 0
      }))
      
      console.log('处理后的学习记录:', result)
      return result
    } catch (error) {
      console.error('获取学习记录失败:', error)
      // 返回空数据以避免页面崩溃
      return {
        records: [],
        total: 0,
        page: 1,
        page_size: 10
      }
    }
  },

  // 获取学习统计信息
  getLearningStats: async () => {
    try {
      const response = await get('/learning/stats', { needAuth: true })
      console.log('学习统计响应:', response)
      
      // 映射后端数据结构到前端预期格式
      const backendData = response.data || response || {}
      const mappedData = {
        totalCourses: backendData.total_courses || 0,
        completedCourses: backendData.completed_courses || 0,
        inProgressCourses: (backendData.total_courses || 0) - (backendData.completed_courses || 0),
        totalStudyTime: backendData.total_study_time || 0,
        totalQuizScore: 0, // 暂时设为0，后续可根据需要添加
        averageProgress: backendData.average_progress || 0
      }
      
      console.log('映射后的学习统计:', mappedData)
      return mappedData
    } catch (error) {
      console.error('获取学习统计失败:', error)
      // 返回默认统计信息以避免页面崩溃
      return {
        totalCourses: 0,
        completedCourses: 0,
        inProgressCourses: 0,
        totalStudyTime: 0,
        totalQuizScore: 0,
        averageProgress: 0
      }
    }
  },

  // 重置学习进度
  resetLearningProgress: async (courseId) => {
    try {
      const response = await post(`/learning/courses/${courseId}/reset`, {}, { needAuth: true })
      return response
    } catch (error) {
      console.error('重置学习进度失败:', error)
      throw error
    }
  },

  // 删除学习记录
  deleteLearningRecord: async (recordId) => {
    try {
      const response = await del(`/learning/record/${recordId}`, { needAuth: true })
      return response
    } catch (error) {
      console.error('删除学习记录失败:', error)
      throw error
    }
  },

  // 记录学习开始
  recordLearningStart: async (courseId, params = {}) => {
    try {
      const response = await post(`/learning/courses/${courseId}/start`, params, { needAuth: true })
      return response
    } catch (error) {
      console.error('记录学习开始失败:', error)
      throw error
    }
  },

  // 记录学习进度
  recordLearningProgress: async (courseId, progress) => {
    try {
      const response = await post(`/learning/courses/${courseId}/progress`, {
        progress: progress,
        timestamp: new Date().getTime()
      }, { needAuth: true })
      return response
    } catch (error) {
      console.error('记录学习进度失败:', error)
      throw error
    }
  },

  // 完成学习
  completeLearning: async (courseId, completionData = {}) => {
    try {
      const response = await post(`/courses/${courseId}/complete`, completionData, { needAuth: true })
      return response
    } catch (error) {
      console.error('完成学习记录失败:', error)
      throw error
    }
  },

  // 获取学习进度
  getLearningProgress: async (courseId) => {
    try {
      const response = await get(`/learning/courses/${courseId}/progress`, { needAuth: true })
      return response
    } catch (error) {
      console.error('获取学习进度失败:', error)
      return { progress: 0 }
    }
  },

  // 🔥 新增：completeCourse方法作为completeLearning的别名
  completeCourse: async (courseId, completionData = {}) => {
    try {
      console.log('🎓 [API] 完成课程:', courseId, completionData)
      const response = await post(`/courses/${courseId}/complete`, completionData, { needAuth: true })
      return response
    } catch (error) {
      console.error('完成课程失败:', error)
      throw error
    }
  },

  // 检查课程是否已有练习
  checkCourseQuiz: async (courseId) => {
    try {
      console.log('🔍 [API] 检查课程练习:', courseId)
      const response = await get(`/exercises/check/${courseId}`, { needAuth: true })
      console.log('🔍 [API] 检查练习响应:', response)
      return response
    } catch (error) {
      console.log('🔍 [API] 检查练习失败:', error.message)
      // 如果API不存在，返回默认值
      if (error.message && error.message.includes('404')) {
        return { exists: false, quiz_id: null }
      }
      throw error
    }
  },

  // 生成练习
  generateExercise: async (courseId, options = {}) => {
    try {
      const data = {
        course_id: courseId,
        quiz_title: options.title || '智能生成练习',
        difficulty: options.difficulty || 'normal',
        question_count: options.questionCount || 8
      }
      
      console.log('🧠 [API] 生成练习请求:', courseId, data)
      const response = await post('/exercises/generate', data, { needAuth: true })
      console.log('🧠 [API] 生成练习响应:', response)
      console.log('🧠 [API] 响应类型:', typeof response, '键:', Object.keys(response || {}))
      return response
    } catch (error) {
      console.error('🧠 [API] 生成练习失败:', error)
      throw error
    }
  },



}

module.exports = courseAPI