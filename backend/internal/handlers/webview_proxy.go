package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// WebViewProxyHandler web-view代理处理器
type WebViewProxyHandler struct{}

// NewWebViewProxyHandler 创建web-view代理处理器
func NewWebViewProxyHandler() *WebViewProxyHandler {
	return &WebViewProxyHandler{}
}

// ProxyHTMLFile 代理HTML文件，解决小程序域名限制问题
func (h *WebViewProxyHandler) ProxyHTMLFile(c *gin.Context) {
	filePath := c.Param("filepath")
	method := c.Request.Method

	log.Printf("🔍 WebView代理收到%s请求: %s", method, filePath)

	if filePath == "" {
		log.Printf("❌ 文件路径为空")
		c.JSON(400, gin.H{"error": "文件路径不能为空"})
		return
	}

	// 移除开头的斜杠
	if filePath[0] == '/' {
		filePath = filePath[1:]
	}

	// 🔧 尝试多个可能的路径位置
	possiblePaths := []string{
		filepath.Join("output/html", filePath),
		filepath.Join("./output/html", filePath),
	}

	// 添加基于工作目录的路径
	if workingDir, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(workingDir, "output/html", filePath))
	}

	var fullPath string
	var fileInfo os.FileInfo
	var fileExists bool

	// 尝试每个可能的路径
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fullPath = path
			fileInfo = info
			fileExists = true
			log.Printf("✅ WebView代理找到HTML文件: %s (大小: %d bytes)", fullPath, info.Size())
			break
		}
	}

	if !fileExists {
		log.Printf("❌ WebView代理：HTML文件不存在: %s", filePath)
		for i, path := range possiblePaths {
			log.Printf("  %d. %s", i+1, path)
		}
		c.JSON(404, gin.H{"error": "文件不存在"})
		return
	}

	// 读取HTML文件内容
	htmlContent, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("❌ WebView代理：读取HTML文件失败: %v", err)
		c.JSON(500, gin.H{"error": "读取文件失败"})
		return
	}

	// 🔧 修改HTML内容，替换外部CDN链接为本地或可信域名
	modifiedHTML := h.modifyHTMLForWebView(string(htmlContent), c)

	// 设置响应头
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("Content-Length", fmt.Sprintf("%d", len(modifiedHTML)))
	c.Header("Last-Modified", fileInfo.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))

	// 对于HEAD请求，只返回头部信息
	if method == "HEAD" {
		log.Printf("✅ WebView代理成功响应HEAD请求: %s", fullPath)
		c.Status(200)
		return
	}

	// 对于GET请求，返回修改后的HTML内容
	log.Printf("✅ WebView代理成功提供HTML文件: %s (修改后大小: %d bytes)", fullPath, len(modifiedHTML))
	c.String(200, modifiedHTML)
}

// modifyHTMLForWebView 修改HTML内容，使其适合在小程序web-view中显示
func (h *WebViewProxyHandler) modifyHTMLForWebView(htmlContent string, c *gin.Context) string {
	// 获取当前请求的基础URL
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	log.Printf("🔧 WebView代理修改HTML内容，baseURL: %s", baseURL)

	// 替换reveal.js CDN链接为本地版本或稳定的CDN
	replacements := map[string]string{
		// reveal.js CSS
		"https://cdn.jsdelivr.net/npm/reveal.js@4.3.1/dist/reveal.css":      "https://unpkg.com/reveal.js@4.3.1/dist/reveal.css",
		"https://cdn.jsdelivr.net/npm/reveal.js@4.3.1/dist/theme/white.css": "https://unpkg.com/reveal.js@4.3.1/dist/theme/white.css",

		// reveal.js JS
		"https://cdn.jsdelivr.net/npm/reveal.js@4.3.1/dist/reveal.js": "https://unpkg.com/reveal.js@4.3.1/dist/reveal.js",

		// Google Fonts（使用国内可访问的CDN）
		"https://fonts.googleapis.com": "https://fonts.font.im",
		"https://fonts.gstatic.com":    "https://fonts.gstatic.font.im",
	}

	modifiedHTML := htmlContent
	for oldURL, newURL := range replacements {
		if strings.Contains(modifiedHTML, oldURL) {
			modifiedHTML = strings.ReplaceAll(modifiedHTML, oldURL, newURL)
			log.Printf("🔄 替换CDN链接: %s -> %s", oldURL, newURL)
		}
	}

	// 添加小程序web-view兼容性脚本
	webviewScript := `
<script>
// 小程序web-view兼容性增强
(function() {
    console.log('🔧 WebView代理：初始化小程序兼容性脚本');
    
    // 检测是否在微信小程序环境中
    const isInMiniProgram = window.__wxjs_environment === 'miniprogram';
    if (isInMiniProgram) {
        console.log('✅ 检测到微信小程序环境');
        
        // 禁用一些可能导致问题的功能
        if (window.Reveal) {
            // 禁用键盘导航，避免与小程序冲突
            window.Reveal.configure({
                keyboard: false,
                overview: false,
                help: false
            });
        }
    }
    
    // 添加错误处理
    window.addEventListener('error', function(e) {
        console.error('🔧 WebView代理：页面错误', e.error);
    });
    
    // 页面加载完成后的处理
    document.addEventListener('DOMContentLoaded', function() {
        console.log('🔧 WebView代理：页面加载完成');
    });
})();
</script>
</body>`

	// 在</body>标签前插入兼容性脚本
	modifiedHTML = strings.Replace(modifiedHTML, "</body>", webviewScript, 1)

	return modifiedHTML
}

// GetWebViewInfo 获取web-view信息，用于调试
func (h *WebViewProxyHandler) GetWebViewInfo(c *gin.Context) {
	info := gin.H{
		"message":   "WebView代理服务正常运行",
		"host":      c.Request.Host,
		"scheme":    "https",
		"headers":   c.Request.Header,
		"userAgent": c.Request.UserAgent(),
	}

	log.Printf("🔍 WebView代理信息请求: %+v", info)
	c.JSON(200, info)
}
