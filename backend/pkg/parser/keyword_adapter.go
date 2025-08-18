package parser

import (
	"log"
)

// KeywordServiceAdapter 关键词服务适配器
// 将新的KeywordExtractionService适配到DocumentParser中
type KeywordServiceAdapter struct {
	keywordService KeywordExtractionServiceInterface
}

// KeywordExtractionServiceInterface 关键词提取服务接口
type KeywordExtractionServiceInterface interface {
	ExtractKeywords(content string, options interface{}) ([]string, error)
}

// NewKeywordServiceAdapter 创建关键词服务适配器
func NewKeywordServiceAdapter(service KeywordExtractionServiceInterface) *KeywordServiceAdapter {
	return &KeywordServiceAdapter{
		keywordService: service,
	}
}

// ExtractKeywords 提取关键词（适配现有KeywordExtractor接口）
func (a *KeywordServiceAdapter) ExtractKeywords(text string) []string {
	if a.keywordService == nil {
		log.Printf("⚠️ KeywordExtractionService未设置，使用空关键词")
		return []string{}
	}

	// 调用新的KeywordExtractionService
	keywords, err := a.keywordService.ExtractKeywords(text, map[string]interface{}{
		"max_keywords": 10,
		"content_type": "academic",
		"use_ai":       false, // 文档解析阶段使用基础提取
	})

	if err != nil {
		log.Printf("⚠️ 关键词提取失败: %v", err)
		return []string{}
	}

	return keywords
}

// EnhancedDocumentParser 增强版文档解析器
// 直接集成KeywordExtractionService
type EnhancedDocumentParser struct {
	parser         DocumentParser
	keywordService KeywordExtractionServiceInterface
}

// NewEnhancedDocumentParser 创建增强版文档解析器
func NewEnhancedDocumentParser(keywordService KeywordExtractionServiceInterface) *EnhancedDocumentParser {
	return &EnhancedDocumentParser{
		parser:         NewDocumentParser(),
		keywordService: keywordService,
	}
}

// ParseFileWithEnhancedKeywords 使用增强关键词提取解析文件
func (e *EnhancedDocumentParser) ParseFileWithEnhancedKeywords(file interface{}) (*ParseResult, error) {
	// 首先使用原始解析器解析文件
	var result *ParseResult
	var err error

	// 根据file类型调用相应的解析方法
	// 这里需要根据实际的参数类型来实现

	// 如果解析成功且有内容，使用增强关键词提取
	if err == nil && result != nil && result.Content != "" {
		if e.keywordService != nil {
			enhancedKeywords, keywordErr := e.keywordService.ExtractKeywords(result.Content, map[string]interface{}{
				"max_keywords": 15,
				"content_type": "academic",
				"use_ai":       true, // 使用AI增强提取
			})

			if keywordErr == nil {
				result.Keywords = enhancedKeywords
				log.Printf("✅ 增强关键词提取成功: %v", enhancedKeywords)
			} else {
				log.Printf("⚠️ 增强关键词提取失败，使用原始关键词: %v", keywordErr)
			}
		}
	}

	return result, err
}
