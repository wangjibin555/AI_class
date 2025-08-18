package parser

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

// FileProcessor 文件处理器
type FileProcessor struct {
	config *FileProcessorConfig
}

// FileProcessorConfig 文件处理器配置
type FileProcessorConfig struct {
	MaxContentLength int      `json:"max_content_length"` // 最大内容长度
	SupportedTypes   []string `json:"supported_types"`    // 支持的文件类型
	TempDir          string   `json:"temp_dir"`           // 临时文件目录
}

// FileContent 文件内容结构体
type FileContent struct {
	Title    string `json:"title"`    // 文件标题
	Content  string `json:"content"`  // 文件内容
	Source   string `json:"source"`   // 文件路径
	Type     string `json:"type"`     // 文件类型
	Size     int64  `json:"size"`     // 文件大小
	Pages    int    `json:"pages"`    // 页数（适用于PDF）
	Encoding string `json:"encoding"` // 编码格式
}

// ProcessResult 处理结果
type ProcessResult struct {
	Success  bool         `json:"success"`
	Content  *FileContent `json:"content"`
	Error    string       `json:"error"`
	Duration string       `json:"duration"`
}

// NewFileProcessor 创建新的文件处理器
func NewFileProcessor() *FileProcessor {
	config := &FileProcessorConfig{
		MaxContentLength: 10000, // 10KB文本内容限制
		SupportedTypes:   []string{".txt", ".pdf", ".docx"},
		TempDir:          "temp",
	}

	// 确保临时目录存在
	os.MkdirAll(config.TempDir, 0755)

	return &FileProcessor{
		config: config,
	}
}

// ProcessFile 处理文件
func (fp *FileProcessor) ProcessFile(filePath string) (*ProcessResult, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &ProcessResult{
			Success: false,
			Error:   fmt.Sprintf("文件不存在: %s", filePath),
		}, err
	}

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return &ProcessResult{
			Success: false,
			Error:   fmt.Sprintf("获取文件信息失败: %v", err),
		}, err
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))

	// 检查是否支持该文件类型
	if !fp.isSupportedType(ext) {
		return &ProcessResult{
			Success: false,
			Error:   fmt.Sprintf("不支持的文件类型: %s", ext),
		}, fmt.Errorf("unsupported file type: %s", ext)
	}

	// 根据文件类型调用相应的处理方法
	var content *FileContent
	switch ext {
	case ".txt":
		content, err = fp.processTXTFile(filePath, fileInfo)
	case ".pdf":
		content, err = fp.processPDFFile(filePath, fileInfo)
	case ".docx":
		content, err = fp.processDOCXFile(filePath, fileInfo)
	default:
		return &ProcessResult{
			Success: false,
			Error:   fmt.Sprintf("不支持的文件类型: %s", ext),
		}, fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		return &ProcessResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 清理和限制内容长度
	content.Content = fp.cleanAndLimitContent(content.Content)

	return &ProcessResult{
		Success: true,
		Content: content,
	}, nil
}

// processTXTFile 处理TXT文件
func (fp *FileProcessor) processTXTFile(filePath string, fileInfo os.FileInfo) (*FileContent, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开TXT文件失败: %v", err)
	}
	defer file.Close()

	// 检测编码并读取内容
	content, encoding, err := fp.readTextFileWithEncoding(file)
	if err != nil {
		return nil, fmt.Errorf("读取TXT文件失败: %v", err)
	}

	// 获取文件名作为标题
	title := filepath.Base(filePath)
	title = strings.TrimSuffix(title, filepath.Ext(title))

	return &FileContent{
		Title:    title,
		Content:  content,
		Source:   filePath,
		Type:     "txt",
		Size:     fileInfo.Size(),
		Encoding: encoding,
	}, nil
}

// processPDFFile 处理PDF文件
func (fp *FileProcessor) processPDFFile(filePath string, fileInfo os.FileInfo) (*FileContent, error) {
	// 打开PDF文件
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开PDF文件失败: %v", err)
	}
	defer file.Close()

	// 获取页数
	totalPage := reader.NumPage()
	var contentBuilder strings.Builder

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

		if text != "" {
			contentBuilder.WriteString(text)
			contentBuilder.WriteString("\n")
		}
	}

	content := contentBuilder.String()
	if content == "" {
		return nil, fmt.Errorf("PDF文件中没有可提取的文本内容")
	}

	// 获取文件名作为标题
	title := filepath.Base(filePath)
	title = strings.TrimSuffix(title, filepath.Ext(title))

	return &FileContent{
		Title:   title,
		Content: content,
		Source:  filePath,
		Type:    "pdf",
		Size:    fileInfo.Size(),
		Pages:   totalPage,
	}, nil
}

// processDOCXFile 处理DOCX文件
func (fp *FileProcessor) processDOCXFile(filePath string, fileInfo os.FileInfo) (*FileContent, error) {
	// 创建Python脚本来处理DOCX文件
	scriptContent := `
import sys
import json
from docx import Document
import os

def process_docx(file_path):
    try:
        doc = Document(file_path)
        content = []
        
        for paragraph in doc.paragraphs:
            if paragraph.text.strip():
                content.append(paragraph.text.strip())
        
        # 处理表格
        for table in doc.tables:
            for row in table.rows:
                row_text = []
                for cell in row.cells:
                    if cell.text.strip():
                        row_text.append(cell.text.strip())
                if row_text:
                    content.append(" | ".join(row_text))
        
        title = os.path.splitext(os.path.basename(file_path))[0]
        
        result = {
            "title": title,
            "content": "\n".join(content),
            "source": file_path,
            "type": "docx"
        }
        
        print(json.dumps(result, ensure_ascii=False))
        
    except Exception as e:
        error_result = {
            "error": str(e)
        }
        print(json.dumps(error_result, ensure_ascii=False))

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(json.dumps({"error": "请提供文件路径"}, ensure_ascii=False))
        sys.exit(1)
    
    process_docx(sys.argv[1])
`

	// 创建临时Python脚本
	scriptPath := filepath.Join(fp.config.TempDir, "docx_processor.py")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		return nil, fmt.Errorf("创建DOCX处理脚本失败: %v", err)
	}
	defer os.Remove(scriptPath) // 清理临时文件

	// 执行Python脚本
	// 首先尝试使用python3的完整路径
	cmd := exec.Command("/usr/bin/python3", scriptPath, filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果完整路径失败，尝试使用PATH中的python3
		cmd = exec.Command("python3", scriptPath, filePath)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("执行DOCX处理脚本失败: %v, 输出: %s", err, string(output))
		}
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
		Size:    fileInfo.Size(),
	}, nil
}

// readTextFileWithEncoding 读取文本文件并检测编码
func (fp *FileProcessor) readTextFileWithEncoding(file *os.File) (string, string, error) {
	// 重置文件指针
	file.Seek(0, 0)

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		return "", "", err
	}

	// 简单的编码检测（可以后续扩展更复杂的检测）
	encoding := "UTF-8"

	// 尝试作为UTF-8处理
	text := string(content)

	return text, encoding, nil
}

// cleanAndLimitContent 清理和限制内容长度
func (fp *FileProcessor) cleanAndLimitContent(content string) string {
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

	// 限制内容长度
	if len(content) > fp.config.MaxContentLength {
		content = content[:fp.config.MaxContentLength] + "..."
	}

	return content
}

// isSupportedType 检查是否支持该文件类型
func (fp *FileProcessor) isSupportedType(ext string) bool {
	for _, supportedType := range fp.config.SupportedTypes {
		if ext == supportedType {
			return true
		}
	}
	return false
}

// GetSupportedTypes 获取支持的文件类型列表
func (fp *FileProcessor) GetSupportedTypes() []string {
	return fp.config.SupportedTypes
}
