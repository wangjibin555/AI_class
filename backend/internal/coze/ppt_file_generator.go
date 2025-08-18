package coze

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// PPTFileGenerator PPT文件生成器
type PPTFileGenerator struct {
	outputDir string
	tempDir   string
}

// NewPPTFileGenerator 创建PPT文件生成器
func NewPPTFileGenerator(outputDir string) *PPTFileGenerator {
	tempDir := filepath.Join(outputDir, "temp")
	return &PPTFileGenerator{
		outputDir: outputDir,
		tempDir:   tempDir,
	}
}

// GeneratePPTFile 生成PPT文件
func (g *PPTFileGenerator) GeneratePPTFile(structure *PPTStructure, taskID string) (string, error) {
	log.Printf("开始生成PPT文件，taskID: %s", taskID)

	// 确保输出目录存在
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		log.Printf("创建输出目录失败: %v", err)
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	if err := os.MkdirAll(g.tempDir, 0755); err != nil {
		log.Printf("创建临时目录失败: %v", err)
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	log.Printf("目录检查完成，开始生成Excel文件")

	// 首先创建Excel文件作为中间格式
	excelPath, err := g.generateExcelFromStructure(structure, taskID)
	if err != nil {
		log.Printf("生成Excel文件失败: %v，直接生成HTML版本", err)
		// 如果Excel也失败，直接生成HTML版本
		htmlFilename := fmt.Sprintf("coze_ppt_%s_%d.html", taskID, time.Now().Unix())
		htmlPath := filepath.Join(g.outputDir, htmlFilename)

		// 创建基本的HTML内容
		htmlContent := g.generateHTMLFromPPTStructure(structure)
		if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
			return "", fmt.Errorf("生成HTML文件失败: %w", err)
		}
		return htmlPath, nil
	}

	log.Printf("Excel文件生成成功: %s", excelPath)

	// 生成文件名
	filename := fmt.Sprintf("coze_ppt_%s_%d.pptx", taskID, time.Now().Unix())
	outputPath := filepath.Join(g.outputDir, filename)

	// 尝试转换为PPT
	pptPath, err := g.convertExcelToPPT(excelPath, outputPath)
	if err != nil {
		log.Printf("PPT转换失败: %v，使用Excel生成HTML版本", err)
		// 使用已生成的Excel文件生成HTML版本
		htmlFilename := fmt.Sprintf("coze_ppt_%s_%d.html", taskID, time.Now().Unix())
		htmlPath := filepath.Join(g.outputDir, htmlFilename)
		htmlResult, htmlErr := g.generateHTMLVersion(excelPath, htmlPath)
		// 清理Excel文件
		os.Remove(excelPath)
		if htmlErr != nil {
			return "", fmt.Errorf("PPT和HTML生成都失败: PPT错误=%v, HTML错误=%v", err, htmlErr)
		}
		return htmlResult, nil
	}

	log.Printf("PPT生成成功: %s", pptPath)

	// 清理临时文件
	os.Remove(excelPath)

	return pptPath, nil
}

// generateExcelFromStructure 从PPT结构生成Excel文件
func (g *PPTFileGenerator) generateExcelFromStructure(structure *PPTStructure, taskID string) (string, error) {
	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "PPT内容"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", fmt.Errorf("创建工作表失败: %w", err)
	}

	// 删除默认的Sheet1
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)

	// 创建样式
	styles := g.createExcelStyles(f)

	// 生成内容
	if err := g.generateExcelContent(f, structure, styles, sheetName); err != nil {
		return "", fmt.Errorf("生成Excel内容失败: %w", err)
	}

	// 保存文件
	filename := fmt.Sprintf("temp_ppt_%s_%d.xlsx", taskID, time.Now().Unix())
	excelPath := filepath.Join(g.tempDir, filename)

	if err := f.SaveAs(excelPath); err != nil {
		return "", fmt.Errorf("保存Excel文件失败: %w", err)
	}

	return excelPath, nil
}

// createExcelStyles 创建Excel样式
func (g *PPTFileGenerator) createExcelStyles(f *excelize.File) map[string]int {
	styles := make(map[string]int)

	// 标题样式
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Italic: false,
			Family: "Microsoft YaHei",
			Size:   18,
			Color:  "#1F497D",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E7F3FF"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#1F497D", Style: 2},
			{Type: "top", Color: "#1F497D", Style: 2},
			{Type: "bottom", Color: "#1F497D", Style: 2},
			{Type: "right", Color: "#1F497D", Style: 2},
		},
	})
	styles["title"] = titleStyle

	// 子标题样式
	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Family: "Microsoft YaHei",
			Size:   14,
			Color:  "#17365D",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F2F8FF"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#B8CCE4", Style: 1},
			{Type: "top", Color: "#B8CCE4", Style: 1},
			{Type: "bottom", Color: "#B8CCE4", Style: 1},
			{Type: "right", Color: "#B8CCE4", Style: 1},
		},
	})
	styles["subtitle"] = subtitleStyle

	// 内容样式
	contentStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "Microsoft YaHei",
			Size:   11,
			Color:  "#000000",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
		},
	})
	styles["content"] = contentStyle

	// 要点样式
	bulletStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "Microsoft YaHei",
			Size:   10,
			Color:  "#333333",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F9F9F9"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#E6E6E6", Style: 1},
			{Type: "top", Color: "#E6E6E6", Style: 1},
			{Type: "bottom", Color: "#E6E6E6", Style: 1},
			{Type: "right", Color: "#E6E6E6", Style: 1},
		},
	})
	styles["bullet"] = bulletStyle

	return styles
}

// generateExcelContent 生成Excel内容
func (g *PPTFileGenerator) generateExcelContent(f *excelize.File, structure *PPTStructure, styles map[string]int, sheetName string) error {
	row := 1

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 25)
	f.SetColWidth(sheetName, "B", "B", 80)
	f.SetColWidth(sheetName, "C", "C", 50)

	// 设置标题行
	f.SetCellValue(sheetName, "A1", "幻灯片")
	f.SetCellValue(sheetName, "B1", "标题")
	f.SetCellValue(sheetName, "C1", "内容")
	f.SetCellStyle(sheetName, "A1", "C1", styles["title"])
	f.SetRowHeight(sheetName, 1, 30)
	row++

	// 添加PPT元数据
	f.SetCellValue(sheetName, "A2", "信息")
	f.SetCellValue(sheetName, "B2", "PPT元数据")
	metaInfo := fmt.Sprintf("标题：%s\n作者：%s\n创建时间：%s\n总页数：%d\n模板：%s",
		structure.Title, structure.Author, structure.CreatedAt.Format("2006-01-02 15:04:05"),
		structure.SlideCount, structure.Template)
	f.SetCellValue(sheetName, "C2", metaInfo)
	f.SetCellStyle(sheetName, "A2", "C2", styles["subtitle"])
	f.SetRowHeight(sheetName, 2, 60)
	row++

	// 添加空行
	row++

	// 生成每个幻灯片的内容
	for _, slide := range structure.Slides {
		// 幻灯片标题行
		slideNum := fmt.Sprintf("第%d页", slide.Index)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), slideNum)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), slide.Title)

		// 格式化幻灯片类型
		slideTypeDesc := g.getSlideTypeDescription(slide.SlideType)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), slideTypeDesc)
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), styles["subtitle"])
		f.SetRowHeight(sheetName, row, 25)
		row++

		// 主要内容
		if len(slide.Content) > 0 {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "")
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), "主要内容")
			contentText := strings.Join(slide.Content, "\n\n")
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), contentText)
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), styles["content"])
			f.SetRowHeight(sheetName, row, g.calculateRowHeight(contentText))
			row++
		}

		// 要点内容
		if len(slide.BulletPoints) > 0 {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "")
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), "要点")
			bulletText := ""
			for i, point := range slide.BulletPoints {
				bulletText += fmt.Sprintf("• %s", point)
				if i < len(slide.BulletPoints)-1 {
					bulletText += "\n"
				}
			}
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), bulletText)
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), styles["bullet"])
			f.SetRowHeight(sheetName, row, g.calculateRowHeight(bulletText))
			row++
		}

		// 演讲者备注
		if slide.Speaker != "" {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "")
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), "演讲备注")
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), slide.Speaker)
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), styles["content"])
			f.SetRowHeight(sheetName, row, g.calculateRowHeight(slide.Speaker))
			row++
		}

		// 添加空行分隔
		row++
	}

	return nil
}

// getSlideTypeDescription 获取幻灯片类型描述
func (g *PPTFileGenerator) getSlideTypeDescription(slideType string) string {
	switch slideType {
	case "title":
		return "标题页"
	case "content":
		return "内容页"
	case "summary":
		return "总结页"
	default:
		return "普通页"
	}
}

// calculateRowHeight 计算行高
func (g *PPTFileGenerator) calculateRowHeight(text string) float64 {
	lines := strings.Count(text, "\n") + 1
	charCount := len(text)

	// 基础行高
	baseHeight := 20.0

	// 根据行数调整
	height := baseHeight * float64(lines)

	// 根据字符数调整（长文本需要更多空间）
	if charCount > 100 {
		height += float64(charCount/100) * 5
	}

	// 限制最大和最小高度
	if height < 20 {
		height = 20
	}
	if height > 200 {
		height = 200
	}

	return height
}

// convertExcelToPPT 将Excel文件转换为PPT
func (g *PPTFileGenerator) convertExcelToPPT(excelPath, outputPath string) (string, error) {
	// 首先尝试使用LibreOffice转换
	if g.hasLibreOffice() {
		return g.convertWithLibreOffice(excelPath, outputPath)
	}

	// 如果LibreOffice不可用，生成HTML版本作为降级
	return g.generateHTMLVersion(excelPath, outputPath)
}

// hasLibreOffice 检查是否安装了LibreOffice
func (g *PPTFileGenerator) hasLibreOffice() bool {
	_, err := exec.LookPath("libreoffice")
	return err == nil
}

// convertWithLibreOffice 使用LibreOffice转换
func (g *PPTFileGenerator) convertWithLibreOffice(excelPath, outputPath string) (string, error) {
	// 使用LibreOffice将Excel转换为PPT
	outputDir := filepath.Dir(outputPath)
	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pptx", "--outdir", outputDir, excelPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("LibreOffice转换失败: %v, 输出: %s", err, string(output))
	}

	// LibreOffice会根据输入文件名生成输出文件
	baseName := strings.TrimSuffix(filepath.Base(excelPath), filepath.Ext(excelPath))
	generatedPath := filepath.Join(outputDir, baseName+".pptx")

	// 如果生成的文件名与目标文件名不同，重命名
	if generatedPath != outputPath {
		if err := os.Rename(generatedPath, outputPath); err != nil {
			return generatedPath, nil // 返回生成的文件路径
		}
	}

	return outputPath, nil
}

// generateHTMLVersion 生成HTML版本作为降级选项
func (g *PPTFileGenerator) generateHTMLVersion(excelPath, outputPath string) (string, error) {
	// 读取Excel文件
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		return "", fmt.Errorf("打开Excel文件失败: %w", err)
	}
	defer f.Close()

	// 获取工作表名称
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		sheetName = "PPT内容"
	}

	// 读取所有行
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return "", fmt.Errorf("读取Excel行失败: %w", err)
	}

	// 生成HTML内容
	htmlContent := g.generateHTMLFromExcel(rows)

	// 保存HTML文件
	htmlPath := strings.TrimSuffix(outputPath, ".pptx") + ".html"
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("保存HTML文件失败: %w", err)
	}

	return htmlPath, nil
}

// generateHTMLFromExcel 从Excel数据生成HTML
func (g *PPTFileGenerator) generateHTMLFromExcel(rows [][]string) string {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Coze生成的PPT</title>
    <style>
        body {
            font-family: 'Microsoft YaHei', 'PingFang SC', Arial, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 10px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .slide {
            padding: 40px;
            border-bottom: 2px solid #f0f0f0;
            min-height: 400px;
        }
        .slide:last-child {
            border-bottom: none;
        }
        .slide-title {
            font-size: 24px;
            font-weight: bold;
            color: #1F497D;
            margin-bottom: 20px;
            border-bottom: 3px solid #1F497D;
            padding-bottom: 10px;
        }
        .slide-content {
            font-size: 16px;
            color: #333;
            margin-bottom: 15px;
            white-space: pre-wrap;
        }
        .slide-bullets {
            background: #f9f9f9;
            padding: 15px;
            border-left: 4px solid #1F497D;
            margin: 15px 0;
        }
        .slide-bullets ul {
            margin: 0;
            padding-left: 20px;
        }
        .slide-bullets li {
            margin: 8px 0;
            color: #555;
        }
        .meta-info {
            background: #E7F3FF;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
            color: #1F497D;
        }
    </style>
</head>
<body>
    <div class="container">`

	currentSlide := ""
	currentContent := ""
	currentBullets := ""

	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue // 跳过标题行或不完整的行
		}

		slideNum := strings.TrimSpace(row[0])
		title := strings.TrimSpace(row[1])
		content := strings.TrimSpace(row[2])

		// 如果是新的幻灯片
		if slideNum != "" && slideNum != currentSlide {
			// 保存上一张幻灯片
			if currentSlide != "" {
				html += g.formatSlideHTML(currentSlide, currentContent, currentBullets)
			}

			// 开始新的幻灯片
			currentSlide = title
			currentContent = ""
			currentBullets = ""
		}

		// 处理内容
		if title == "主要内容" {
			currentContent = content
		} else if title == "要点" {
			currentBullets = content
		} else if title == "PPT元数据" {
			html += fmt.Sprintf(`<div class="meta-info">%s</div>`, strings.ReplaceAll(content, "\n", "<br/>"))
		}
	}

	// 添加最后一张幻灯片
	if currentSlide != "" {
		html += g.formatSlideHTML(currentSlide, currentContent, currentBullets)
	}

	html += `    </div>
</body>
</html>`

	return html
}

// formatSlideHTML 格式化幻灯片HTML
func (g *PPTFileGenerator) formatSlideHTML(title, content, bullets string) string {
	html := fmt.Sprintf(`        <div class="slide">
            <div class="slide-title">%s</div>`, title)

	if content != "" {
		html += fmt.Sprintf(`
            <div class="slide-content">%s</div>`, strings.ReplaceAll(content, "\n", "<br/>"))
	}

	if bullets != "" {
		html += `
            <div class="slide-bullets">
                <ul>`

		bulletLines := strings.Split(bullets, "\n")
		for _, bullet := range bulletLines {
			bullet = strings.TrimSpace(bullet)
			if bullet != "" {
				// 移除bullet符号
				bullet = strings.TrimPrefix(bullet, "• ")
				bullet = strings.TrimPrefix(bullet, "- ")
				bullet = strings.TrimPrefix(bullet, "* ")
				html += fmt.Sprintf(`
                    <li>%s</li>`, bullet)
			}
		}

		html += `
                </ul>
            </div>`
	}

	html += `
        </div>`

	return html
}

// GenerateFileInfo 生成文件信息
func (g *PPTFileGenerator) GenerateFileInfo(filePath string, structure *PPTStructure) *GeneratedFileInfo {
	fileInfo, _ := os.Stat(filePath)
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	return &GeneratedFileInfo{
		FilePath:    filePath,
		FileName:    filepath.Base(filePath),
		FileSize:    fileSize,
		FileType:    g.getFileExtension(filePath),
		CreatedAt:   time.Now(),
		SlideCount:  structure.SlideCount,
		Title:       structure.Title,
		Author:      structure.Author,
		Description: structure.Description,
	}
}

// getFileExtension 获取文件扩展名
func (g *PPTFileGenerator) getFileExtension(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pptx":
		return "pptx"
	case ".html":
		return "html"
	case ".xlsx":
		return "xlsx"
	default:
		return "unknown"
	}
}

// GeneratedFileInfo 生成的文件信息
type GeneratedFileInfo struct {
	FilePath    string    `json:"file_path"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	FileType    string    `json:"file_type"`
	CreatedAt   time.Time `json:"created_at"`
	SlideCount  int       `json:"slide_count"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
}

// generateHTMLFromPPTStructure 直接从PPT结构生成HTML
func (g *PPTFileGenerator) generateHTMLFromPPTStructure(structure *PPTStructure) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>`)
	html.WriteString(structure.Title)
	html.WriteString(`</title>
    <style>
        body {
            font-family: 'Microsoft YaHei', 'Helvetica Neue', Arial, sans-serif;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
            color: white;
            padding: 40px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .meta {
            margin-top: 20px;
            font-size: 1.1em;
            opacity: 0.9;
        }
        .slide {
            padding: 40px;
            border-bottom: 1px solid #eee;
            transition: all 0.3s ease;
        }
        .slide:hover {
            background-color: #f8f9fa;
        }
        .slide:last-child {
            border-bottom: none;
        }
        .slide-number {
            display: inline-block;
            background: #007bff;
            color: white;
            padding: 8px 16px;
            border-radius: 25px;
            font-size: 0.9em;
            margin-bottom: 20px;
            font-weight: 500;
        }
        .slide-title {
            font-size: 1.8em;
            color: #333;
            margin-bottom: 20px;
            font-weight: 600;
        }
        .slide-content {
            font-size: 1.1em;
            color: #555;
            margin-bottom: 20px;
            white-space: pre-line;
        }
        .bullet-points {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
            border-left: 4px solid #007bff;
        }
        .bullet-points ul {
            margin: 0;
            padding-left: 20px;
        }
        .bullet-points li {
            margin-bottom: 8px;
            color: #666;
        }
        .slide-type {
            font-size: 0.9em;
            color: #888;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 10px;
        }
        @media (max-width: 768px) {
            body { padding: 10px; }
            .slide { padding: 20px; }
            .header { padding: 20px; }
            .header h1 { font-size: 2em; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>`)
	html.WriteString(structure.Title)
	html.WriteString(`</h1>
            <div class="meta">
                <p>作者：`)
	html.WriteString(structure.Author)
	html.WriteString(` | 创建时间：`)
	html.WriteString(structure.CreatedAt.Format("2006-01-02 15:04"))
	html.WriteString(` | 共 `)
	html.WriteString(fmt.Sprintf("%d", structure.SlideCount))
	html.WriteString(` 页</p>
                <p>`)
	html.WriteString(structure.Description)
	html.WriteString(`</p>
            </div>
        </div>`)

	// 生成每个幻灯片
	for _, slide := range structure.Slides {
		html.WriteString(`
        <div class="slide">
            <div class="slide-number">第 `)
		html.WriteString(fmt.Sprintf("%d", slide.Index))
		html.WriteString(` 页</div>
            <div class="slide-type">`)
		html.WriteString(slide.SlideType)
		html.WriteString(`</div>
            <div class="slide-title">`)
		html.WriteString(slide.Title)
		html.WriteString(`</div>`)

		if len(slide.Content) > 0 {
			html.WriteString(`
            <div class="slide-content">`)
			html.WriteString(strings.Join(slide.Content, "\n\n"))
			html.WriteString(`</div>`)
		}

		if len(slide.BulletPoints) > 0 {
			html.WriteString(`
            <div class="bullet-points">
                <ul>`)
			for _, point := range slide.BulletPoints {
				html.WriteString(`
                    <li>`)
				html.WriteString(point)
				html.WriteString(`</li>`)
			}
			html.WriteString(`
                </ul>
            </div>`)
		}

		html.WriteString(`
        </div>`)
	}

	html.WriteString(`
    </div>
</body>
</html>`)

	return html.String()
}
