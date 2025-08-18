/**
 * 字体自适应工具类
 * 根据内容长度动态调整字体大小和布局
 */

// 字体大小计算算法
function calculateFontSize(content, slideSize) {
    const charCount = content.length
    const baseSize = 24 // 基础字体大小（px）
    
    // 字符数量分级
    const levels = [
        { max: 100, fontSize: 24, lineHeight: 1.5 },    // 简短内容
        { max: 200, fontSize: 20, lineHeight: 1.4 },    // 中等内容  
        { max: 350, fontSize: 18, lineHeight: 1.3 },    // 较多内容
        { max: 500, fontSize: 16, lineHeight: 1.2 },    // 大量内容
        { max: 800, fontSize: 14, lineHeight: 1.1 },    // 超多内容
        { max: Infinity, fontSize: 12, lineHeight: 1.0 } // 极多内容（需分页）
    ]
    
    for (let level of levels) {
        if (charCount <= level.max) {
            return {
                fontSize: level.fontSize,
                lineHeight: level.lineHeight,
                needSplit: charCount > 500 // 超过500字需要考虑分页
            }
        }
    }
}

// 布局自适应算法
function adjustLayout(content, fontConfig) {
    const config = {
        // 容器配置
        containerWidth: 800,  // 幻灯片宽度
        containerHeight: 600, // 幻灯片高度
        padding: 40,         // 内边距
        
        // 字体配置
        fontSize: fontConfig.fontSize,
        lineHeight: fontConfig.lineHeight,
        
        // 标题配置
        titleFontSize: Math.max(fontConfig.fontSize * 1.5, 20),
        titleMarginBottom: 20,
        
        // 列表配置
        bulletIndent: 20,
        itemSpacing: 8,
        
        // 分栏配置
        useColumns: content.length > 400,
        columnGap: 30
    }
    
    return config
}

// 动态应用样式
class SlideStyler {
    constructor() {
        this.thresholds = {
            short: 100,
            medium: 200,
            long: 400,
            extraLong: 600
        }
    }
    
    applyStyles(content) {
        const charCount = content.length
        const className = this.getContentClass(charCount)
        
        // 处理特殊元素
        const processedContent = this.processSpecialElements(content)
        
        // 返回样式配置，供页面使用
        return {
            className: className,
            charCount: charCount,
            fontSize: this.calculateFontSize(charCount),
            lineHeight: this.calculateLineHeight(charCount),
            spacing: this.calculateSpacing(charCount),
            cssVariables: this.setCSSVariables(charCount),
            processedContent: processedContent
        }
    }
    
    calculateFontSize(charCount) {
        if (charCount <= this.thresholds.short) return 48
        if (charCount <= this.thresholds.medium) return 40
        if (charCount <= this.thresholds.long) return 32
        return 28
    }
    
    calculateLineHeight(charCount) {
        if (charCount <= this.thresholds.short) return 1.6
        if (charCount <= this.thresholds.medium) return 1.5
        if (charCount <= this.thresholds.long) return 1.4
        return 1.3
    }
    
    calculateSpacing(charCount) {
        return Math.max(8 - Math.floor(charCount / 100), 4)
    }
    
    getContentClass(charCount) {
        if (charCount <= this.thresholds.short) return 'content-short'
        if (charCount <= this.thresholds.medium) return 'content-medium'
        if (charCount <= this.thresholds.long) return 'content-long'
        return 'content-extra-long'
    }
    
    setCSSVariables(charCount) {
        const config = calculateFontSize('x'.repeat(charCount), null)
        
        // 返回CSS变量配置，供页面使用
        return {
            '--content-font-size': `${config.fontSize}px`,
            '--content-line-height': config.lineHeight,
            '--title-font-size': `${Math.max(config.fontSize * 1.4, 20)}px`,
            '--item-spacing': `${Math.max(8 - Math.floor(charCount / 100), 4)}px`
        }
    }
    
    processSpecialElements(content) {
        let processedContent = content
        
        // 处理重要性标记
        const importancePattern = /重要性：\s*(\d+)\/10/g
        processedContent = processedContent.replace(importancePattern, (match, score) => {
            const color = this.getImportanceColor(parseInt(score))
            return `<span class="importance-badge" style="background: ${color}">重要性: ${score}/10</span>`
        })
        
        // 处理分类标记
        const categoryPattern = /分类：\s*([^|]+)/g
        processedContent = processedContent.replace(categoryPattern, (match, category) => {
            return `<span class="category-tag">${category.trim()}</span>`
        })
        
        // 处理列表
        processedContent = this.convertToList(processedContent)
        
        return processedContent
    }
    
    getImportanceColor(score) {
        if (score >= 8) return 'linear-gradient(135deg, #ff4757, #ff6b6b)'
        if (score >= 6) return 'linear-gradient(135deg, #ffa502, #ffb142)'
        return 'linear-gradient(135deg, #70a1ff, #5352ed)'
    }
    
    convertToList(content) {
        // 将项目符号文本转换为真正的列表
        const lines = content.split('\n')
        const listItems = []
        let regularContent = []
        
        lines.forEach(line => {
            if (line.trim().match(/^[•·-]/)) {
                if (regularContent.length > 0) {
                    listItems.push(`<p>${regularContent.join('<br>')}</p>`)
                    regularContent = []
                }
                listItems.push(`<li>${line.replace(/^[•·-]\s*/, '')}</li>`)
            } else if (line.trim()) {
                regularContent.push(line)
            }
        })
        
        if (regularContent.length > 0) {
            listItems.push(`<p>${regularContent.join('<br>')}</p>`)
        }
        
        // 构建最终HTML
        let html = ''
        let inList = false
        
        listItems.forEach(item => {
            if (item.startsWith('<li>')) {
                if (!inList) {
                    html += '<ul class="content-list">'
                    inList = true
                }
                html += item
            } else {
                if (inList) {
                    html += '</ul>'
                    inList = false
                }
                html += item
            }
        })
        
        if (inList) {
            html += '</ul>'
        }
        
        return html
    }
}

// 内容长度分析器
class ContentAnalyzer {
    constructor() {
        this.styler = new SlideStyler()
    }
    
    analyzeContent(content) {
        const charCount = content.length
        const wordCount = content.split(/\s+/).length
        const fontConfig = calculateFontSize(content, null)
        
        return {
            charCount,
            wordCount,
            fontSize: fontConfig.fontSize,
            lineHeight: fontConfig.lineHeight,
            contentClass: this.styler.getContentClass(charCount),
            needSplit: fontConfig.needSplit,
            readability: this.getReadabilityScore(charCount),
            suggestions: this.getSuggestions(charCount)
        }
    }
    
    getReadabilityScore(charCount) {
        if (charCount <= 100) return 'excellent'
        if (charCount <= 200) return 'good'
        if (charCount <= 350) return 'fair'
        if (charCount <= 500) return 'poor'
        return 'very_poor'
    }
    
    getSuggestions(charCount) {
        const suggestions = []
        
        if (charCount <= 100) {
            suggestions.push('内容长度适中，可读性优秀')
        } else if (charCount <= 200) {
            suggestions.push('内容适中，建议保持当前长度')
        } else if (charCount <= 350) {
            suggestions.push('内容较多，建议精简部分要点')
        } else if (charCount <= 500) {
            suggestions.push('内容过多，强烈建议精简或分页')
        } else {
            suggestions.push('内容严重超标，必须分页或大幅精简')
            suggestions.push('考虑拆分为多个幻灯片')
            suggestions.push('移除非关键信息')
        }
        
        return suggestions
    }
}

// 导出工具类
module.exports = {
    calculateFontSize,
    adjustLayout,
    SlideStyler,
    ContentAnalyzer
} 