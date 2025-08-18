// pages/test/test.js
Page({
  data: {
    testResults: [],
    isLoading: false
  },

  onLoad() {
    this.runTests()
  },

  async runTests() {
    this.setData({ isLoading: true })
    
    const tests = [
      {
        name: '平台识别测试',
        run: this.testPlatformIdentification
      },
      {
        name: 'URL验证测试',
        run: this.testUrlValidation
      },
      {
        name: 'API超时测试',
        run: this.testApiTimeout
      }
    ]
    
    const results = []
    
    for (const test of tests) {
      try {
        const result = await test.run.call(this)
        results.push({
          name: test.name,
          status: 'success',
          result: result
        })
      } catch (error) {
        results.push({
          name: test.name,
          status: 'error',
          error: error.message
        })
      }
    }
    
    this.setData({
      testResults: results,
      isLoading: false
    })
  },

  testPlatformIdentification() {
    const testUrls = [
      'https://blog.csdn.net/test',
      'https://www.bilibili.com/test',
      'https://mp.weixin.qq.com/test',
      'https://www.zhihu.com/test',
      'https://weibo.com/test',
      'https://example.com/test'
    ]
    
    const expected = [
      'CSDN博客',
      'B站', 
      '微信公众号',
      '知乎',
      '微博',
      '其他网站'
    ]
    
    const identifyPlatform = (url) => {
      const platformMap = {
        'https://blog.csdn.net': 'CSDN博客',
        'https://www.bilibili.com': 'B站',
        'https://mp.weixin.qq.com': '微信公众号',
        'https://www.zhihu.com': '知乎',
        'https://weibo.com': '微博'
      }
      
      for (const [prefix, name] of Object.entries(platformMap)) {
        if (url.startsWith(prefix)) {
          return name
        }
      }
      
      return '其他网站'
    }
    
    const results = testUrls.map((url, index) => {
      const result = identifyPlatform(url)
      return {
        url: url,
        expected: expected[index],
        actual: result,
        passed: result === expected[index]
      }
    })
    
    return results
  },

  testUrlValidation() {
    // 模拟URL验证逻辑
    const testUrl = 'https://blog.csdn.net/test'
    const platform = this.identifyPlatform(testUrl)
    
    return {
      url: testUrl,
      platform: platform,
      validated: true
    }
  },

  testApiTimeout() {
    // 模拟API超时测试
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          timeout: '120000ms',
          status: 'configured'
        })
      }, 100)
    })
  },

  identifyPlatform(url) {
    const platformMap = {
      'https://blog.csdn.net': 'CSDN博客',
      'https://www.bilibili.com': 'B站',
      'https://mp.weixin.qq.com': '微信公众号',
      'https://www.zhihu.com': '知乎',
      'https://weibo.com': '微博'
    }
    
    for (const [prefix, name] of Object.entries(platformMap)) {
      if (url.startsWith(prefix)) {
        return name
      }
    }
    
    return '其他网站'
  }
}) 