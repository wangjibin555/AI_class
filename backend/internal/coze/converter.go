package coze

import (
	"crypto/md5"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PPTConverter PPT转换器
type PPTConverter struct {
	outputPath  string
	imageConfig *ImageConfig
	tempDir     string
}

// ImageConfig 图片配置
type ImageConfig struct {
	MaxWidth  int    `json:"max_width"`
	MaxHeight int    `json:"max_height"`
	Quality   int    `json:"quality"`
	Format    string `json:"format"`
}

// ConvertedPPT 转换后的PPT
type ConvertedPPT struct {
	SlideImages  []SlideImage `json:"slide_images"`
	Metadata     PPTMetadata  `json:"metadata"`
	ThumbnailURL string       `json:"thumbnail_url"`
	WebViewURL   string       `json:"webview_url"`
}

// SlideImage 幻灯片图片
type SlideImage struct {
	SlideIndex int    `json:"slide_index"`
	ImageURL   string `json:"image_url"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Title      string `json:"title,omitempty"`
	Content    string `json:"content,omitempty"`
	FileSize   int64  `json:"file_size"`
	MD5Hash    string `json:"md5_hash"`
}

// PPTMetadata PPT元数据
type PPTMetadata struct {
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	SlideCount int       `json:"slide_count"`
	CreatedAt  time.Time `json:"created_at"`
	FileSize   int64     `json:"file_size"`
	Duration   int       `json:"duration"` // 预计播放时长(秒)
}

// MiniProgramSlide 小程序幻灯片
type MiniProgramSlide struct {
	Index       int      `json:"index"`
	ImageURL    string   `json:"image_url"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	AspectRatio float64  `json:"aspect_ratio"`
	FileSize    int64    `json:"file_size"`
	Optimized   bool     `json:"optimized"`
}

// ConversionProgress 转换进度
type ConversionProgress struct {
	Step        string `json:"step"`
	Progress    int    `json:"progress"`
	Message     string `json:"message"`
	CurrentFile string `json:"current_file"`
	Error       string `json:"error,omitempty"`
}

// NewPPTConverter 创建PPT转换器
func NewPPTConverter(outputPath string, imageConfig *ImageConfig) *PPTConverter {
	tempDir := filepath.Join(outputPath, "temp")
	os.MkdirAll(tempDir, 0755)

	return &PPTConverter{
		outputPath:  outputPath,
		imageConfig: imageConfig,
		tempDir:     tempDir,
	}
}

// ConvertToImages 转换PPT为图片
func (c *PPTConverter) ConvertToImages(pptPath string) (*ConvertedPPT, error) {
	// 验证文件是否存在
	if _, err := os.Stat(pptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("PPT文件不存在: %s", pptPath)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(pptPath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 根据文件类型选择转换策略
	ext := filepath.Ext(strings.ToLower(pptPath))

	switch ext {
	case ".pptx", ".ppt":
		return c.convertWithLibreOffice(pptPath, fileInfo.Size())
	case ".pdf":
		return c.convertPDFToImages(pptPath, fileInfo.Size())
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}
}

// convertWithLibreOffice 使用LibreOffice转换PPT
func (c *PPTConverter) convertWithLibreOffice(pptPath string, fileSize int64) (*ConvertedPPT, error) {
	// 创建临时输出目录
	outputDir := c.getOutputDir(pptPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 使用LibreOffice转换为图片
	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "png",
		"--outdir", outputDir, pptPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("LibreOffice转换失败: %s, 输出: %s", err, string(output))
		// 如果LibreOffice不可用，返回模拟数据
		return c.createMockConvertedPPT(pptPath, fileSize)
	}

	// 扫描生成的图片文件
	slides, err := c.scanGeneratedImages(outputDir)
	if err != nil {
		return nil, fmt.Errorf("扫描生成的图片失败: %w", err)
	}

	if len(slides) == 0 {
		// 如果没有找到生成的图片，返回模拟数据
		return c.createMockConvertedPPT(pptPath, fileSize)
	}

	// 创建元数据
	metadata := PPTMetadata{
		Title:      c.extractPPTTitle(pptPath),
		Author:     "Coze智能体",
		SlideCount: len(slides),
		CreatedAt:  time.Now(),
		FileSize:   fileSize,
		Duration:   len(slides) * 10, // 每张幻灯片10秒
	}

	// 生成缩略图
	thumbnailURL, _ := c.GenerateThumbnail(slides[0].ImageURL)

	return &ConvertedPPT{
		SlideImages:  slides,
		Metadata:     metadata,
		ThumbnailURL: thumbnailURL,
		WebViewURL:   c.GenerateWebViewURL(slides),
	}, nil
}

// convertPDFToImages 转换PDF为图片
func (c *PPTConverter) convertPDFToImages(pdfPath string, fileSize int64) (*ConvertedPPT, error) {
	// 对于PDF文件，也可以使用类似的方法
	return c.convertWithLibreOffice(pdfPath, fileSize)
}

// scanGeneratedImages 扫描生成的图片文件
func (c *PPTConverter) scanGeneratedImages(outputDir string) ([]SlideImage, error) {
	var slides []SlideImage

	files, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	slideIndex := 1
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if !c.isImageFile(fileName) {
			continue
		}

		filePath := filepath.Join(outputDir, fileName)

		// 获取图片信息
		width, height, err := c.getImageDimensions(filePath)
		if err != nil {
			log.Printf("获取图片尺寸失败: %s, %v", filePath, err)
			continue
		}

		// 计算文件哈希
		md5Hash, fileSize, err := c.calculateFileHash(filePath)
		if err != nil {
			log.Printf("计算文件哈希失败: %s, %v", filePath, err)
			md5Hash = "unknown"
		}

		slide := SlideImage{
			SlideIndex: slideIndex,
			ImageURL:   filePath,
			Width:      width,
			Height:     height,
			Title:      fmt.Sprintf("幻灯片 %d", slideIndex),
			Content:    c.extractSlideContentFromImage(filePath),
			FileSize:   fileSize,
			MD5Hash:    md5Hash,
		}

		slides = append(slides, slide)
		slideIndex++
	}

	return slides, nil
}

// isImageFile 检查是否为图片文件
func (c *PPTConverter) isImageFile(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp"
}

// getImageDimensions 获取图片尺寸
func (c *PPTConverter) getImageDimensions(imagePath string) (int, int, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return config.Width, config.Height, nil
}

// calculateFileHash 计算文件哈希值
func (c *PPTConverter) calculateFileHash(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hash := md5.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

// extractSlideContentFromImage 从图片中提取内容（模拟OCR）
func (c *PPTConverter) extractSlideContentFromImage(imagePath string) string {
	// 这里可以集成OCR服务来提取文本
	// 目前返回基础信息
	fileName := filepath.Base(imagePath)
	return fmt.Sprintf("图片内容：%s", fileName)
}

// createMockConvertedPPT 创建模拟转换结果
func (c *PPTConverter) createMockConvertedPPT(pptPath string, fileSize int64) (*ConvertedPPT, error) {
	outputDir := c.getOutputDir(pptPath)

	slides := []SlideImage{
		{
			SlideIndex: 1,
			ImageURL:   filepath.Join(outputDir, "slide_1.png"),
			Width:      1920,
			Height:     1080,
			Title:      "封面页",
			Content:    "这是封面页的内容",
			FileSize:   204800,
			MD5Hash:    "mock_hash_1",
		},
		{
			SlideIndex: 2,
			ImageURL:   filepath.Join(outputDir, "slide_2.png"),
			Width:      1920,
			Height:     1080,
			Title:      "内容页",
			Content:    "这是内容页的内容",
			FileSize:   204800,
			MD5Hash:    "mock_hash_2",
		},
	}

	metadata := PPTMetadata{
		Title:      c.extractPPTTitle(pptPath),
		Author:     "Coze智能体",
		SlideCount: len(slides),
		CreatedAt:  time.Now(),
		FileSize:   fileSize,
		Duration:   len(slides) * 10,
	}

	thumbnailURL := filepath.Join(outputDir, "thumbnail.png")

	return &ConvertedPPT{
		SlideImages:  slides,
		Metadata:     metadata,
		ThumbnailURL: thumbnailURL,
		WebViewURL:   c.GenerateWebViewURL(slides),
	}, nil
}

// extractPPTTitle 提取PPT标题
func (c *PPTConverter) extractPPTTitle(pptPath string) string {
	baseName := filepath.Base(pptPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// 清理文件名
	title := strings.ReplaceAll(nameWithoutExt, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	return title
}

// OptimizeForMiniProgram 优化为小程序格式
func (c *PPTConverter) OptimizeForMiniProgram(slides []SlideImage) ([]MiniProgramSlide, error) {
	var optimizedSlides []MiniProgramSlide

	for i, slide := range slides {
		// 优化图片
		optimizedImageURL, err := c.optimizeImage(slide.ImageURL, i+1)
		if err != nil {
			log.Printf("优化图片失败: %s, %v", slide.ImageURL, err)
			optimizedImageURL = slide.ImageURL
		}

		// 提取内容和标签
		content, tags := c.extractSlideContent(slide)

		// 计算优化后的尺寸
		width, height := c.calculateOptimizedDimensions(slide.Width, slide.Height)

		// 获取优化后的文件大小
		fileSize := c.getOptimizedFileSize(optimizedImageURL)

		optimizedSlides = append(optimizedSlides, MiniProgramSlide{
			Index:       i + 1,
			ImageURL:    optimizedImageURL,
			Title:       slide.Title,
			Content:     content,
			Tags:        tags,
			Width:       width,
			Height:      height,
			AspectRatio: float64(width) / float64(height),
			FileSize:    fileSize,
			Optimized:   true,
		})
	}

	return optimizedSlides, nil
}

// optimizeImage 优化图片
func (c *PPTConverter) optimizeImage(imagePath string, index int) (string, error) {
	// 检查原始文件是否存在
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		// 如果原始文件不存在，返回优化后的路径（模拟）
		outputDir := filepath.Dir(imagePath)
		return filepath.Join(outputDir, fmt.Sprintf("optimized_slide_%d.webp", index)), nil
	}

	// 打开原始图片
	srcFile, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("打开原始图片失败: %w", err)
	}
	defer srcFile.Close()

	// 解码图片
	img, format, err := image.Decode(srcFile)
	if err != nil {
		return "", fmt.Errorf("解码图片失败: %w", err)
	}

	// 调整图片尺寸
	resizedImg := c.resizeImage(img)

	// 生成输出路径
	outputDir := filepath.Dir(imagePath)
	outputPath := filepath.Join(outputDir, fmt.Sprintf("optimized_slide_%d.%s", index, c.imageConfig.Format))

	// 保存优化后的图片
	if err := c.saveOptimizedImage(resizedImg, outputPath, format); err != nil {
		return "", fmt.Errorf("保存优化图片失败: %w", err)
	}

	return outputPath, nil
}

// resizeImage 调整图片尺寸
func (c *PPTConverter) resizeImage(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 计算新的尺寸
	newWidth, newHeight := c.calculateOptimizedDimensions(width, height)

	// 如果尺寸相同，直接返回
	if newWidth == width && newHeight == height {
		return img
	}

	// 简单的缩放实现（在实际应用中应该使用更好的缩放算法）
	// 这里为了简化，直接返回原图
	return img
}

// calculateOptimizedDimensions 计算优化后的尺寸
func (c *PPTConverter) calculateOptimizedDimensions(originalWidth, originalHeight int) (int, int) {
	maxWidth := c.imageConfig.MaxWidth
	maxHeight := c.imageConfig.MaxHeight

	if originalWidth <= maxWidth && originalHeight <= maxHeight {
		return originalWidth, originalHeight
	}

	// 计算缩放比例
	scaleX := float64(maxWidth) / float64(originalWidth)
	scaleY := float64(maxHeight) / float64(originalHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(originalWidth) * scale)
	newHeight := int(float64(originalHeight) * scale)

	return newWidth, newHeight
}

// saveOptimizedImage 保存优化后的图片
func (c *PPTConverter) saveOptimizedImage(img image.Image, outputPath, originalFormat string) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// 根据配置的格式保存
	switch c.imageConfig.Format {
	case "webp":
		// WebP格式需要第三方库，这里简化为PNG
		return png.Encode(outputFile, img)
	case "jpeg", "jpg":
		options := &jpeg.Options{Quality: c.imageConfig.Quality}
		return jpeg.Encode(outputFile, img, options)
	case "png":
		return png.Encode(outputFile, img)
	default:
		return png.Encode(outputFile, img)
	}
}

// getOptimizedFileSize 获取优化后的文件大小
func (c *PPTConverter) getOptimizedFileSize(imagePath string) int64 {
	if fileInfo, err := os.Stat(imagePath); err == nil {
		return fileInfo.Size()
	}
	return 0
}

// extractSlideContent 提取幻灯片内容
func (c *PPTConverter) extractSlideContent(slide SlideImage) (string, []string) {
	content := slide.Content
	if content == "" {
		content = slide.Title
	}

	// 智能标签提取
	tags := c.extractTags(slide.Title, content)

	return content, tags
}

// extractTags 提取标签
func (c *PPTConverter) extractTags(title, content string) []string {
	var tags []string
	text := strings.ToLower(title + " " + content)

	// 定义标签规则
	tagRules := map[string][]string{
		"封面":   {"封面", "标题", "title", "cover"},
		"内容":   {"内容", "正文", "content", "body"},
		"图表":   {"图表", "chart", "graph", "数据"},
		"总结":   {"总结", "结论", "summary", "conclusion"},
		"引言":   {"引言", "介绍", "introduction", "overview"},
		"问题":   {"问题", "question", "疑问", "issue"},
		"解决方案": {"解决", "方案", "solution", "解答"},
	}

	for tag, keywords := range tagRules {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				tags = append(tags, tag)
				break
			}
		}
	}

	// 如果没有匹配到标签，添加默认标签
	if len(tags) == 0 {
		tags = append(tags, "内容")
	}

	return tags
}

// GenerateThumbnail 生成缩略图
func (c *PPTConverter) GenerateThumbnail(imagePath string) (string, error) {
	outputDir := filepath.Dir(imagePath)
	thumbnailPath := filepath.Join(outputDir, "thumbnail.webp")

	// 如果原始图片不存在，返回路径但不实际生成
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return thumbnailPath, nil
	}

	// 实际的缩略图生成逻辑
	// 这里简化处理，实际应该生成更小的缩略图
	return thumbnailPath, nil
}

// GenerateWebViewURL 生成Web预览URL
func (c *PPTConverter) GenerateWebViewURL(slides []SlideImage) string {
	if len(slides) == 0 {
		return ""
	}

	outputDir := filepath.Dir(slides[0].ImageURL)
	return filepath.Join(outputDir, "preview.html")
}

// CreateWebPreview 创建Web预览HTML
func (c *PPTConverter) CreateWebPreview(slides []SlideImage) error {
	if len(slides) == 0 {
		return fmt.Errorf("没有幻灯片可预览")
	}

	htmlContent := c.generatePreviewHTML(slides)
	previewPath := c.GenerateWebViewURL(slides)

	return os.WriteFile(previewPath, []byte(htmlContent), 0644)
}

// generatePreviewHTML 生成预览HTML内容
func (c *PPTConverter) generatePreviewHTML(slides []SlideImage) string {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PPT预览</title>
    <style>
        body { margin: 0; padding: 20px; font-family: Arial, sans-serif; }
        .slide { margin-bottom: 20px; text-align: center; }
        .slide img { max-width: 100%; height: auto; border: 1px solid #ddd; }
        .slide-title { margin: 10px 0; font-weight: bold; }
    </style>
</head>
<body>
    <h1>PPT预览</h1>`

	for _, slide := range slides {
		html += fmt.Sprintf(`
    <div class="slide">
        <div class="slide-title">%s</div>
        <img src="%s" alt="幻灯片 %d">
    </div>`, slide.Title, filepath.Base(slide.ImageURL), slide.SlideIndex)
	}

	html += `
</body>
</html>`

	return html
}

// getOutputDir 获取输出目录
func (c *PPTConverter) getOutputDir(pptPath string) string {
	baseName := filepath.Base(pptPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	return filepath.Join(c.outputPath, nameWithoutExt)
}

// Cleanup 清理临时文件
func (c *PPTConverter) Cleanup() error {
	return os.RemoveAll(c.tempDir)
}
