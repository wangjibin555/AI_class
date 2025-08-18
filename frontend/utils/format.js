/**
 * 格式化工具函数
 */

/**
 * 格式化相对时间
 * @param {Date|string|number} date 日期
 * @returns {string} 格式化后的时间字符串
 */
function formatRelativeTime(date) {
  try {
    const now = new Date()
    const targetDate = new Date(date)
    const diff = now - targetDate
    
    if (diff < 0) {
      return '未来'
    }
    
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    const months = Math.floor(days / 30)
    const years = Math.floor(days / 365)
    
    if (years > 0) {
      return `${years}年前`
    } else if (months > 0) {
      return `${months}个月前`
    } else if (days > 0) {
      return `${days}天前`
    } else if (hours > 0) {
      return `${hours}小时前`
    } else if (minutes > 0) {
      return `${minutes}分钟前`
    } else {
      return '刚刚'
    }
  } catch (e) {
    console.error('格式化时间失败:', e)
    return '未知时间'
  }
}

/**
 * 格式化日期
 * @param {Date|string|number} date 日期
 * @param {string} format 格式 (YYYY-MM-DD, YYYY-MM-DD HH:mm:ss)
 * @returns {string} 格式化后的日期字符串
 */
function formatDate(date, format = 'YYYY-MM-DD') {
  try {
    const d = new Date(date)
    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const hours = String(d.getHours()).padStart(2, '0')
    const minutes = String(d.getMinutes()).padStart(2, '0')
    const seconds = String(d.getSeconds()).padStart(2, '0')
    
    if (format === 'YYYY-MM-DD HH:mm:ss') {
      return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
    } else if (format === 'YYYY-MM-DD HH:mm') {
      return `${year}-${month}-${day} ${hours}:${minutes}`
    } else {
      return `${year}-${month}-${day}`
    }
  } catch (e) {
    console.error('格式化日期失败:', e)
    return '无效日期'
  }
}

/**
 * 格式化文件大小
 * @param {number} bytes 字节数
 * @returns {string} 格式化后的大小字符串
 */
function formatFileSize(bytes) {
  try {
    if (!bytes || bytes === 0) return '0 B'
    
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    const k = 1024
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + units[i]
  } catch (e) {
    console.error('格式化文件大小失败:', e)
    return '未知大小'
  }
}

/**
 * 格式化数字（添加千分位分隔符）
 * @param {number} num 数字
 * @returns {string} 格式化后的数字字符串
 */
function formatNumber(num) {
  try {
    if (num === null || num === undefined) return '0'
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  } catch (e) {
    console.error('格式化数字失败:', e)
    return '0'
  }
}

/**
 * 截断文本
 * @param {string} text 文本
 * @param {number} maxLength 最大长度
 * @param {string} suffix 后缀
 * @returns {string} 截断后的文本
 */
function truncateText(text, maxLength = 50, suffix = '...') {
  try {
    if (!text || text.length <= maxLength) {
      return text || ''
    }
    return text.substring(0, maxLength) + suffix
  } catch (e) {
    console.error('截断文本失败:', e)
    return ''
  }
}

module.exports = {
  formatRelativeTime,
  formatDate,
  formatFileSize,
  formatNumber,
  truncateText
} 