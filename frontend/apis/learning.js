// apis/learning.js
const { get, post, put, delete: del } = require('../utils/request.js')

const learningAPI = {
  // 开始学习记录
  startLearning: (courseId, learningType = 'audio') => {
    return post('/learning/start', {
      course_id: courseId,
      learning_type: learningType,
      start_time: Date.now()
    }, {
      needAuth: true
    })
  },

  // 更新学习进度
  updateProgress: (learningRecordId, progressData) => {
    return put('/learning/progress', {
      learning_record_id: learningRecordId,
      ...progressData
    }, {
      needAuth: true
    })
  },

  // 获取学习记录
  getLearningRecord: (learningRecordId) => {
    return get(`/learning/record/${learningRecordId}`, {
      needAuth: true
    })
  },

  // 获取课程学习记录
  getCourseLearningRecord: (courseId) => {
    return get(`/learning/course/${courseId}`, {
      needAuth: true
    })
  },

  // 获取用户学习记录列表
  getLearningRecords: (params = {}) => {
    return get('/learning/records', {
      params,
      needAuth: true
    })
  },

  // 获取学习统计
  getLearningStats: (userId) => {
    return get(`/learning/statistics/${userId}`, {
      needAuth: true
    })
  },

  // 结束学习记录
  endLearning: (learningRecordId) => {
    return put('/learning/end', {
      learning_record_id: learningRecordId,
      end_time: Date.now()
    }, {
      needAuth: true
    })
  },

  // 暂停学习
  pauseLearning: (learningRecordId) => {
    return put('/learning/pause', {
      learning_record_id: learningRecordId,
      pause_time: Date.now()
    }, {
      needAuth: true
    })
  },

  // 恢复学习
  resumeLearning: (learningRecordId) => {
    return put('/learning/resume', {
      learning_record_id: learningRecordId,
      resume_time: Date.now()
    }, {
      needAuth: true
    })
  },

  // 记录幻灯片访问
  recordSlideAccess: (learningRecordId, slideIndex, audioTime = 0) => {
    return post('/learning/slide-access', {
      learning_record_id: learningRecordId,
      slide_index: slideIndex,
      audio_time: audioTime,
      access_time: Date.now()
    }, {
      needAuth: true
    })
  },

  // 标记幻灯片完成
  completeSlide: (learningRecordId, slideIndex) => {
    return post('/learning/slide-complete', {
      learning_record_id: learningRecordId,
      slide_index: slideIndex,
      complete_time: Date.now()
    }, {
      needAuth: true
    })
  },

  // 删除学习记录
  deleteLearningRecord: (learningRecordId) => {
    return del(`/learning/record/${learningRecordId}`, {
      needAuth: true
    })
  },

  // 获取学习成就
  getLearningAchievements: (userId) => {
    return get(`/learning/achievements/${userId}`, {
      needAuth: true
    })
  },

  // 获取学习排行榜
  getLearningRanking: (type = 'time', period = 'week') => {
    return get('/learning/ranking', {
      params: {
        type,
        period
      },
      needAuth: true
    })
  }
}

module.exports = learningAPI 