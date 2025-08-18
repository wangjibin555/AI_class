package coze

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TMPToHTMLConverter .tmp文件到HTML转换器
type TMPToHTMLConverter struct {
	config        *ConvertConfig
	templateDir   string
	outputDir     string
	tempDir       string
	cdnUploader   *CDNUploader
	fileStorage   *FileStorage
	fileValidator *FileValidator
}

// TMPFileInfo .tmp文件信息
type TMPFileInfo struct {
	OriginalPath   string    `json:"original_path"`
	DetectedType   string    `json:"detected_type"` // .pptx, .ppt, .pdf
	FileSize       int64     `json:"file_size"`
	LastModified   time.Time `json:"last_modified"`
	IsValid        bool      `json:"is_valid"`
	ProcessingPath string    `json:"processing_path"` // 重命名后的处理路径
}

// ConversionInfo转换信息
type ConversionInfo struct {
	Method               string   `json:"method"`          // "libreoffice", "backup"
	ProcessingTime       int64    `json:"processing_time"` // 处理时间（毫秒）
	LibreOfficeAvailable bool     `json:"libreoffice_available"`
	ImageCount           int      `json:"image_count"`      // 生成的图片数量
	Errors               []string `json:"errors,omitempty"` // 转换过程中的错误
}

// NewTMPToHTMLConverter 创建.tmp文件转换器
func NewTMPToHTMLConverter(config *ConvertConfig, cozeConfig *CozeConfig) *TMPToHTMLConverter {
	return &TMPToHTMLConverter{
		config:        config,
		templateDir:   "./templates/html",
		outputDir:     "./output/html",
		tempDir:       "./temp/conversion",
		cdnUploader:   NewCDNUploader(config),
		fileStorage:   NewFileStorage(cozeConfig),
		fileValidator: NewFileValidator(),
	}
}

// ConvertTMPToHTML 转换.tmp文件为HTML
func (c *TMPToHTMLConverter) ConvertTMPToHTML(tmpPath string, options *ConvertOptions) (*HTMLResult, error) {
	startTime := time.Now()
	log.Printf("开始转换.tmp文件: %s", tmpPath)

	// 1. 验证和识别.tmp文件格式
	tmpInfo, err := c.validateAndIdentifyTMPFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf(".tmp文件验证失败: %w", err)
	}

	// 2. 创建处理用的文件副本（重命名）
	processingPath, err := c.createProcessingCopy(tmpInfo)
	if err != nil {
		return nil, fmt.Errorf("创建处理副本失败: %w", err)
	}
	defer os.Remove(processingPath) // 清理临时文件

	// 3. 解析PPT结构（使用重命名后的文件）
	slides, err := c.parsePPTStructure(processingPath)
	if err != nil {
		return nil, fmt.Errorf("解析PPT结构失败: %w", err)
	}

	// 4. 提取资源文件
	resources, err := c.extractPPTResources(processingPath, slides)
	if err != nil {
		return nil, fmt.Errorf("提取PPT资源失败: %w", err)
	}

	// 5. 优化资源文件
	optimizedResources, err := c.optimizeResources(resources)
	if err != nil {
		return nil, fmt.Errorf("优化资源文件失败: %w", err)
	}

	// 6. 生成reveal.js HTML内容
	htmlContent, err := c.generateRevealJSHTML(slides, optimizedResources, options)
	if err != nil {
		return nil, fmt.Errorf("生成HTML内容失败: %w", err)
	}

	// 7. 保存HTML文件
	localPath, err := c.saveHTMLFile(htmlContent, options, tmpInfo)
	if err != nil {
		return nil, fmt.Errorf("保存HTML文件失败: %w", err)
	}

	// 8. 上传到CDN（如果配置了）
	var cdnURL string
	if c.cdnUploader != nil {
		cdnURL, err = c.uploadToCDN(localPath, optimizedResources)
		if err != nil {
			log.Printf("上传CDN失败，使用本地路径: %v", err)
		}
	}

	// 9. 生成预览URL
	previewURL := c.generatePreviewURL(cdnURL, localPath)

	// 10. 构建结果
	processingTime := time.Since(startTime).Milliseconds()
	result := &HTMLResult{
		LocalPath:   localPath,
		CDNURL:      cdnURL,
		PreviewURL:  previewURL,
		TotalSlides: len(slides),
		FileSize:    c.getFileSize(localPath),
		GeneratedAt: time.Now(),
		Slides:      slides,
		Resources:   optimizedResources,
		Metadata:    c.extractMetadata(tmpPath, slides),
		SourceInfo:  tmpInfo,
		ConversionInfo: &ConversionInfo{
			Method:               "libreoffice",
			ProcessingTime:       processingTime,
			LibreOfficeAvailable: c.isLibreOfficeAvailable(),
			ImageCount:           len(slides),
		},
	}

	log.Printf(".tmp文件转换完成: %s -> %s (耗时: %dms)", tmpPath, previewURL, processingTime)
	return result, nil
}

// validateAndIdentifyTMPFile 验证和识别.tmp文件
func (c *TMPToHTMLConverter) validateAndIdentifyTMPFile(tmpPath string) (*TMPFileInfo, error) {
	// 获取文件信息
	stat, err := os.Stat(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("无法获取文件信息: %w", err)
	}

	// 创建文件信息结构
	tmpInfo := &TMPFileInfo{
		OriginalPath: tmpPath,
		FileSize:     stat.Size(),
		LastModified: stat.ModTime(),
	}

	// 使用文件验证器检测实际格式
	detectedType, err := c.fileValidator.detectFileTypeByHeader(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("文件格式检测失败: %w", err)
	}

	tmpInfo.DetectedType = detectedType

	// 验证是否为支持的PPT格式
	supportedTypes := []string{".pptx", ".ppt", ".pdf"}
	tmpInfo.IsValid = false
	for _, supported := range supportedTypes {
		if detectedType == supported {
			tmpInfo.IsValid = true
			break
		}
	}

	if !tmpInfo.IsValid {
		return nil, fmt.Errorf("不支持的文件格式: %s", detectedType)
	}

	log.Printf(".tmp文件验证成功: 大小=%d bytes, 检测格式=%s", tmpInfo.FileSize, tmpInfo.DetectedType)
	return tmpInfo, nil
}

// createProcessingCopy 创建用于处理的文件副本（重命名扩展名）
func (c *TMPToHTMLConverter) createProcessingCopy(tmpInfo *TMPFileInfo) (string, error) {
	// 确保临时目录存在
	if err := os.MkdirAll(c.tempDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 生成处理文件名（使用检测到的扩展名）
	baseName := filepath.Base(tmpInfo.OriginalPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	processingFileName := nameWithoutExt + tmpInfo.DetectedType
	processingPath := filepath.Join(c.tempDir, processingFileName)

	// 复制文件
	srcFile, err := os.Open(tmpInfo.OriginalPath)
	if err != nil {
		return "", fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(processingPath)
	if err != nil {
		return "", fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", fmt.Errorf("复制文件失败: %w", err)
	}

	tmpInfo.ProcessingPath = processingPath
	log.Printf("创建处理副本: %s -> %s", tmpInfo.OriginalPath, processingPath)
	return processingPath, nil
}

// parsePPTStructure 解析PPT结构（适配.tmp重命名后的文件）
func (c *TMPToHTMLConverter) parsePPTStructure(processingPath string) ([]HTMLPPTSlide, error) {
	log.Printf("🚀 开始解析PPT文件: %s", processingPath)

	// 方法1：尝试使用PPTX解析器直接解析内容
	log.Printf("📋 方法1：尝试使用PPTX解析器直接解析内容")
	slides, err := c.parsePPTXContent(processingPath)
	if err != nil {
		log.Printf("⚠️ PPTX内容解析失败，尝试LibreOffice方法: %v", err)

		// 方法2：使用LibreOffice转换为图片
		imageDir, err := c.convertPPTToImages(processingPath)
		if err != nil {
			log.Printf("LibreOffice图片转换失败，使用备用方法: %v", err)
			return c.parseWithBackupMethod(processingPath)
		}
		defer os.RemoveAll(imageDir) // 清理临时图片

		// 读取转换后的图片文件
		imageFiles, err := filepath.Glob(filepath.Join(imageDir, "*.png"))
		if err != nil {
			return nil, fmt.Errorf("读取图片文件失败: %w", err)
		}

		// 对图片文件排序
		sort.Strings(imageFiles)

		// 为每个图片创建幻灯片结构
		slides = []HTMLPPTSlide{}
		for i, imagePath := range imageFiles {
			slideData := HTMLPPTSlide{
				ID:         fmt.Sprintf("slide-%d", i+1),
				Title:      fmt.Sprintf("幻灯片 %d", i+1),
				Transition: "slide",
				Duration:   5000,
				Layout:     "image-based", // 标记为基于图片的布局
			}

			// 处理图片元素
			imageElement, err := c.processSlideImage(imagePath, i+1)
			if err != nil {
				log.Printf("处理幻灯片 %d 图片失败: %v", i+1, err)
				continue
			}

			slideData.Elements = []SlideElement{imageElement}
			slideData.Content = fmt.Sprintf("幻灯片 %d 内容", i+1)

			slides = append(slides, slideData)
		}
	}

	log.Printf("PPT解析完成，共 %d 张幻灯片", len(slides))
	return slides, nil
}

// parsePPTXContent 使用PPTX解析器解析内容
func (c *TMPToHTMLConverter) parsePPTXContent(processingPath string) ([]HTMLPPTSlide, error) {
	log.Printf("🔍 尝试使用PPTX解析器解析文件: %s", processingPath)

	parser, err := NewPPTXParser(processingPath)
	if err != nil {
		log.Printf("❌ 创建PPTX解析器失败: %v", err)
		return nil, fmt.Errorf("创建PPTX解析器失败: %w", err)
	}
	defer parser.Close()
	log.Printf("✅ PPTX解析器创建成功")

	slides, err := parser.ConvertToHTMLSlides()
	if err != nil {
		log.Printf("❌ 转换PPTX为HTML幻灯片失败: %v", err)
		return nil, fmt.Errorf("转换PPTX为HTML幻灯片失败: %w", err)
	}

	if len(slides) == 0 {
		log.Printf("⚠️ PPTX文件中没有找到幻灯片")
		return nil, fmt.Errorf("PPTX文件中没有找到幻灯片")
	}

	log.Printf("🎉 PPTX解析成功，共解析出 %d 张幻灯片", len(slides))
	for i, slide := range slides {
		log.Printf("  幻灯片 %d: ID=%s, 标题=%s, 元素数量=%d", i+1, slide.ID, slide.Title, len(slide.Elements))
	}
	return slides, nil
}

// convertPPTToImages 使用LibreOffice将PPT转换为图片
func (c *TMPToHTMLConverter) convertPPTToImages(pptPath string) (string, error) {
	// 创建输出目录
	outputDir := filepath.Join(c.tempDir, "slide_images", fmt.Sprintf("%d", time.Now().Unix()))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 使用LibreOffice将PPT转换为图片
	cmd := exec.Command(
		"libreoffice",
		"--headless",
		"--convert-to", "png",
		"--outdir", outputDir,
		pptPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("LibreOffice转换失败: %v, 输出: %s", err, string(output))
	}

	log.Printf("LibreOffice转换成功，输出目录: %s", outputDir)
	return outputDir, nil
}

// processSlideImage 处理幻灯片图片
func (c *TMPToHTMLConverter) processSlideImage(imagePath string, slideIndex int) (SlideElement, error) {
	// 获取图片信息
	img, err := os.Open(imagePath)
	if err != nil {
		return SlideElement{}, fmt.Errorf("打开图片失败: %w", err)
	}
	defer img.Close()

	// 解码图片获取尺寸
	imgConfig, _, err := image.DecodeConfig(img)
	if err != nil {
		return SlideElement{}, fmt.Errorf("解码图片失败: %w", err)
	}

	// 获取文件大小
	stat, err := os.Stat(imagePath)
	if err != nil {
		return SlideElement{}, fmt.Errorf("获取图片大小失败: %w", err)
	}

	// 复制图片到资源目录
	resourcePath, err := c.copyImageToResources(imagePath, slideIndex)
	if err != nil {
		return SlideElement{}, fmt.Errorf("复制图片失败: %w", err)
	}

	element := SlideElement{
		ID:      fmt.Sprintf("slide-image-%d", slideIndex),
		Type:    "image",
		Content: resourcePath,
		Position: ElementPosition{
			X:      0,
			Y:      0,
			Width:  float64(imgConfig.Width),
			Height: float64(imgConfig.Height),
			ZIndex: 1,
		},
		Style: map[string]interface{}{
			"width":      "100%",
			"height":     "100%",
			"object-fit": "contain",
		},
		Attributes: map[string]string{
			"alt":       fmt.Sprintf("幻灯片 %d", slideIndex),
			"data-size": fmt.Sprintf("%d", stat.Size()),
		},
	}

	return element, nil
}

// copyImageToResources 复制图片到资源目录
func (c *TMPToHTMLConverter) copyImageToResources(imagePath string, slideIndex int) (string, error) {
	resourceDir := filepath.Join(c.outputDir, "resources", "images")
	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		return "", fmt.Errorf("创建资源目录失败: %w", err)
	}

	fileName := fmt.Sprintf("slide_%d.png", slideIndex)
	targetPath := filepath.Join(resourceDir, fileName)

	srcFile, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", err
	}

	return fmt.Sprintf("./resources/images/%s", fileName), nil
}

// parseWithBackupMethod 备用解析方法（当LibreOffice不可用时）
func (c *TMPToHTMLConverter) parseWithBackupMethod(processingPath string) ([]HTMLPPTSlide, error) {
	log.Printf("使用备用方法解析PPT文件")

	// 创建基础幻灯片结构
	slides := []HTMLPPTSlide{
		{
			ID:         "slide-1",
			Title:      "PPT演示文稿",
			Content:    "此PPT文件暂时无法完全解析，请下载原文件查看完整内容。",
			Transition: "slide",
			Duration:   5000,
			Layout:     "text-only",
			Elements: []SlideElement{
				{
					ID:      "text-element-1",
					Type:    "text",
					Content: "<h1>PPT演示文稿</h1><p>此PPT文件暂时无法完全解析，请下载原文件查看完整内容。</p>",
					Position: ElementPosition{
						X:      50,
						Y:      200,
						Width:  800,
						Height: 400,
						ZIndex: 1,
					},
					Style: map[string]interface{}{
						"text-align": "center",
						"font-size":  "24px",
						"color":      "#333",
					},
				},
			},
		},
	}

	return slides, nil
}

// extractPPTResources 提取PPT资源文件
func (c *TMPToHTMLConverter) extractPPTResources(processingPath string, slides []HTMLPPTSlide) ([]HTMLResource, error) {
	resources := []HTMLResource{}

	// 为每个幻灯片的图片添加资源记录
	for _, slide := range slides {
		for _, element := range slide.Elements {
			if element.Type == "image" {
				resource := HTMLResource{
					Type:      "image",
					LocalPath: element.Content,
					Size:      0, // 将在优化时计算
					Hash:      "",
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// optimizeResources 优化资源文件
func (c *TMPToHTMLConverter) optimizeResources(resources []HTMLResource) ([]HTMLResource, error) {
	optimized := make([]HTMLResource, len(resources))
	copy(optimized, resources)

	// 这里可以添加图片压缩、格式转换等优化逻辑
	for i := range optimized {
		if optimized[i].LocalPath != "" {
			if stat, err := os.Stat(optimized[i].LocalPath); err == nil {
				optimized[i].Size = stat.Size()
			}
		}
	}

	return optimized, nil
}

// generateRevealJSHTML 生成reveal.js HTML内容
func (c *TMPToHTMLConverter) generateRevealJSHTML(slides []HTMLPPTSlide, resources []HTMLResource, options *ConvertOptions) (string, error) {
	// 这里暂时生成基础的HTML模板，后续可以优化为使用模板引擎
	html := c.generateBaseHTMLTemplate(slides, resources, options)
	return html, nil
}

// generateBaseHTMLTemplate 生成基础HTML模板
func (c *TMPToHTMLConverter) generateBaseHTMLTemplate(slides []HTMLPPTSlide, resources []HTMLResource, options *ConvertOptions) string {
	theme := "league"
	if options != nil && options.Theme != "" {
		theme = options.Theme
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>PPT演示文稿</title>
    
    <!-- League Spartan字体 -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=League+Spartan:wght@300;400;500;600;700;800;900&display=swap" rel="stylesheet">
    
    <!-- reveal.js CSS -->
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@4.3.1/dist/reveal.css">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/reveal.js@4.3.1/dist/theme/%s.css">
    
    <style>
        * {
            box-sizing: border-box;
        }
        
        body {
            font-family: "League Spartan", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #e3f2fd 0%%, #bbdefb 50%%, #90caf9 100%%);
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }
        
        .reveal {
            background: transparent;
        }
        
        .reveal .slides {
            background: transparent;
        }
        
        .reveal .slides section {
            text-align: center;
            padding: 40px;
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(20px);
            border-radius: 20px;
            box-shadow: 0 8px 32px rgba(25, 118, 210, 0.15);
            margin: 20px;
            border: 1px solid rgba(25, 118, 210, 0.1);
        }
        
        .reveal .slides section h1 {
            font-size: 38px;
            font-weight: 700;
            color: #1976d2;
            margin-bottom: 30px;
            letter-spacing: 0.5px;
            line-height: 1.2;
        }
        
        .reveal .slides section h2 {
            font-size: 24px;
            font-weight: 600;
            color: #1976d2;
            margin-bottom: 25px;
            letter-spacing: 0.3px;
            line-height: 1.3;
        }
        
        .reveal .slides section p, 
        .reveal .slides section div,
        .reveal .slides section li {
            font-size: 20px;
            font-weight: 400;
            color: #424242;
            line-height: 1.6;
            margin-bottom: 20px;
            letter-spacing: 0.2px;
        }
        
        .reveal .slides section ul, 
        .reveal .slides section ol {
            text-align: left;
            margin: 0 auto;
            max-width: 80%%;
        }
        
        .reveal .slides section img {
            border: none;
            box-shadow: 0 4px 20px rgba(25, 118, 210, 0.2);
            border-radius: 12px;
            max-width: 90%%;
            max-height: 70vh;
            object-fit: contain;
            margin: 20px 0;
        }
        
        /* 移动端优化 */
        @media (max-width: 768px) {
            .reveal .slides section {
                padding: 30px 20px;
                margin: 10px;
                border-radius: 15px;
            }
            
            .reveal .slides section h1 {
                font-size: 28px;
            }
            
            .reveal .slides section h2 {
                font-size: 22px;
            }
            
            .reveal .slides section p,
            .reveal .slides section div,
            .reveal .slides section li {
                font-size: 18px;
            }
        }
        
        .custom-controls {
            position: fixed;
            bottom: 30px;
            left: 30px;
            z-index: 1000;
            display: flex;
            gap: 12px;
            flex-direction: column;
        }
        
        .custom-controls button {
            background: linear-gradient(135deg, #1976d2, #1565c0);
            border: none;
            color: white;
            padding: 12px;
            border-radius: 50%%;
            cursor: pointer;
            width: 48px;
            height: 48px;
            font-size: 16px;
            box-shadow: 0 4px 15px rgba(25, 118, 210, 0.3);
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        }
        
        .custom-controls button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(25, 118, 210, 0.4);
        }
        
        @media (max-width: 768px) {
            .custom-controls {
                bottom: 20px;
                left: 20px;
                gap: 10px;
            }
            
            .custom-controls button {
                width: 40px;
                height: 40px;
                font-size: 14px;
                padding: 10px;
            }
        }
        
        /* 进度条和控制按钮样式 */
        .reveal .progress {
            background: rgba(25, 118, 210, 0.2);
        }
        
        .reveal .progress span {
            background: #1976d2;
        }
        
        .reveal .controls {
            color: #1976d2;
        }
        
        .reveal .controls button {
            background: rgba(25, 118, 210, 0.1);
        }
        
        /* 触摸手势优化 */
        .reveal .slides {
            touch-action: pan-y;
        }
        
        /* 动画效果 */
        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateY(30px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }
        
        .reveal .slides section {
            animation: slideIn 0.6s cubic-bezier(0.4, 0, 0.2, 1);
        }
    </style>
</head>
<body>
    <div class="reveal">
        <div class="slides">`, theme)

	// 添加幻灯片内容
	for _, slide := range slides {
		html += fmt.Sprintf(`
            <section id="%s">`, slide.ID)

		for _, element := range slide.Elements {
			if element.Type == "image" {
				html += fmt.Sprintf(`
                <img src="%s" alt="%s">`, element.Content, element.Attributes["alt"])
			} else if element.Type == "text" {
				html += fmt.Sprintf(`
                <div>%s</div>`, element.Content)
			}
		}

		html += `
            </section>`
	}

	html += `
        </div>
    </div>
    
    <div class="custom-controls">
        <button onclick="previewPresentation()" title="预览">👁️</button>
        <button onclick="copyPresentationLink()" title="复制链接">🔗</button>
        <button onclick="Reveal.toggleFullscreen()" title="全屏">🔍</button>
        <button onclick="sharePresentation()" title="分享">📤</button>
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/reveal.js@4.3.1/dist/reveal.js"></script>
    <script>
        Reveal.initialize({
            hash: true,
            transition: 'slide',
            transitionSpeed: 'fast',
            controls: true,
            progress: true,
            center: true,
            touch: true,
            keyboard: true,
            mouseWheel: true,
            // 移动端优化配置
            embedded: false,
            autoSlide: 0,
            autoSlideStoppable: true,
            // 手势配置
            touchSensitivity: 15,
            // 视差背景
            parallaxBackgroundImage: '',
            parallaxBackgroundSize: '',
            // 响应式字体
            width: '100%%',
            height: '100%%',
            margin: 0.04,
            minScale: 0.2,
            maxScale: 2.0,
            // 插件
            plugins: []
        });
        
        // 移动端手势增强
        let touchStartX = 0;
        let touchStartY = 0;
        let touchEndX = 0;
        let touchEndY = 0;
        
        document.addEventListener('touchstart', function(e) {
            touchStartX = e.changedTouches[0].screenX;
            touchStartY = e.changedTouches[0].screenY;
        });
        
        document.addEventListener('touchend', function(e) {
            touchEndX = e.changedTouches[0].screenX;
            touchEndY = e.changedTouches[0].screenY;
            handleGesture();
        });
        
        function handleGesture() {
            const deltaX = touchEndX - touchStartX;
            const deltaY = touchEndY - touchStartY;
            const minSwipeDistance = 50;
            
            if (Math.abs(deltaX) > Math.abs(deltaY)) {
                // 水平滑动
                if (Math.abs(deltaX) > minSwipeDistance) {
                    if (deltaX > 0) {
                        Reveal.prev(); // 右滑：上一页
                    } else {
                        Reveal.next(); // 左滑：下一页
                    }
                }
            } else {
                // 垂直滑动
                if (Math.abs(deltaY) > minSwipeDistance) {
                    if (deltaY > 0) {
                        Reveal.up(); // 下滑：向上
                    } else {
                        Reveal.down(); // 上滑：向下
                    }
                }
            }
        }
        
        // 双击全屏
        let lastTap = 0;
        document.addEventListener('touchend', function(e) {
            const currentTime = new Date().getTime();
            const tapLength = currentTime - lastTap;
            if (tapLength < 500 && tapLength > 0) {
                if (Reveal.isFullscreen()) {
                    Reveal.toggleFullscreen();
                } else {
                    Reveal.toggleFullscreen();
                }
            }
            lastTap = currentTime;
        });
        
        // 预览功能
        function previewPresentation() {
            const currentSlide = Reveal.getCurrentSlide();
            const slideIndex = Reveal.getIndices().h + 1;
            const totalSlides = Reveal.getTotalSlides();
            
            showToast('预览第 ' + slideIndex + ' 页，共 ' + totalSlides + ' 页');
            
            // 可以在这里添加预览相关的逻辑
            // 比如显示幻灯片缩略图或预览窗口
        }
        
        // 复制链接功能
        function copyPresentationLink() {
            const url = window.location.href;
            
            if (navigator.clipboard) {
                navigator.clipboard.writeText(url).then(() => {
                    showToast('演示链接已复制到剪贴板');
                }).catch(err => {
                    console.log('复制失败:', err);
                    fallbackCopyTextToClipboard(url);
                });
            } else {
                fallbackCopyTextToClipboard(url);
            }
        }
        
        // 备用复制方法
        function fallbackCopyTextToClipboard(text) {
            const textArea = document.createElement('textarea');
            textArea.value = text;
            textArea.style.position = 'fixed';
            textArea.style.left = '-999999px';
            textArea.style.top = '-999999px';
            document.body.appendChild(textArea);
            textArea.focus();
            textArea.select();
            
            try {
                const successful = document.execCommand('copy');
                if (successful) {
                    showToast('演示链接已复制到剪贴板');
                } else {
                    showToast('复制失败，请手动复制链接');
                }
            } catch (err) {
                console.log('复制失败:', err);
                showToast('复制失败，请手动复制链接');
            }
            
            document.body.removeChild(textArea);
        }
        
        // 分享功能增强
        function sharePresentation() {
            const title = document.title || 'PPT演示文稿';
            const url = window.location.href;
            
            if (navigator.share) {
                navigator.share({
                    title: title,
                    url: url,
                    text: '查看这个精彩的PPT演示'
                }).catch(err => console.log('分享失败:', err));
            } else {
                copyPresentationLink();
            }
        }
        
        // 简单的Toast提示
        function showToast(message) {
            const toast = document.createElement('div');
            toast.textContent = message;
            toast.style.position = 'fixed';
            toast.style.top = '50%%';
            toast.style.left = '50%%';
            toast.style.transform = 'translate(-50%%, -50%%)';
            toast.style.background = 'rgba(25, 118, 210, 0.9)';
            toast.style.color = 'white';
            toast.style.padding = '12px 24px';
            toast.style.borderRadius = '8px';
            toast.style.fontSize = '16px';
            toast.style.zIndex = '10000';
            toast.style.backdropFilter = 'blur(10px)';
            document.body.appendChild(toast);
            
            setTimeout(() => {
                toast.remove();
            }, 2000);
        }
        
        // 自适应字体大小
        function adjustFontSize() {
            const sections = document.querySelectorAll('.reveal .slides section');
            const viewportWidth = window.innerWidth;
            
            sections.forEach(section => {
                if (viewportWidth < 768) {
                    section.style.fontSize = '0.8em';
                } else if (viewportWidth < 1024) {
                    section.style.fontSize = '0.9em';
                } else {
                    section.style.fontSize = '1em';
                }
            });
        }
        
        // 窗口大小改变时调整字体
        window.addEventListener('resize', adjustFontSize);
        window.addEventListener('orientationchange', adjustFontSize);
        
        // 初始化时调整字体
        document.addEventListener('DOMContentLoaded', adjustFontSize);
    </script>
</body>
</html>`

	return html
}

// saveHTMLFile 保存HTML文件
func (c *TMPToHTMLConverter) saveHTMLFile(htmlContent string, options *ConvertOptions, tmpInfo *TMPFileInfo) (string, error) {
	// 确保输出目录存在
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 生成文件名
	baseName := filepath.Base(tmpInfo.OriginalPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	htmlFileName := fmt.Sprintf("%s_%d.html", nameWithoutExt, time.Now().Unix())
	htmlPath := filepath.Join(c.outputDir, htmlFileName)

	// 保存文件
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("保存HTML文件失败: %w", err)
	}

	log.Printf("HTML文件保存成功: %s", htmlPath)
	return htmlPath, nil
}

// uploadToCDN 上传到CDN（如果配置了）
func (c *TMPToHTMLConverter) uploadToCDN(localPath string, resources []HTMLResource) (string, error) {
	if c.cdnUploader == nil {
		return "", fmt.Errorf("CDN上传器未配置")
	}

	// 这里应该实现实际的CDN上传逻辑
	// 暂时返回空字符串，表示未上传
	return "", nil
}

// generatePreviewURL 生成预览URL
func (c *TMPToHTMLConverter) generatePreviewURL(cdnURL, localPath string) string {
	if cdnURL != "" {
		return cdnURL
	}

	// 如果没有CDN，生成本地预览URL
	fileName := filepath.Base(localPath)
	return fmt.Sprintf("/api/v1/tmp/html/preview/%s", fileName)
}

// extractMetadata 提取元数据
func (c *TMPToHTMLConverter) extractMetadata(tmpPath string, slides []HTMLPPTSlide) HTMLMetadata {
	baseName := filepath.Base(tmpPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	return HTMLMetadata{
		Title:       nameWithoutExt,
		Author:      "AI课堂",
		Description: "基于.tmp文件转换的PPT演示",
		Theme:       "default",
		Language:    "zh-CN",
		Keywords:    []string{"PPT", "演示", "幻灯片"},
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
	}
}

// getFileSize 获取文件大小
func (c *TMPToHTMLConverter) getFileSize(filePath string) int64 {
	if stat, err := os.Stat(filePath); err == nil {
		return stat.Size()
	}
	return 0
}

// isLibreOfficeAvailable 检查LibreOffice是否可用
func (c *TMPToHTMLConverter) isLibreOfficeAvailable() bool {
	cmd := exec.Command("libreoffice", "--version")
	err := cmd.Run()
	return err == nil
}
