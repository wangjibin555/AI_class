package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// PPTGenerator PPT生成器
type PPTGenerator struct {
	outputDir string
}

// NewPPTGenerator 创建新的PPT生成器
func NewPPTGenerator(outputDir string) *PPTGenerator {
	return &PPTGenerator{
		outputDir: outputDir,
	}
}

// GeneratePPT 生成PPT文件
func (pg *PPTGenerator) GeneratePPT(summary *SummaryResult) error {
	// 首先生成Excel文件作为中间格式
	excelFile, err := pg.generateExcelFile(summary)
	if err != nil {
		return err
	}

	// 尝试转换为PPT格式
	pptFile, err := pg.convertToPPT(excelFile)
	if err != nil {
		fmt.Printf("⚠️  PPT转换失败，使用Excel格式: %v\n", err)
		fmt.Printf("Excel文件已保存: %s\n", excelFile)
		return nil
	}

	// 删除临时Excel文件
	os.Remove(excelFile)

	fmt.Printf("PPT文件已保存: %s\n", pptFile)
	return nil
}

// generateExcelFile 生成Excel文件
func (pg *PPTGenerator) generateExcelFile(summary *SummaryResult) (string, error) {
	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "内容总结"

	// 根据输入类型设置不同的工作表名称
	switch summary.Type {
	case "url":
		sheetName = "网页内容总结"
	case "pdf":
		sheetName = "PDF文档总结"
	case "docx":
		sheetName = "Word文档总结"
	case "txt":
		sheetName = "文本文件总结"
	default:
		sheetName = "文档内容总结"
	}

	f.SetSheetName("Sheet1", sheetName)

	// 创建样式
	styles := pg.createStyles(f)

	// 生成内容
	err := pg.generateContent(f, summary, styles)
	if err != nil {
		return "", err
	}

	// 生成文件名 - 根据类型生成不同的文件名
	timestamp := time.Now().Format("20060102_150405")
	var filename string

	switch summary.Type {
	case "url":
		filename = fmt.Sprintf("%s/网页内容总结_%s.xlsx", pg.outputDir, timestamp)
	case "pdf":
		filename = fmt.Sprintf("%s/PDF文档总结_%s.xlsx", pg.outputDir, timestamp)
	case "docx":
		filename = fmt.Sprintf("%s/Word文档总结_%s.xlsx", pg.outputDir, timestamp)
	case "txt":
		filename = fmt.Sprintf("%s/文本文件总结_%s.xlsx", pg.outputDir, timestamp)
	default:
		filename = fmt.Sprintf("%s/文档内容总结_%s.xlsx", pg.outputDir, timestamp)
	}

	// 保存文件
	if err := f.SaveAs(filename); err != nil {
		return "", err
	}

	return filename, nil
}

// convertToPPT 将Excel转换为PPT
func (pg *PPTGenerator) convertToPPT(excelFile string) (string, error) {
	// 检查是否有可用的转换工具
	if pg.hasPandoc() {
		return pg.convertWithPandoc(excelFile)
	}

	if pg.hasLibreOffice() {
		return pg.convertWithLibreOffice(excelFile)
	}

	// 如果没有转换工具，尝试使用Python脚本
	return pg.convertWithPython(excelFile)
}

// hasPandoc 检查是否安装了pandoc
func (pg *PPTGenerator) hasPandoc() bool {
	_, err := exec.LookPath("pandoc")
	return err == nil
}

// hasLibreOffice 检查是否安装了LibreOffice
func (pg *PPTGenerator) hasLibreOffice() bool {
	_, err := exec.LookPath("libreoffice")
	return err == nil
}

// convertWithPandoc 使用pandoc转换
func (pg *PPTGenerator) convertWithPandoc(excelFile string) (string, error) {
	pptFile := strings.Replace(excelFile, ".xlsx", ".pptx", 1)

	cmd := exec.Command("pandoc", excelFile, "-o", pptFile, "--to", "pptx")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pandoc转换失败: %v, 输出: %s", err, string(output))
	}

	return pptFile, nil
}

// convertWithLibreOffice 使用LibreOffice转换
func (pg *PPTGenerator) convertWithLibreOffice(excelFile string) (string, error) {
	pptFile := strings.Replace(excelFile, ".xlsx", ".pptx", 1)

	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pptx", "--outdir", pg.outputDir, excelFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("LibreOffice转换失败: %v, 输出: %s", err, string(output))
	}

	return pptFile, nil
}

// convertWithPython 使用Python脚本转换
func (pg *PPTGenerator) convertWithPython(excelFile string) (string, error) {
	// 检查Python脚本是否存在
	scriptPath := "convert_to_ppt.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("Python转换脚本不存在: %s", scriptPath)
	}

	pptFile := strings.Replace(excelFile, ".xlsx", ".pptx", 1)

	cmd := exec.Command("python3", scriptPath, excelFile, pptFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Python转换失败: %v, 输出: %s", err, string(output))
	}

	return pptFile, nil
}

// createStyles 创建样式
func (pg *PPTGenerator) createStyles(f *excelize.File) map[string]int {
	styles := make(map[string]int)

	// 标题样式
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  20,
			Color: "FFFFFF",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#2E86AB"},
			Pattern: 1,
		},
	})
	styles["title"] = titleStyle

	// 副标题样式
	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  16,
			Color: "2E86AB",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E6F3FF"},
			Pattern: 1,
		},
	})
	styles["subtitle"] = subtitleStyle

	// 内容样式
	contentStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size: 12,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
	})
	styles["content"] = contentStyle

	// 要点样式
	bulletStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size: 11,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
			Indent:     1,
		},
	})
	styles["bullet"] = bulletStyle

	return styles
}

// generateContent 生成PPT内容
func (pg *PPTGenerator) generateContent(f *excelize.File, summary *SummaryResult, styles map[string]int) error {
	sheetName := "内容总结"

	// 根据输入类型设置不同的工作表名称
	switch summary.Type {
	case "url":
		sheetName = "网页内容总结"
	case "pdf":
		sheetName = "PDF文档总结"
	case "docx":
		sheetName = "Word文档总结"
	case "txt":
		sheetName = "文本文件总结"
	default:
		sheetName = "文档内容总结"
	}

	f.SetSheetName("Sheet1", sheetName)

	// 设置列宽
	f.SetColWidth(sheetName, "A", "H", 15)

	// 第一页：标题页
	currentRow := 1

	// 主标题
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), summary.Title)
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), styles["title"])
	f.MergeCell(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("H%d", currentRow))
	f.SetRowHeight(sheetName, currentRow, 60)
	currentRow += 2

	// 副标题 - 根据类型显示不同的副标题
	var subtitle string
	switch summary.Type {
	case "url":
		subtitle = "网页内容AI智能总结报告"
	case "pdf":
		subtitle = "PDF文档AI智能总结报告"
	case "docx":
		subtitle = "Word文档AI智能总结报告"
	case "txt":
		subtitle = "文本文件AI智能总结报告"
	default:
		subtitle = "文档内容AI智能总结报告"
	}

	f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), subtitle)
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), styles["subtitle"])
	f.MergeCell(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("H%d", currentRow))
	f.SetRowHeight(sheetName, currentRow, 40)
	currentRow += 2

	// 来源信息
	if summary.Source != "" {
		sourceText := "来源：" + summary.Source
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), sourceText)
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), styles["content"])
		f.MergeCell(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("H%d", currentRow))
		currentRow++
	}

	// 生成时间
	timestamp := time.Now().Format("2006年01月02日 15:04:05")
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), "生成时间："+timestamp)
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), styles["content"])
	f.MergeCell(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("H%d", currentRow))
	currentRow += 3

	// 分割总结内容
	sections := pg.parseSummarySections(summary.Summary)

	// 添加各个部分
	for sectionTitle, sectionContent := range sections {
		// 部分标题
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), sectionTitle)
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), styles["subtitle"])
		f.MergeCell(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("H%d", currentRow))
		f.SetRowHeight(sheetName, currentRow, 35)
		currentRow++

		// 部分内容
		paragraphs := strings.Split(sectionContent, "\n")
		for _, paragraph := range paragraphs {
			paragraph = strings.TrimSpace(paragraph)
			if paragraph == "" {
				continue
			}

			// 检查是否是要点（以数字、字母或符号开头）
			if pg.isBulletPoint(paragraph) {
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), "• "+paragraph)
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), styles["bullet"])
			} else {
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), paragraph)
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), styles["content"])
			}

			f.MergeCell(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("H%d", currentRow))
			f.SetRowHeight(sheetName, currentRow, 50)
			currentRow++
		}
		currentRow++ // 部分间空行
	}

	return nil
}

// parseSummarySections 解析总结内容为各个部分
func (pg *PPTGenerator) parseSummarySections(summary string) map[string]string {
	sections := make(map[string]string)

	// 按行分割
	lines := strings.Split(summary, "\n")

	var currentSection string
	var currentContent strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否是新的部分标题
		if pg.isSectionTitle(line) {
			// 保存前一个部分
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(currentContent.String())
			}

			// 开始新部分
			currentSection = line
			currentContent.Reset()
		} else {
			// 添加到当前部分
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(line)
		}
	}

	// 保存最后一个部分
	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(currentContent.String())
	}

	// 如果没有识别到部分，将整个内容作为"主要内容"
	if len(sections) == 0 {
		sections["主要内容"] = summary
	}

	return sections
}

// isSectionTitle 判断是否是部分标题
func (pg *PPTGenerator) isSectionTitle(line string) bool {
	// 检查是否以数字和点开头（如 "1. 主要内容概述"）
	if len(line) > 2 && line[1] == '.' && line[0] >= '0' && line[0] <= '9' {
		return true
	}

	// 检查是否包含关键词
	keywords := []string{"概述", "要点", "信息", "结论", "总结", "主要内容", "关键要点", "重要信息"}
	for _, keyword := range keywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}

	return false
}

// isBulletPoint 判断是否是要点
func (pg *PPTGenerator) isBulletPoint(line string) bool {
	// 检查是否以数字、字母或符号开头
	if len(line) == 0 {
		return false
	}

	firstChar := line[0]

	// 数字
	if firstChar >= '0' && firstChar <= '9' {
		return true
	}

	// 字母
	if (firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z') {
		return true
	}

	// 特殊符号
	symbols := []rune{'-', '•', '·', '▪', '▫', '○', '●', '◆', '◇'}
	for _, symbol := range symbols {
		if rune(firstChar) == symbol {
			return true
		}
	}

	return false
}
