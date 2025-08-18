package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// FileContent 文件内容结构体
type FileContent struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Source  string `json:"source"`
	Type    string `json:"type"`
}

// FileProcessor 文件处理器
type FileProcessor struct{}

// NewFileProcessor 创建新的文件处理器
func NewFileProcessor() *FileProcessor {
	return &FileProcessor{}
}

// ProcessFile 处理文件
func (fp *FileProcessor) ProcessFile(filePath string) (*FileContent, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))

	// 根据文件类型调用相应的处理方法
	switch ext {
	case ".txt":
		return fp.processTXTFile(filePath)
	case ".pdf":
		return fp.processPDFFile(filePath)
	case ".docx":
		return fp.processDOCXFile(filePath)
	default:
		return nil, fmt.Errorf("不支持的文件类型: %s", ext)
	}
}

// processTXTFile 处理TXT文件
func (fp *FileProcessor) processTXTFile(filePath string) (*FileContent, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开TXT文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取TXT文件失败: %v", err)
	}

	// 获取文件名作为标题
	title := filepath.Base(filePath)
	title = strings.TrimSuffix(title, filepath.Ext(title))

	return &FileContent{
		Title:   title,
		Content: string(content),
		Source:  filePath,
		Type:    "txt",
	}, nil
}

// processPDFFile 处理PDF文件
func (fp *FileProcessor) processPDFFile(filePath string) (*FileContent, error) {
	// 打开PDF文件
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开PDF文件失败: %v", err)
	}
	defer file.Close()

	// 获取页数
	totalPage := reader.NumPage()

	var content strings.Builder

	// 逐页读取内容
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		page := reader.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}

		// 提取文本
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // 跳过无法读取的页面
		}

		content.WriteString(text)
		content.WriteString("\n")
	}

	// 获取文件名作为标题
	title := filepath.Base(filePath)
	title = strings.TrimSuffix(title, filepath.Ext(title))

	return &FileContent{
		Title:   title,
		Content: content.String(),
		Source:  filePath,
		Type:    "pdf",
	}, nil
}

// processDOCXFile 处理DOCX文件
func (fp *FileProcessor) processDOCXFile(filePath string) (*FileContent, error) {
	// 检查Python脚本是否存在
	scriptPath := "docx_processor.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("DOCX处理脚本不存在: %s", scriptPath)
	}

	// 执行Python脚本
	cmd := exec.Command("python3", scriptPath, filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("执行DOCX处理脚本失败: %v, 输出: %s", err, string(output))
	}

	// 解析JSON输出
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("解析DOCX处理结果失败: %v", err)
	}

	// 检查是否有错误
	if errorMsg, exists := result["error"]; exists {
		return nil, fmt.Errorf("DOCX处理错误: %v", errorMsg)
	}

	// 提取结果
	title, _ := result["title"].(string)
	content, _ := result["content"].(string)
	source, _ := result["source"].(string)
	docxType, _ := result["type"].(string)

	if content == "" {
		return nil, fmt.Errorf("DOCX文件内容为空")
	}

	return &FileContent{
		Title:   title,
		Content: content,
		Source:  source,
		Type:    docxType,
	}, nil
}

// CleanTextContent 清理文本内容
func (fp *FileProcessor) CleanTextContent(fileContent *FileContent) string {
	content := fileContent.Content

	// 移除多余的空白字符
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// 移除多余的空行
	lines := strings.Split(content, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	content = strings.Join(cleanedLines, "\n")

	// 限制内容长度（通义千问有token限制）
	if len(content) > 8000 {
		content = content[:8000] + "..."
	}

	return content
}
