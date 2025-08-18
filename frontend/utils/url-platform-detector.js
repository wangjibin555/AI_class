/**
 * URL平台识别工具
 * 自动识别URL对应的平台并返回平台信息
 */

class URLPlatformDetector {
  constructor() {
    // 平台识别规则配置
    this.platformRules = [
      // 微信相关
      {
        name: '微信开发者平台',
        domains: ['developers.weixin.qq.com'],
        icon: '🔧',
        color: '#07C160',
        category: 'development',
        description: '微信小程序、公众号开发文档'
      },
      {
        name: '微信公众平台',
        domains: ['mp.weixin.qq.com'],
        icon: '📱',
        color: '#07C160',
        category: 'platform',
        description: '微信公众号管理平台'
      },
      
      // 技术博客
      {
        name: 'CSDN博客',
        domains: ['blog.csdn.net', 'csdn.net'],
        icon: '💻',
        color: '#FC5531',
        category: 'blog',
        description: '技术博客分享平台'
      },
      {
        name: '掘金',
        domains: ['juejin.cn', 'juejin.im'],
        icon: '⚡',
        color: '#1E80FF',
        category: 'blog',
        description: '优质技术文章社区'
      },
      {
        name: '知乎',
        domains: ['zhihu.com', 'zhuanlan.zhihu.com'],
        icon: '🎯',
        color: '#0084FF',
        category: 'knowledge',
        description: '知识分享问答社区'
      },
      {
        name: '简书',
        domains: ['jianshu.com'],
        icon: '📝',
        color: '#EA6F5A',
        category: 'blog',
        description: '创作分享平台'
      },
      {
        name: '博客园',
        domains: ['cnblogs.com'],
        icon: '🌐',
        color: '#2E8B57',
        category: 'blog',
        description: '技术博客园地'
      },
      
      // 学习平台
      {
        name: 'GitHub',
        domains: ['github.com'],
        icon: '🐙',
        color: '#181717',
        category: 'code',
        description: '代码托管与协作平台'
      },
      {
        name: 'Stack Overflow',
        domains: ['stackoverflow.com'],
        icon: '📚',
        color: '#F48024',
        category: 'qa',
        description: '程序员问答社区'
      },
      {
        name: 'MDN文档',
        domains: ['developer.mozilla.org'],
        icon: '🦎',
        color: '#000000',
        category: 'documentation',
        description: 'Web开发权威文档'
      },
      {
        name: 'Vue.js官网',
        domains: ['vuejs.org', 'cn.vuejs.org'],
        icon: '💚',
        color: '#4FC08D',
        category: 'documentation',
        description: 'Vue.js官方文档'
      },
      {
        name: 'React官网',
        domains: ['reactjs.org', 'react.dev'],
        icon: '⚛️',
        color: '#61DAFB',
        category: 'documentation',
        description: 'React官方文档'
      },
      
      // 视频平台
      {
        name: '哔哩哔哩',
        domains: ['bilibili.com', 'b23.tv'],
        icon: '📺',
        color: '#FB7299',
        category: 'video',
        description: '弹幕视频网站'
      },
      {
        name: 'YouTube',
        domains: ['youtube.com', 'youtu.be'],
        icon: '🎬',
        color: '#FF0000',
        category: 'video',
        description: '全球视频分享平台'
      },
      {
        name: '抖音',
        domains: ['douyin.com'],
        icon: '🎵',
        color: '#FE2C55',
        category: 'video',
        description: '短视频分享平台'
      },

      // 社交与内容
      {
        name: '小红书',
        domains: ['xiaohongshu.com', 'xhslink.com'],
        icon: '📷',
        color: '#FE2C55',
        category: 'social',
        description: '生活方式分享平台'
      },
      
      // 新闻媒体
      {
        name: '腾讯新闻',
        domains: ['new.qq.com', 'news.qq.com'],
        icon: '📰',
        color: '#4A90E2',
        category: 'news',
        description: '腾讯新闻资讯'
      },
      {
        name: '新浪新闻',
        domains: ['sina.com.cn', 'news.sina.com.cn'],
        icon: '📰',
        color: '#E60012',
        category: 'news',
        description: '新浪新闻资讯'
      },
      
      // 电商平台
      {
        name: '淘宝',
        domains: ['taobao.com', 'tmall.com'],
        icon: '🛒',
        color: '#FF6600',
        category: 'ecommerce',
        description: '购物电商平台'
      },
      {
        name: '京东',
        domains: ['jd.com', '3.cn'],
        icon: '🛍️',
        color: '#E1251B',
        category: 'ecommerce',
        description: '京东购物商城'
      },
      
      // 其他常见平台
      {
        name: '百度',
        domains: ['baidu.com'],
        icon: '🔍',
        color: '#2932E1',
        category: 'search',
        description: '百度搜索引擎'
      },
      {
        name: '谷歌',
        domains: ['google.com', 'google.cn'],
        icon: '🌐',
        color: '#4285F4',
        category: 'search',
        description: '谷歌搜索引擎'
      },
      {
        name: '维基百科',
        domains: ['wikipedia.org', 'zh.wikipedia.org'],
        icon: '📖',
        color: '#000000',
        category: 'encyclopedia',
        description: '自由的百科全书'
      }
    ]
  }

  /**
   * 从URL中提取域名
   * @param {string} url - 要分析的URL
   * @returns {string|null} 域名
   */
  extractDomain(url) {
    try {
      // 如果URL不包含协议，自动添加
      if (!url.includes('://')) {
        url = 'https://' + url
      }
      
      const urlObj = new URL(url)
      return urlObj.hostname.toLowerCase()
    } catch (error) {
      // 尝试简单的正则匹配
      const match = url.match(/(?:https?:\/\/)?(?:www\.)?([^\/\?#]+)/i)
      return match ? match[1].toLowerCase() : null
    }
  }

  /**
   * 识别URL平台
   * @param {string} url - 要识别的URL
   * @returns {object|null} 平台信息
   */
  detectPlatform(url) {
    if (!url || typeof url !== 'string') {
      return null
    }

    const domain = this.extractDomain(url.trim())
    if (!domain) {
      return null
    }

    // 查找匹配的平台
    for (const platform of this.platformRules) {
      for (const ruleDomain of platform.domains) {
        if (domain === ruleDomain || domain.endsWith('.' + ruleDomain)) {
          return {
            ...platform,
            matchedDomain: domain,
            confidence: this.calculateConfidence(domain, ruleDomain)
          }
        }
      }
    }

    // 如果没有找到匹配的平台，返回通用信息
    return {
      name: '未知网站',
      domains: [domain],
      icon: '🌐',
      color: '#666666',
      category: 'unknown',
      description: '未识别的网站',
      matchedDomain: domain,
      confidence: 0.5
    }
  }

  /**
   * 计算匹配置信度
   * @param {string} domain - 实际域名
   * @param {string} ruleDomain - 规则域名
   * @returns {number} 置信度 (0-1)
   */
  calculateConfidence(domain, ruleDomain) {
    if (domain === ruleDomain) {
      return 1.0
    }
    if (domain.endsWith('.' + ruleDomain)) {
      return 0.9
    }
    return 0.8
  }

  /**
   * 验证URL格式
   * @param {string} url - 要验证的URL
   * @returns {object} 验证结果
   */
  validateURL(url) {
    const result = {
      isValid: false,
      url: url,
      normalizedURL: '',
      errors: [],
      warnings: []
    }

    if (!url || typeof url !== 'string') {
      result.errors.push('URL不能为空')
      return result
    }

    const trimmedURL = url.trim()
    if (!trimmedURL) {
      result.errors.push('URL不能为空')
      return result
    }

    try {
      // 尝试标准化URL
      let normalizedURL = trimmedURL
      
      // 如果没有协议，自动添加https
      if (!normalizedURL.match(/^https?:\/\//i)) {
        normalizedURL = 'https://' + normalizedURL
      }

      // 验证URL格式
      const urlObj = new URL(normalizedURL)
      
      // 检查协议
      if (!['http:', 'https:'].includes(urlObj.protocol)) {
        result.errors.push('仅支持HTTP和HTTPS协议')
        return result
      }

      // 检查域名
      if (!urlObj.hostname) {
        result.errors.push('无效的域名')
        return result
      }

      // 检查域名格式
      const domainRegex = /^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$/
      if (!domainRegex.test(urlObj.hostname)) {
        result.errors.push('域名格式不正确')
        return result
      }

      // 检查是否为本地地址
      if (['localhost', '127.0.0.1', '0.0.0.0'].includes(urlObj.hostname)) {
        result.warnings.push('检测到本地地址，可能无法正常访问')
      }

      // 检查端口
      if (urlObj.port && (parseInt(urlObj.port) < 1 || parseInt(urlObj.port) > 65535)) {
        result.errors.push('端口号不在有效范围内')
        return result
      }

      result.isValid = true
      result.normalizedURL = normalizedURL
      
    } catch (error) {
      result.errors.push('URL格式不正确: ' + error.message)
    }

    return result
  }

  /**
   * 获取平台分类列表
   * @returns {Array} 分类列表
   */
  getCategories() {
    const categories = new Set()
    this.platformRules.forEach(platform => {
      categories.add(platform.category)
    })
    return Array.from(categories)
  }

  /**
   * 根据分类获取平台列表
   * @param {string} category - 分类名称
   * @returns {Array} 平台列表
   */
  getPlatformsByCategory(category) {
    return this.platformRules.filter(platform => platform.category === category)
  }
}

// 创建全局实例
const urlPlatformDetector = new URLPlatformDetector()

module.exports = urlPlatformDetector