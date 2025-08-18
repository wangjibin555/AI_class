package main

import (
	"fmt"
	"log"

	"ai-classroom/internal/services"
	"ai-classroom/pkg/ai"
	"ai-classroom/pkg/parser"
)

func main() {
	fmt.Println("🚀 测试增强PPT生成功能")

	// 1. 测试智能幻灯片数量计算器
	fmt.Println("\n📊 测试智能幻灯片数量计算器")
	testSlideCountCalculator()

	// 2. 测试技术关键词提取增强
	fmt.Println("\n🔍 测试技术关键词提取增强")
	testEnhancedKeywordExtraction()

	// 3. 测试技术PPT生成器
	fmt.Println("\n🎨 测试技术PPT生成器")
	testTechnicalPPTGenerator()

	fmt.Println("\n✅ 所有测试完成！")
}

func testSlideCountCalculator() {
	calculator := services.NewSlideCountCalculator()

	// 创建模拟的结构化内容
	content := &parser.StructuredContent{
		CleanText: generateTestContent(),
		Sections: []parser.Section{
			{Type: "heading", Content: "微信小程序概述"},
			{Type: "content", Content: "小程序是一种新的开发模式"},
			{Type: "heading", Content: "开发环境搭建"},
			{Type: "content", Content: "需要安装微信开发者工具"},
			{Type: "heading", Content: "核心技术栈"},
			{Type: "content", Content: "WXML、WXSS、JavaScript"},
		},
		KeyInfo: &parser.KeyInfo{
			Keywords: []string{"微信小程序", "WXML", "WXSS", "JavaScript", "开发者工具", "API", "组件", "生命周期"},
			Summary:  "微信小程序开发框架介绍",
		},
	}

	// 测试不同用户类型的幻灯片数量计算
	userTypes := []string{"regular", "premium", "vip"}

	for _, userType := range userTypes {
		recommendation, err := calculator.CalculateOptimalSlideCount(content, userType, "url")
		if err != nil {
			log.Printf("❌ %s用户计算失败: %v", userType, err)
			continue
		}

		fmt.Printf("👤 %s用户: 推荐%d张 (最少%d张, 最多%d张)\n",
			userType, recommendation.Recommended, recommendation.Minimum, recommendation.Maximum)
		fmt.Printf("   理由: %s\n", recommendation.Reasoning)
		fmt.Printf("   置信度: %.2f\n", recommendation.Confidence)
		fmt.Printf("   分布: 概念%d张, 教程%d张, 代码%d张, 实践%d张\n\n",
			recommendation.Distribution.ConceptSlides,
			recommendation.Distribution.TutorialSlides,
			recommendation.Distribution.CodeSlides,
			recommendation.Distribution.PracticeSlides)
	}
}

func testEnhancedKeywordExtraction() {
	// 创建关键词提取服务实例
	keywordService := services.NewKeywordExtractionService(nil) // 简化测试，不需要AI客户端

	content := generateTestContent()

	// 测试增强的关键词提取
	options := &services.ExtractionOptions{
		MaxKeywords: 30, // 增加到30个关键词
		ContentType: "technical",
		UseAI:       false, // 关闭AI以便测试基础功能
	}

	result, err := keywordService.ExtractKeywords(content, options)
	if err != nil {
		log.Printf("❌ 关键词提取失败: %v", err)
		return
	}

	fmt.Printf("✅ 关键词提取成功\n")

	if len(result.Primary) > 0 {
		fmt.Printf("主要关键词: %v\n", result.Primary)
	}

	if len(result.Secondary) > 0 {
		fmt.Printf("次要关键词: %v\n", result.Secondary)
	}
}

func testTechnicalPPTGenerator() {
	// 创建AI客户端（这里使用nil进行测试）
	var aiClient *ai.DashScopeClient = nil

	_ = services.NewTechnicalPPTGenerator(aiClient) // 使用下划线避免未使用警告

	// 创建技术生成参数
	params := &services.TechnicalGenerationParams{
		ContentType:         "framework_docs",
		TechnicalLevel:      "intermediate",
		UserType:            "regular",
		Language:            "zh-CN",
		MaxSlideCount:       15,
		ContentDensity:      1.2,
		IncludeCodeExample:  true,
		IncludeBestPractice: true,
		DetailLevel:         "high",
		FocusAreas:          []string{"concept", "practice", "code"},
		GenerationStyle:     "comprehensive",
	}

	// 由于没有真实的AI客户端，这里只测试结构和参数传递
	fmt.Printf("✅ 技术PPT生成器初始化成功\n")
	fmt.Printf("📋 生成参数: 内容类型=%s, 技术水平=%s, 最大幻灯片=%d\n",
		params.ContentType, params.TechnicalLevel, params.MaxSlideCount)
	fmt.Printf("🎯 关注领域: %v\n", params.FocusAreas)
	fmt.Printf("📊 内容密度: %.1f\n", params.ContentDensity)

	// 注意：实际的GenerateTechnicalSlides需要真实的AI客户端
	// 这里只是验证组件能够正确初始化
	fmt.Printf("⚠️ 注意: 完整测试需要配置AI客户端\n")
}

func generateTestContent() string {
	return `
微信小程序开发框架概述

微信小程序是一种全新的连接用户与服务的方式，它可以在微信内被便捷地获取和传播，同时具有出色的使用体验。

## 核心技术栈

### 1. WXML（WeiXin Markup Language）
WXML是框架设计的一套标签语言，用来构建小程序页面的结构，其语法类似于HTML。

主要特性：
- 数据绑定
- 列表渲染  
- 条件渲染
- 模板
- 事件

示例代码：
<view class="container">
  <text>{{message}}</text>
  <button bindtap="clickMe">点击我</button>
</view>

### 2. WXSS（WeiXin Style Sheets）
WXSS是一套样式语言，用于描述WXML的组件样式，决定WXML的组件应该怎么显示。

核心特性：
- 尺寸单位rpx
- 样式导入@import
- 内联样式
- 选择器

### 3. JavaScript逻辑层
JavaScript负责小程序的逻辑处理，包括数据处理、API调用、用户交互等。

重要概念：
- App()函数 - 注册小程序
- Page()函数 - 注册页面
- 生命周期回调
- 事件处理函数
- 数据绑定

## 开发环境搭建

### 1. 下载开发者工具
访问微信公众平台下载专用的开发者工具。

### 2. 创建项目
使用开发者工具创建新的小程序项目。

### 3. 项目结构
- app.js - 小程序逻辑
- app.json - 小程序公共配置  
- app.wxss - 小程序公共样式表
- pages/ - 页面文件夹

## API接口

小程序提供了丰富的API接口：
- 界面API：如wx.showToast()、wx.showModal()
- 网络API：如wx.request()
- 媒体API：如wx.chooseImage()、wx.previewImage()
- 位置API：如wx.getLocation()
- 设备API：如wx.getSystemInfo()
- 存储API：如wx.setStorageSync()

## 最佳实践

1. 合理使用生命周期
2. 优化数据绑定性能
3. 减少setData调用频率
4. 图片资源优化
5. 代码包大小控制

## 常见问题

1. 如何处理异步请求？
2. 如何实现页面间通信？
3. 如何优化小程序性能？
4. 如何调试小程序？

通过学习这些核心概念和技术要点，开发者可以快速掌握微信小程序开发的精髓。
`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
