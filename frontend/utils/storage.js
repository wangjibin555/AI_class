/**
 * 本地存储工具
 */
const storage = {
  /**
   * 存储数据
   * @param {string} key 键名
   * @param {any} value 值
   */
  setStorage(key, value) {
    try {
      wx.setStorageSync(key, value)
      console.log(`存储成功: ${key}`)
    } catch (error) {
      console.error(`存储失败: ${key}`, error)
    }
  },

  /**
   * 获取数据
   * @param {string} key 键名
   * @param {any} defaultValue 默认值
   */
  getStorage(key, defaultValue = null) {
    try {
      const value = wx.getStorageSync(key)
      return value !== '' ? value : defaultValue
    } catch (error) {
      console.error(`获取存储失败: ${key}`, error)
      return defaultValue
    }
  },

  /**
   * 删除数据
   * @param {string} key 键名
   */
  removeStorage(key) {
    try {
      wx.removeStorageSync(key)
      console.log(`删除存储成功: ${key}`)
    } catch (error) {
      console.error(`删除存储失败: ${key}`, error)
    }
  },

  /**
   * 清空所有数据
   */
  clearStorage() {
    try {
      wx.clearStorageSync()
      console.log('清空存储成功')
    } catch (error) {
      console.error('清空存储失败', error)
    }
  },

  /**
   * 异步存储数据
   * @param {string} key 键名
   * @param {any} value 值
   */
  setStorageAsync(key, value) {
    return new Promise((resolve, reject) => {
      wx.setStorage({
        key: key,
        data: value,
        success: () => {
          console.log(`异步存储成功: ${key}`)
          resolve()
        },
        fail: (error) => {
          console.error(`异步存储失败: ${key}`, error)
          reject(error)
        }
      })
    })
  },

  /**
   * 异步获取数据
   * @param {string} key 键名
   */
  getStorageAsync(key) {
    return new Promise((resolve, reject) => {
      wx.getStorage({
        key: key,
        success: (res) => {
          console.log(`异步获取存储成功: ${key}`)
          resolve(res.data)
        },
        fail: (error) => {
          console.error(`异步获取存储失败: ${key}`, error)
          reject(error)
        }
      })
    })
  },

  /**
   * 获取存储信息
   */
  getStorageInfo() {
    try {
      const info = wx.getStorageInfoSync()
      console.log('存储信息:', info)
      return info
    } catch (error) {
      console.error('获取存储信息失败', error)
      return null
    }
  }
}

module.exports = storage 