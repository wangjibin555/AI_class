package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// 配置结构体
type Config struct {
	QianwenAPIKey string `json:"qianwen_api_key"`
	OutputDir     string `json:"output_dir"`
}

// 网页内容结构体
type WebContent struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
}

// AI总结结果结构体
type SummaryResult struct {
	Summary string `json:"summary"`
	Title   string `json:"title"`
	Source  string `json:"source"`
	Type    string `json:"type"`
}

// 通义千问API请求结构体
type QianwenRequest struct {
	Model      string            `json:"model"`
	Input      QianwenInput      `json:"input"`
	Parameters QianwenParameters `json:"parameters"`
}

type QianwenInput struct {
	Messages []QianwenMessage `json:"messages"`
}

type QianwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type QianwenParameters struct {
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// 通义千问API响应结构体
type QianwenResponse struct {
	Output QianwenOutput `json:"output"`
}

type QianwenOutput struct {
	Text string `json:"text"`
}

func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("使用方法:")
		fmt.Println("  URL爬取: go run main.go <URL>")
		fmt.Println("  文件处理: go run main.go --file <文件路径>")
		fmt.Println("示例:")
		fmt.Println("  go run main.go https://example.com")
		fmt.Println("  go run main.go --file document.pdf")
		fmt.Println("  go run main.go --file report.docx")
		fmt.Println("  go run main.go --file notes.txt")
		os.Exit(1)
	}

	// 加载配置
	config, err := loadConfig()
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}

	var content string
	var title string
	var source string
	var contentType string

	// 判断是URL还是文件
	if os.Args[1] == "--file" {
		if len(os.Args) < 3 {
			fmt.Println("请指定文件路径")
			fmt.Println("示例: go run main.go --file document.pdf")
			os.Exit(1)
		}

		filePath := os.Args[2]
		fmt.Printf("开始处理文件: %s\n", filePath)

		// 处理文件
		fileProcessor := NewFileProcessor()
		fileContent, err := fileProcessor.ProcessFile(filePath)
		if err != nil {
			log.Fatal("处理文件失败:", err)
		}
		fmt.Println("✅ 文件内容读取完成")

		// 清理文本内容
		content = fileProcessor.CleanTextContent(fileContent)
		if content == "" {
			log.Fatal("无法提取到有效文本内容")
		}
		fmt.Println("✅ 文本内容清理完成")

		title = fileContent.Title
		source = fileContent.Source
		contentType = fileContent.Type
	} else {
		// 处理URL
		url := os.Args[1]
		fmt.Printf("开始处理URL: %s\n", url)

		// 1. 爬取网页内容
		webContent, err := scrapeWebPage(url)
		if err != nil {
			log.Fatal("爬取网页失败:", err)
		}
		fmt.Println("✅ 网页内容爬取完成")

		// 2. 提取文本内容
		content = extractTextContent(webContent)
		if content == "" {
			log.Fatal("无法提取到有效文本内容")
		}
		fmt.Println("✅ 文本内容提取完成")

		title = webContent.Title
		source = webContent.URL
		contentType = "url"
	}

	// 3. 使用通义千问进行总结
	summary, err := generateSummary(content, config.QianwenAPIKey)
	if err != nil {
		log.Fatal("AI总结失败:", err)
	}
	fmt.Println("✅ AI内容总结完成")

	// 设置总结结果的来源信息
	summary.Source = source
	summary.Type = contentType
	if summary.Title == "" {
		summary.Title = title
	}

	// 4. 生成PPT
	pptGenerator := NewPPTGenerator(config.OutputDir)
	err = pptGenerator.GeneratePPT(summary)
	if err != nil {
		log.Fatal("PPT生成失败:", err)
	}
	fmt.Println("✅ PPT生成完成")

	fmt.Printf("处理完成！PPT文件保存在: %s\n", config.OutputDir)
}

// 加载配置文件
func loadConfig() (*Config, error) {
	config := &Config{
		QianwenAPIKey: os.Getenv("QIANWEN_API_KEY"),
		OutputDir:     "output",
	}

	// 如果环境变量中没有API密钥，尝试从配置文件读取
	if config.QianwenAPIKey == "" {
		if _, err := os.Stat("config.json"); err == nil {
			file, err := os.Open("config.json")
			if err != nil {
				return nil, err
			}
			defer file.Close()

			if err := json.NewDecoder(file).Decode(config); err != nil {
				return nil, err
			}
		}
	}

	// 检查API密钥
	if config.QianwenAPIKey == "" {
		return nil, fmt.Errorf("请设置通义千问API密钥，可以通过环境变量QIANWEN_API_KEY或config.json文件设置")
	}

	// 创建输出目录
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, err
	}

	return config, nil
}

// 爬取网页内容
func scrapeWebPage(url string) (*WebContent, error) {
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 发送HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// 提取标题
	title := doc.Find("title").Text()
	if title == "" {
		title = doc.Find("h1").First().Text()
	}
	if title == "" {
		title = "网页内容总结"
	}

	// 提取正文内容
	content := extractMainContent(doc)

	return &WebContent{
		Title:   title,
		Content: content,
		URL:     url,
	}, nil
}

// 提取网页主要内容
func extractMainContent(doc *goquery.Document) string {
	// 移除不需要的元素
	doc.Find("script, style, nav, header, footer, aside, .ad, .advertisement, .sidebar").Remove()

	// 尝试找到主要内容区域
	contentSelectors := []string{
		"main",
		"article",
		".content",
		".main-content",
		".post-content",
		".entry-content",
		"#content",
		"#main",
	}

	var content string
	for _, selector := range contentSelectors {
		if selection := doc.Find(selector); selection.Length() > 0 {
			content = selection.Text()
			if len(content) > 100 {
				break
			}
		}
	}

	// 如果没有找到主要内容，使用body
	if content == "" || len(content) < 100 {
		content = doc.Find("body").Text()
	}

	return content
}

// 提取纯文本内容
func extractTextContent(webContent *WebContent) string {
	// 清理文本内容
	content := webContent.Content

	// 移除多余的空白字符
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")
	content = strings.ReplaceAll(content, "\t", " ")

	// 移除多余的空格
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}

	content = strings.TrimSpace(content)

	// 限制内容长度（通义千问有token限制）
	if len(content) > 8000 {
		content = content[:8000] + "..."
	}

	return content
}

// 使用通义千问生成总结
func generateSummary(content, apiKey string) (*SummaryResult, error) {
	// 构建请求
	request := QianwenRequest{
		Model: "qwen-turbo",
		Input: QianwenInput{
			Messages: []QianwenMessage{
				{
					Role: "system",
					Content: "你是一个专业的内容总结助手。请对给定的内容进行总结，生成一个结构化的总结报告，包括：\n" +
						"1. 主要内容概述\n" +
						"2. 关键要点（3-5个）\n" +
						"3. 重要信息提取\n" +
						"4. 总结结论\n\n" +
						"请用中文回答，格式要清晰易读。",
				},
				{
					Role:    "user",
					Content: content,
				},
			},
		},
		Parameters: QianwenParameters{
			MaxTokens:   2000,
			Temperature: 0.7,
		},
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-DashScope-SSE", "disable")

	// 发送请求
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var response QianwenResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(respBody))
	}

	if response.Output.Text == "" {
		return nil, fmt.Errorf("通义千问返回结果为空，完整响应: %s", string(respBody))
	}

	summary := response.Output.Text

	return &SummaryResult{
		Summary: summary,
		Title:   "内容总结",
	}, nil
}
