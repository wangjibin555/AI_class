package parser

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// EnhancedFileContent 增强文件内容结构体
type EnhancedFileContent struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Source   string `json:"source"`
	Type     string `json:"type"`
	FileSize int64  `json:"file_size"`
	Pages    int    `json:"pages,omitempty"`
}

// EnhancedFileProcessor 增强文件处理器
type EnhancedFileProcessor struct{}

// NewEnhancedFileProcessor 创建新的增强文件处理器
func NewEnhancedFileProcessor() *EnhancedFileProcessor {
	return &EnhancedFileProcessor{}
}

// ProcessFile 处理文件
func (fp *EnhancedFileProcessor) ProcessFile(filePath string) (*EnhancedFileContent, error) {
	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))

	// 获取文件名作为标题
	title := strings.TrimSuffix(filepath.Base(filePath), ext)

	// 根据文件类型调用相应的处理方法
	var content string
	var pages int

	switch ext {
	case ".txt":
		content, err = fp.processTXTFile(filePath)
	case ".pdf":
		content, pages, err = fp.processPDFFile(filePath)
	case ".docx":
		content, err = fp.processDOCXFile(filePath)
	default:
		return nil, fmt.Errorf("不支持的文件类型: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("处理%s文件失败: %v", ext, err)
	}

	return &EnhancedFileContent{
		Title:    title,
		Content:  content,
		Source:   filePath,
		Type:     ext[1:], // 去掉点号
		FileSize: fileInfo.Size(),
		Pages:    pages,
	}, nil
}

// processTXTFile 处理TXT文件
func (fp *EnhancedFileProcessor) processTXTFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开TXT文件: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("无法读取TXT文件内容: %v", err)
	}

	return string(content), nil
}

// processPDFFile 处理PDF文件
func (fp *EnhancedFileProcessor) processPDFFile(filePath string) (string, int, error) {
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("无法打开PDF文件: %v", err)
	}
	defer file.Close()

	var content strings.Builder
	totalPages := reader.NumPage()

	for i := 1; i <= totalPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// 获取页面字体信息
		fonts := make(map[string]*pdf.Font)
		text, err := page.GetPlainText(fonts)
		if err != nil {
			continue // 跳过无法解析的页面
		}

		if strings.TrimSpace(text) != "" {
			content.WriteString(text)
			content.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(content.String()), totalPages, nil
}

// processDOCXFile 处理DOCX文件
func (fp *EnhancedFileProcessor) processDOCXFile(filePath string) (string, error) {
	// 创建临时Python脚本
	pythonScript := `
import sys
import json
from docx import Document

def extract_docx_content(file_path):
    try:
        doc = Document(file_path)
        content = []
        
        for paragraph in doc.paragraphs:
            text = paragraph.text.strip()
            if text:
                content.append(text)
        
        return '\n\n'.join(content)
    except Exception as e:
        return f"Error: {str(e)}"

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python script.py <docx_file_path>")
        sys.exit(1)
    
    file_path = sys.argv[1]
    content = extract_docx_content(file_path)
    print(content)
`

	// 创建临时文件
	tempDir := os.TempDir()
	scriptPath := filepath.Join(tempDir, "docx_processor.py")

	err := os.WriteFile(scriptPath, []byte(pythonScript), 0755) // 添加执行权限
	if err != nil {
		return "", fmt.Errorf("创建Python脚本失败: %v", err)
	}
	defer os.Remove(scriptPath)

	// 执行Python脚本，添加详细的错误处理
	// 首先尝试使用python3的完整路径
	cmd := exec.Command("/usr/bin/python3", scriptPath, filePath)
	output, err := cmd.CombinedOutput() // 使用CombinedOutput获取stdout和stderr
	if err != nil {
		// 如果完整路径失败，尝试使用PATH中的python3
		cmd = exec.Command("python3", scriptPath, filePath)
		output, err = cmd.CombinedOutput()
		if err != nil {
			// 添加更详细的错误信息
			return "", fmt.Errorf("执行Python脚本失败: %v, 输出: %s", err, string(output))
		}
	}

	content := string(output)
	if strings.HasPrefix(content, "Error:") {
		return "", fmt.Errorf("Python脚本执行错误: %s", content)
	}

	return strings.TrimSpace(content), nil
}

// GetSupportedFileTypes 获取支持的文件类型
func (fp *EnhancedFileProcessor) GetSupportedFileTypes() []string {
	return []string{".txt", ".pdf", ".docx"}
}

// ValidateFileType 验证文件类型
func (fp *EnhancedFileProcessor) ValidateFileType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedTypes := fp.GetSupportedFileTypes()

	for _, supportedType := range supportedTypes {
		if ext == supportedType {
			return true
		}
	}

	return false
}

// GetFileSize 获取文件大小限制（字节）
func (fp *EnhancedFileProcessor) GetFileSize() int64 {
	return 10 * 1024 * 1024 // 10MB
}

// ValidateFileSize 验证文件大小
func (fp *EnhancedFileProcessor) ValidateFileSize(size int64) bool {
	return size <= fp.GetFileSize()
}
