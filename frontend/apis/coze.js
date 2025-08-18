const { request, get, post, put, delete: del } = require('../utils/request.js')
const { getBaseURL } = require('../utils/config.js')

/**
 * Coze智能体API封装
 * 基于已实现的后端API接口进行前端集成
 */
const cozeAPI = {
  
  /**
   * 获取可用的AI引擎列表
   * @returns {Promise}
   */
  getAIEngines: async () => {
    try {
      const response = await get('/ai-engines', { needAuth: false })
      return response
    } catch (error) {
      console.error('获取AI引擎列表失败:', error)
      throw error
    }
  },

  /**
   * 获取Coze引擎配置信息
   * @returns {Promise}
   */
  getCozeConfig: async () => {
    try {
      const response = await get('/ai-engines/coze/config', { needAuth: false })
      return response
    } catch (error) {
      console.error('获取Coze配置失败:', error)
      throw error
    }
  },

  /**
   * 使用Coze智能体生成PPT
   * @param {Object} data - 生成参数
   * @param {string} data.url - 要分析的URL
   * @param {string} data.topic - PPT主题（可选）
   * @param {number} data.slides_count - 幻灯片数量
   * @param {string} data.template - 模板类型
   * @param {Object} data.options - 其他选项
   * @returns {Promise}
   */
  generatePPT: async (data) => {
    try {
      console.log('Coze生成PPT请求:', data)
      const response = await post('/coze/generate-ppt', data, { 
        needAuth: true,
        timeout: 180000 // 3分钟超时
      })
      return response
    } catch (error) {
      console.error('Coze生成PPT失败:', error)
      throw error
    }
  },

  /**
   * 查询PPT生成任务状态
   * @param {string} taskId - 任务ID
   * @returns {Promise}
   */
  getTaskStatus: async (taskId) => {
    try {
      const response = await get(`/coze/tasks/${taskId}/status`, { needAuth: true })
      return response
    } catch (error) {
      console.error('查询任务状态失败:', error)
      throw error
    }
  },

  /**
   * 获取任务完成结果
   * @param {string} taskId - 任务ID
   * @returns {Promise}
   */
  getTaskResult: async (taskId) => {
    try {
      const response = await get(`/coze/tasks/${taskId}/result`, { needAuth: true })
      return response
    } catch (error) {
      console.error('获取任务结果失败:', error)
      throw error
    }
  },

  /**
   * 获取任务HTML预览信息
   * @param {string} taskId - 任务ID
   * @returns {Promise}
   */
  getTaskHTMLPreview: async (taskId) => {
    try {
      const response = await get(`/coze/tasks/${taskId}/html-preview`, { needAuth: true })
      return response
    } catch (error) {
      console.error('获取HTML预览信息失败:', error)
      throw error
    }
  },

  /**
   * 处理Coze返回的PPT链接
   * @param {Object} data - 链接处理参数
   * @param {string} data.coze_url - Coze返回的PPT链接
   * @param {string} data.title - PPT标题（可选）
   * @returns {Promise}
   */
  processCozeLink: async (data) => {
    try {
      console.log('处理Coze链接请求:', data)
      const response = await post('/coze/process-link', data, { 
        needAuth: true,
        timeout: 300000 // 5分钟超时，因为需要下载和转换
      })
      return response
    } catch (error) {
      console.error('处理Coze链接失败:', error)
      throw error
    }
  },

  /**
   * 清理/删除任务
   * @param {string} taskId - 任务ID
   * @returns {Promise}
   */
  cleanupTask: async (taskId) => {
    try {
      const response = await del(`/coze/tasks/${taskId}`, { needAuth: true })
      return response
    } catch (error) {
      console.error('清理任务失败:', error)
      throw error
    }
  },

  /**
   * 获取所有任务列表（调试用）
   * @returns {Promise}
   */
  getAllTasks: async () => {
    try {
      const response = await get('/coze/tasks', { needAuth: true })
      return response
    } catch (error) {
      console.error('获取任务列表失败:', error)
      throw error
    }
  },

  /**
   * 验证URL格式
   * @param {string} url - 要验证的URL
   * @returns {Promise}
   */
  validateURL: async (url) => {
    try {
      const response = await get(`/coze/validate-url?url=${encodeURIComponent(url)}`, { needAuth: false })
      return response
    } catch (error) {
      console.error('URL验证失败:', error)
      throw error
    }
  },

  /**
   * 获取Coze Bot配置信息
   * @returns {Promise}
   */
  getBotConfig: async () => {
    try {
      const response = await get('/coze/bot-config', { needAuth: false })
      return response
    } catch (error) {
      console.error('获取Bot配置失败:', error)
      throw error
    }
  },

  /**
   * 检查Coze服务健康状态
   * @returns {Promise}
   */
  checkHealth: async () => {
    try {
      const response = await get('/coze/health', { needAuth: false })
      return response
    } catch (error) {
      console.error('Coze健康检查失败:', error)
      throw error
    }
  },

  /**
   * 轮询任务状态直到完成（优化版）
   * @param {string} taskId - 任务ID
   * @param {number} initialInterval - 初始轮询间隔（毫秒），默认2秒
   * @param {number} maxAttempts - 最大尝试次数，默认30次
   * @param {Object} options - 轮询选项
   * @returns {Promise}
   */
  pollTaskUntilComplete: async (taskId, initialInterval = 2000, maxAttempts = 60, options = {}) => {
    const {
      maxInterval = 10000,        // 最大轮询间隔
      backoffMultiplier = 1.2,    // 退让倍数
      fastCheckCount = 5,         // 快速检查次数（前几次使用较短间隔）
      networkErrorRetries = 3,    // 网络错误重试次数
      enableAdaptive = true       // 启用自适应间隔
    } = options

    let attempts = 0
    let consecutiveProcessing = 0
    let consecutiveErrors = 0
    let currentInterval = initialInterval
    let lastProgressTime = Date.now()
    let lastProgress = 0
    
    console.log(`🔄 开始智能轮询任务状态: ${taskId}`)
    console.log(`📋 轮询配置: 初始间隔=${initialInterval}ms, 最大尝试=${maxAttempts}, 自适应=${enableAdaptive}`)
    
    while (attempts < maxAttempts) {
      const attemptStartTime = Date.now()
      
      try {
        console.log(`📊 轮询尝试 ${attempts + 1}/${maxAttempts} (间隔: ${currentInterval}ms)`)
        const status = await cozeAPI.getTaskStatus(taskId)
        console.log(`📋 任务状态:`, status)
        
        // 重置连续错误计数
        consecutiveErrors = 0
        
        // 处理完成状态
        if (status.status === 'completed') {
          console.log(`✅ 任务完成，获取结果`)
          try {
            const result = await cozeAPI.getTaskResult(taskId)
            console.log(`🎉 任务成功完成，总计尝试 ${attempts + 1} 次`)
            return {
              success: true,
              status: 'completed',
              result: result,
              metadata: {
                totalAttempts: attempts + 1,
                totalTime: Date.now() - (attemptStartTime - currentInterval * attempts),
                avgInterval: currentInterval
              }
            }
          } catch (resultError) {
            console.warn(`⚠️ 获取任务结果失败，但任务状态为已完成:`, resultError)
            return {
              success: true,
              status: 'completed',
              result: null,
              warning: '任务完成但无法获取详细结果'
            }
          }
        } 
        
        // 处理失败状态
        else if (status.status === 'failed') {
          console.log(`❌ 任务失败:`, status)
          return {
            success: false,
            status: 'failed',
            error: status.error || '任务执行失败'
          }
        } 
        
        // 处理进行中状态
        else if (status.status === 'processing') {
          consecutiveProcessing++
          const currentProgress = status.progress || 0
          
          // 检查进度是否有变化
          let progressChanged = currentProgress > lastProgress
          if (progressChanged) {
            lastProgressTime = Date.now()
            lastProgress = currentProgress
            console.log(`⏳ 任务处理中... 进度: ${currentProgress}% (第${consecutiveProcessing}次检查)`)
          } else {
            const stuckTime = Date.now() - lastProgressTime
            console.log(`⏳ 任务处理中... 进度: ${currentProgress}% (无变化 ${Math.round(stuckTime/1000)}s)`)
            
            // 如果进度长时间无变化，调整策略
            if (stuckTime > 60000) { // 1分钟无进度变化
              console.warn(`⚠️ 任务进度长时间无变化，可能存在问题`)
              return {
                success: false,
                status: 'stalled',
                error: '任务长时间无进度变化，可能卡住了。建议重新生成或联系技术支持。',
                suggestion: 'retry_or_contact_support'
              }
            }
          }
          
          // 自适应调整轮询间隔
          if (enableAdaptive) {
            if (progressChanged) {
              // 进度有变化，减少间隔
              currentInterval = Math.max(initialInterval, currentInterval * 0.8)
            } else if (attempts > fastCheckCount) {
              // 进度无变化且过了快速检查期，增加间隔
              currentInterval = Math.min(maxInterval, currentInterval * backoffMultiplier)
            }
          }
          
          // 连续processing检查
          if (consecutiveProcessing >= 20) {
            console.warn(`⚠️ 任务长时间处于处理状态`)
            return {
              success: false,
              status: 'timeout',
              error: '任务处理时间过长。建议尝试：\n1. 检查网络连接\n2. 简化生成内容\n3. 重新生成\n4. 联系技术支持',
              suggestion: 'retry_or_contact_support'
            }
          }
        }
        
        // 其他未知状态
        else {
          console.warn(`❓ 未知任务状态: ${status.status}`)
        }
        
        // 任务仍在处理中，准备下次轮询
        attempts++
        
        if (attempts >= maxAttempts) {
          console.warn(`⏰ 轮询超时，达到最大尝试次数`)
          break
        }
        
        // 根据是否在快速检查期调整等待时间
        const waitTime = attempts <= fastCheckCount ? 
          Math.min(currentInterval, initialInterval) : currentInterval
        
        console.log(`⏸️ 等待 ${waitTime/1000} 秒后继续轮询...`)
        await new Promise(resolve => setTimeout(resolve, waitTime))
        
      } catch (error) {
        consecutiveErrors++
        console.error(`❌ 轮询任务状态失败 (尝试 ${attempts + 1}, 连续错误 ${consecutiveErrors}):`, error)
        
        // 网络错误重试逻辑
        if (consecutiveErrors <= networkErrorRetries) {
          console.log(`🔄 网络错误重试 (${consecutiveErrors}/${networkErrorRetries})`)
          
          // 网络错误时使用指数退让
          const retryWaitTime = Math.min(1000 * Math.pow(2, consecutiveErrors), 8000)
          console.log(`⏸️ 网络错误等待 ${retryWaitTime/1000} 秒重试...`)
          await new Promise(resolve => setTimeout(resolve, retryWaitTime))
          
          continue // 不增加 attempts，给网络错误更多重试机会
        }
        
        attempts++
        
        // 连续错误过多，直接失败
        if (consecutiveErrors > networkErrorRetries) {
          console.error(`❌ 连续网络错误过多，终止轮询`)
          return {
            success: false,
            status: 'network_error',
            error: '网络连接问题，请检查网络后重试: ' + error.message,
            suggestion: 'check_network_and_retry'
          }
        }
        
        if (attempts >= maxAttempts) {
          return {
            success: false,
            status: 'error',
            error: '任务状态查询失败: ' + error.message
          }
        }
      }
    }
    
    // 超时处理
    console.warn(`⏰ 任务轮询超时`)
    return {
      success: false,
      status: 'timeout',
      error: `任务执行超时，已尝试 ${attempts} 次。建议：\n1. 稍后手动查看结果\n2. 重新生成\n3. 联系技术支持`,
      suggestion: 'check_later_or_retry',
      metadata: {
        totalAttempts: attempts,
        finalInterval: currentInterval
      }
    }
  },

  /**
   * 轮询任务状态的简化版本（向后兼容）
   * @param {string} taskId - 任务ID
   * @param {number} interval - 轮询间隔（毫秒）
   * @param {number} maxAttempts - 最大尝试次数
   * @returns {Promise}
   */
  pollTaskUntilCompleteSimple: async (taskId, interval = 3000, maxAttempts = 20) => {
    return cozeAPI.pollTaskUntilComplete(taskId, interval, maxAttempts, {
      enableAdaptive: false // 使用固定间隔
    })
  },

  /**
   * 任务恢复管理器
   */
  taskRecoveryManager: {
    // 获取存储键
    getStorageKey: (taskId) => `coze_task_${taskId}`,
    
    // 保存任务状态
    saveTaskState: (taskId, state) => {
      try {
        const key = cozeAPI.taskRecoveryManager.getStorageKey(taskId)
        const taskState = {
          ...state,
          lastUpdate: Date.now(),
          retryCount: state.retryCount || 0
        }
        wx.setStorageSync(key, JSON.stringify(taskState))
        console.log(`💾 任务状态已保存: ${taskId}`)
      } catch (error) {
        console.warn('保存任务状态失败:', error)
      }
    },
    
    // 加载任务状态
    loadTaskState: (taskId) => {
      try {
        const key = cozeAPI.taskRecoveryManager.getStorageKey(taskId)
        const stateStr = wx.getStorageSync(key)
        if (stateStr) {
          const state = JSON.parse(stateStr)
          console.log(`📂 已加载任务状态: ${taskId}`)
          return state
        }
      } catch (error) {
        console.warn('加载任务状态失败:', error)
      }
      return null
    },
    
    // 清理任务状态
    clearTaskState: (taskId) => {
      try {
        const key = cozeAPI.taskRecoveryManager.getStorageKey(taskId)
        wx.removeStorageSync(key)
        console.log(`🗑️ 任务状态已清理: ${taskId}`)
      } catch (error) {
        console.warn('清理任务状态失败:', error)
      }
    },
    
    // 获取所有未完成的任务
    getPendingTasks: () => {
      try {
        const storageInfo = wx.getStorageInfoSync()
        const taskKeys = storageInfo.keys.filter(key => key.startsWith('coze_task_'))
        const pendingTasks = []
        
        taskKeys.forEach(key => {
          try {
            const stateStr = wx.getStorageSync(key)
            if (stateStr) {
              const state = JSON.parse(stateStr)
              if (state.status === 'processing' || state.status === 'pending') {
                const taskId = key.replace('coze_task_', '')
                pendingTasks.push({ taskId, ...state })
              }
            }
          } catch (error) {
            console.warn(`解析任务状态失败: ${key}`, error)
          }
        })
        
        return pendingTasks
      } catch (error) {
        console.warn('获取待处理任务失败:', error)
        return []
      }
    },
    
    // 清理过期任务
    cleanupExpiredTasks: () => {
      try {
        const storageInfo = wx.getStorageInfoSync()
        const taskKeys = storageInfo.keys.filter(key => key.startsWith('coze_task_'))
        const expireTime = 24 * 60 * 60 * 1000 // 24小时过期
        const now = Date.now()
        
        taskKeys.forEach(key => {
          try {
            const stateStr = wx.getStorageSync(key)
            if (stateStr) {
              const state = JSON.parse(stateStr)
              if (now - state.lastUpdate > expireTime) {
                wx.removeStorageSync(key)
                console.log(`🗑️ 已清理过期任务: ${key}`)
              }
            }
          } catch (error) {
            console.warn(`清理过期任务失败: ${key}`, error)
          }
        })
      } catch (error) {
        console.warn('清理过期任务失败:', error)
      }
    }
  },

  /**
   * 错误分类和恢复策略
   */
  errorRecoveryStrategies: {
    // 网络错误恢复
    network: {
      shouldRetry: true,
      maxRetries: 5,
      retryDelay: (attempt) => Math.min(1000 * Math.pow(2, attempt), 30000),
      recovery: async (taskId, error, attempt) => {
        console.log(`🔄 网络错误恢复策略 (尝试 ${attempt})`)
        
        // 检查网络状态
        const networkType = await new Promise(resolve => {
          wx.getNetworkType({
            success: (res) => resolve(res.networkType),
            fail: () => resolve('unknown')
          })
        })
        
        if (networkType === 'none') {
          throw new Error('无网络连接，请检查网络设置')
        }
        
        // 指数退让
        const delay = cozeAPI.errorRecoveryStrategies.network.retryDelay(attempt)
        await new Promise(resolve => setTimeout(resolve, delay))
        
        return true // 可以重试
      }
    },
    
    // 服务器错误恢复
    server: {
      shouldRetry: true,
      maxRetries: 3,
      retryDelay: (attempt) => 5000 * attempt,
      recovery: async (taskId, error, attempt) => {
        console.log(`🔄 服务器错误恢复策略 (尝试 ${attempt})`)
        
        // 检查服务器状态
        try {
          await cozeAPI.checkHealth()
          return true
        } catch (healthError) {
          if (attempt >= 3) {
            throw new Error('服务器暂时不可用，请稍后重试')
          }
          return true
        }
      }
    },
    
    // 任务超时恢复
    timeout: {
      shouldRetry: true,
      maxRetries: 2,
      retryDelay: () => 10000,
      recovery: async (taskId, error, attempt) => {
        console.log(`🔄 超时错误恢复策略 (尝试 ${attempt})`)
        
        // 检查任务是否真的超时还是网络问题
        try {
          const status = await cozeAPI.getTaskStatus(taskId)
          if (status.status === 'completed') {
            // 任务实际已完成
            return { recovered: true, status }
          }
        } catch (checkError) {
          console.warn('检查任务状态失败:', checkError)
        }
        
        return true
      }
    },
    
    // 其他错误恢复
    other: {
      shouldRetry: false,
      maxRetries: 1,
      retryDelay: () => 5000,
      recovery: async (taskId, error, attempt) => {
        console.log(`🔄 通用错误恢复策略 (尝试 ${attempt})`)
        return false // 大多数其他错误不重试
      }
    }
  },

  /**
   * 智能重试包装器
   * @param {Function} operation - 要执行的操作
   * @param {string} taskId - 任务ID
   * @param {string} errorType - 错误类型
   * @param {Object} options - 重试选项
   */
  retryWithRecovery: async (operation, taskId, errorType = 'other', options = {}) => {
    const strategy = cozeAPI.errorRecoveryStrategies[errorType] || cozeAPI.errorRecoveryStrategies.other
    const maxRetries = options.maxRetries || strategy.maxRetries
    let attempt = 0
    let lastError
    
    while (attempt <= maxRetries) {
      try {
        const result = await operation()
        
        // 成功时清理重试计数
        if (taskId) {
          const taskState = cozeAPI.taskRecoveryManager.loadTaskState(taskId)
          if (taskState) {
            taskState.retryCount = 0
            cozeAPI.taskRecoveryManager.saveTaskState(taskId, taskState)
          }
        }
        
        return result
      } catch (error) {
        lastError = error
        attempt++
        
        console.error(`操作失败 (尝试 ${attempt}/${maxRetries + 1}):`, error)
        
        if (attempt > maxRetries) {
          break
        }
        
        // 应用恢复策略
        try {
          const recoveryResult = await strategy.recovery(taskId, error, attempt)
          
          // 如果恢复策略返回了结果，直接返回
          if (recoveryResult && typeof recoveryResult === 'object' && recoveryResult.recovered) {
            return recoveryResult
          }
          
          // 如果恢复策略指示不应重试
          if (!recoveryResult) {
            break
          }
          
          // 保存重试状态
          if (taskId) {
            const taskState = cozeAPI.taskRecoveryManager.loadTaskState(taskId) || {}
            taskState.retryCount = attempt
            taskState.lastError = error.message
            cozeAPI.taskRecoveryManager.saveTaskState(taskId, taskState)
          }
          
        } catch (recoveryError) {
          console.error('恢复策略执行失败:', recoveryError)
          throw recoveryError
        }
      }
    }
    
    throw new Error(`操作失败，已重试${maxRetries}次: ${lastError.message}`)
  },

  /**
   * 带恢复机制的任务轮询
   * @param {string} taskId - 任务ID
   * @param {Object} options - 轮询选项
   */
  pollTaskWithRecovery: async (taskId, options = {}) => {
    // 加载之前的任务状态
    let taskState = cozeAPI.taskRecoveryManager.loadTaskState(taskId) || {
      status: 'pending',
      progress: 0,
      retryCount: 0
    }
    
    console.log(`🔄 开始带恢复机制的任务轮询: ${taskId}`)
    console.log(`📋 已保存的任务状态:`, taskState)
    
    try {
      // 先检查任务是否已完成
      if (taskState.status === 'completed') {
        console.log(`✅ 任务已完成，直接返回结果`)
        const result = await cozeAPI.getTaskResult(taskId)
        cozeAPI.taskRecoveryManager.clearTaskState(taskId)
        return {
          success: true,
          status: 'completed',
          result: result,
          recovered: true
        }
      }
      
      // 如果之前的重试次数过多，询问用户是否继续
      if (taskState.retryCount >= 3) {
        console.warn(`⚠️ 任务之前已重试 ${taskState.retryCount} 次`)
        // 这里可以添加用户确认逻辑
      }
      
      // 执行轮询操作
      const pollOperation = () => cozeAPI.pollTaskUntilComplete(taskId, options.initialInterval, options.maxAttempts, options)
      
      const result = await cozeAPI.retryWithRecovery(
        pollOperation,
        taskId,
        'network', // 默认按网络错误处理
        { maxRetries: 2 }
      )
      
      // 成功完成，清理状态
      cozeAPI.taskRecoveryManager.clearTaskState(taskId)
      
      return result
      
    } catch (error) {
      console.error(`任务轮询最终失败: ${taskId}`, error)
      
      // 保存失败状态
      taskState.status = 'failed'
      taskState.error = error.message
      taskState.lastUpdate = Date.now()
      cozeAPI.taskRecoveryManager.saveTaskState(taskId, taskState)
      
      throw error
    }
  },

  /**
   * 恢复未完成的任务
   */
  recoverPendingTasks: async () => {
    console.log('🔄 开始恢复未完成的任务...')
    
    // 清理过期任务
    cozeAPI.taskRecoveryManager.cleanupExpiredTasks()
    
    // 获取待处理任务
    const pendingTasks = cozeAPI.taskRecoveryManager.getPendingTasks()
    
    if (pendingTasks.length === 0) {
      console.log('✅ 没有待恢复的任务')
      return []
    }
    
    console.log(`📋 发现 ${pendingTasks.length} 个待恢复的任务`)
    
    const recoveryResults = []
    
    for (const task of pendingTasks) {
      try {
        console.log(`🔄 尝试恢复任务: ${task.taskId}`)
        
        // 检查任务当前状态
        const currentStatus = await cozeAPI.getTaskStatus(task.taskId)
        
        if (currentStatus.status === 'completed') {
          // 任务已完成
          const result = await cozeAPI.getTaskResult(task.taskId)
          cozeAPI.taskRecoveryManager.clearTaskState(task.taskId)
          
          recoveryResults.push({
            taskId: task.taskId,
            status: 'recovered',
            result: result
          })
          
          console.log(`✅ 任务恢复成功: ${task.taskId}`)
        } else if (currentStatus.status === 'failed') {
          // 任务失败
          cozeAPI.taskRecoveryManager.clearTaskState(task.taskId)
          
          recoveryResults.push({
            taskId: task.taskId,
            status: 'failed',
            error: currentStatus.error
          })
          
          console.log(`❌ 任务确认失败: ${task.taskId}`)
        } else {
          // 任务仍在进行中，继续轮询
          recoveryResults.push({
            taskId: task.taskId,
            status: 'continuing',
            progress: currentStatus.progress
          })
          
          console.log(`⏳ 任务继续进行中: ${task.taskId}`)
        }
        
      } catch (error) {
        console.error(`恢复任务失败: ${task.taskId}`, error)
        
        recoveryResults.push({
          taskId: task.taskId,
          status: 'recovery_failed',
          error: error.message
        })
      }
    }
    
    console.log(`🎯 任务恢复完成，结果:`, recoveryResults)
    return recoveryResults
  },

  /**
   * 一站式Coze PPT生成流程（带恢复机制）
   * 包含：生成PPT -> 轮询状态 -> 获取结果 -> 错误恢复
   * @param {Object} data - 生成参数
   * @returns {Promise}
   */
  createCourseWithCozeRecovery: async (data) => {
    let taskId = null
    
    try {
      console.log('🚀 启动带恢复机制的Coze PPT生成流程:', data)
      
      // 第一步：提交PPT生成任务（带重试）
      const createOperation = () => cozeAPI.generatePPT(data)
      const createResult = await cozeAPI.retryWithRecovery(
        createOperation,
        null,
        'server',
        { maxRetries: 3 }
      )
      
      taskId = createResult.task_id
      
      if (!taskId) {
        throw new Error('任务创建失败，未获取到任务ID')
      }
      
      console.log('✅ 任务创建成功，开始轮询状态:', taskId)
      
      // 保存初始任务状态
      cozeAPI.taskRecoveryManager.saveTaskState(taskId, {
        status: 'processing',
        progress: 0,
        createTime: Date.now(),
        data: data
      })
      
      // 第二步：轮询任务状态直到完成（带恢复机制）
      const pollResult = await cozeAPI.pollTaskWithRecovery(taskId, {
        initialInterval: 2000,
        maxAttempts: 30
      })
      
      if (pollResult.success) {
        console.log('🎉 Coze PPT生成流程完成:', pollResult.result)
        return {
          success: true,
          taskId: taskId,
          result: pollResult.result,
          recovered: pollResult.recovered || false
        }
      } else {
        console.error('❌ Coze PPT生成流程失败:', pollResult.error)
        return {
          success: false,
          taskId: taskId,
          error: pollResult.error
        }
      }
      
    } catch (error) {
      console.error('💥 Coze PPT生成流程异常:', error)
      
      // 保存错误状态
      if (taskId) {
        const taskState = cozeAPI.taskRecoveryManager.loadTaskState(taskId) || {}
        taskState.status = 'failed'
        taskState.error = error.message
        cozeAPI.taskRecoveryManager.saveTaskState(taskId, taskState)
      }
      
      throw error
    }
  },

  /**
   * 一站式Coze PPT生成流程（原版，保持向后兼容）
   * @param {Object} data - 生成参数
   * @returns {Promise}
   */
  createCourseWithCoze: async (data) => {
    try {
      console.log('启动Coze一站式PPT生成流程:', data)
      
      // 第一步：提交PPT生成任务
      const createResult = await cozeAPI.generatePPT(data)
      const taskId = createResult.task_id
      
      if (!taskId) {
        throw new Error('任务创建失败，未获取到任务ID')
      }
      
      console.log('任务创建成功，开始轮询状态:', taskId)
      
      // 第二步：轮询任务状态直到完成
      const pollResult = await cozeAPI.pollTaskUntilComplete(taskId)
      
      if (pollResult.success) {
        console.log('Coze PPT生成流程完成:', pollResult.result)
        return {
          success: true,
          taskId: taskId,
          result: pollResult.result
        }
      } else {
        console.error('Coze PPT生成流程失败:', pollResult.error)
        return {
          success: false,
          taskId: taskId,
          error: pollResult.error
        }
      }
      
    } catch (error) {
      console.error('Coze PPT生成流程异常:', error)
      throw error
    }
  },

  /**
   * 🆕 使用Coze工作流生成PPT
   * @param {Object} data - 生成参数
   * @returns {Promise}
   */
  generatePPTWithWorkflow: async (data) => {
    try {
      console.log('🔥 启动Coze工作流PPT生成:', data)
      
      const response = await post('/coze/workflow/generate-ppt', data, { needAuth: true })
      
      if (response && response.task_id) {
        console.log('✅ 工作流任务创建成功:', response.task_id)
        return {
          success: true,
          task_id: response.task_id,
          message: response.message || '工作流任务创建成功'
        }
      } else {
        throw new Error('工作流任务创建失败')
      }
    } catch (error) {
      console.error('❌ 工作流PPT生成失败:', error)
      throw error
    }
  },

  /**
   * 🆕 使用工作流的一站式PPT生成流程
   * @param {Object} data - 生成参数
   * @returns {Promise}
   */
  createCourseWithWorkflow: async (data) => {
    try {
      console.log('🚀 启动Coze工作流一站式PPT生成流程:', data)
      
      // 第一步：提交工作流PPT生成任务
      const createResult = await cozeAPI.generatePPTWithWorkflow(data)
      const taskId = createResult.task_id
      
      if (!taskId) {
        throw new Error('工作流任务创建失败，未获取到任务ID')
      }
      
      console.log('工作流任务创建成功，开始轮询状态:', taskId)
      
      // 第二步：轮询任务状态直到完成
      const pollResult = await cozeAPI.pollTaskUntilComplete(taskId)
      
      if (pollResult.success) {
        console.log('Coze工作流PPT生成流程完成:', pollResult.result)
        return {
          success: true,
          taskId: taskId,
          result: {
            ...pollResult.result,
            id: taskId  // 🔧 修复：使用任务ID而不是文件路径作为result.id
          }
        }
      } else {
        console.error('Coze工作流PPT生成流程失败:', pollResult.error)
        return {
          success: false,
          taskId: taskId,
          error: pollResult.error
        }
      }
      
    } catch (error) {
      console.error('Coze工作流PPT生成流程异常:', error)
      throw error
    }
  }
}

module.exports = cozeAPI 