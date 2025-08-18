package middleware

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticFileOptions 静态文件服务配置
type StaticFileOptions struct {
	Root         string            // 根目录
	Index        string            // 默认索引文件
	Headers      map[string]string // 自定义响应头
	MaxAge       int               // 缓存时间（秒）
	EnableLogs   bool              // 是否启用日志
	AllowMethods []string          // 允许的HTTP方法
}

// StaticFileMiddleware 创建静态文件服务中间件
func StaticFileMiddleware(opts StaticFileOptions) gin.HandlerFunc {
	// 设置默认值
	if opts.Index == "" {
		opts.Index = "index.html"
	}
	if len(opts.AllowMethods) == 0 {
		opts.AllowMethods = []string{"GET", "HEAD"}
	}
	if opts.Headers == nil {
		opts.Headers = make(map[string]string)
	}

	return func(c *gin.Context) {
		method := c.Request.Method

		// 检查是否为允许的方法
		allowedMethod := false
		for _, allowedM := range opts.AllowMethods {
			if method == allowedM {
				allowedMethod = true
				break
			}
		}

		if !allowedMethod {
			c.Next()
			return
		}

		// 获取请求路径
		reqPath := c.Request.URL.Path

		// 移除路由前缀，获取实际文件路径
		// 这里假设路由是 /api/v1/tmp/html/preview/*filepath
		parts := strings.Split(reqPath, "/")
		if len(parts) < 6 {
			c.Next()
			return
		}

		// 提取文件路径部分
		filePath := strings.Join(parts[6:], "/")
		if filePath == "" {
			c.Next()
			return
		}

		if opts.EnableLogs {
			log.Printf("🔍 静态文件请求: %s %s -> %s", method, reqPath, filePath)
		}

		// 尝试多个可能的文件路径
		possiblePaths := []string{
			filepath.Join(opts.Root, filePath),
			filepath.Join("./"+opts.Root, filePath),
		}

		// 如果配置了工作目录
		if workingDir, err := os.Getwd(); err == nil {
			possiblePaths = append(possiblePaths, filepath.Join(workingDir, opts.Root, filePath))
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
				if opts.EnableLogs {
					log.Printf("✅ 找到静态文件: %s (大小: %d bytes)", fullPath, info.Size())
				}
				break
			}
		}

		if !fileExists {
			if opts.EnableLogs {
				log.Printf("❌ 静态文件不存在: %s", filePath)
				for i, path := range possiblePaths {
					log.Printf("  %d. %s", i+1, path)
				}
			}
			c.Next()
			return
		}

		// 设置响应头
		c.Header("Content-Type", getContentType(fullPath))
		c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		c.Header("Last-Modified", fileInfo.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))

		// 设置缓存头
		if opts.MaxAge > 0 {
			c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", opts.MaxAge))
		} else {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		// 设置自定义头
		for key, value := range opts.Headers {
			c.Header(key, value)
		}

		// 对于HEAD请求，只返回头部信息
		if method == "HEAD" {
			if opts.EnableLogs {
				log.Printf("✅ 成功响应HEAD请求: %s", fullPath)
			}
			c.Status(200)
			c.Abort()
			return
		}

		// 对于GET请求，返回文件内容
		if opts.EnableLogs {
			log.Printf("✅ 成功提供静态文件: %s", fullPath)
		}
		c.File(fullPath)
		c.Abort()
	}
}

// getContentType 根据文件扩展名获取Content-Type
func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
