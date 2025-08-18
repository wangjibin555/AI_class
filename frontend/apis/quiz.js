// apis/quiz.js
const { get, post } = require('../utils/request.js')

const quizAPI = {
  // 根据课程生成练习
  generateQuiz: (courseId, options = {}) => {
    return post(`/courses/${courseId}/quizzes/generate`, options, {
      needAuth: true
    })
  },

  // 获取练习详情
  getQuiz: (quizId) => {
    return get(`/quiz/${quizId}`, {
      needAuth: true
    })
  },

  // 开始练习
  startQuiz: (quizId) => {
    return post(`/quiz/${quizId}/start`, {}, {
      needAuth: true
    })
  },

  // 提交答案
  submitAnswer: (attemptId, request) => {
    return post(`/quiz/attempts/${attemptId}/answer`, request, {
      needAuth: true
    })
  },

  // 提交练习
  submitQuiz: (attemptId) => {
    return post(`/quiz/attempts/${attemptId}/submit`, {}, {
      needAuth: true
    })
  },

  // 获取练习记录
  getAttempt: (attemptId) => {
    return get(`/quiz/attempts/${attemptId}`, {
      needAuth: true
    })
  },

  // 获取练习统计
  getQuizStats: (quizId) => {
    return get(`/quiz/${quizId}/stats`, {
      needAuth: true
    })
  },

  // 获取练习记录列表
  getAttempts: (params = {}) => {
    return get('/quiz/attempts', {
      params,
      needAuth: true
    })
  }
}

module.exports = quizAPI