package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ai-classroom/internal/models"
	"ai-classroom/internal/services"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	pptDir     = flag.String("dir", "./ppt", "PPT文件目录")
	single     = flag.String("file", "", "单个PPT文件路径")
	courseID   = flag.Uint("course", 0, "指定课程ID进行导入")
	scanOnly   = flag.Bool("scan", false, "只扫描文件，不进行转换")
	newOnly    = flag.Bool("new", false, "只处理新文件")
	help       = flag.Bool("help", false, "显示帮助信息")
	dbHost     = flag.String("db-host", "summer-camp-dev.rwlb.rds.aliyuncs.com", "数据库主机")
	dbPort     = flag.String("db-port", "3306", "数据库端口")
	dbUser     = flag.String("db-user", "wangjibin", "数据库用户名")
	dbPassword = flag.String("db-password", "nuGM8Vb9QBRhtH#zmM", "数据库密码")
	dbName     = flag.String("db-name", "wangjibin", "数据库名称")
)

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// 连接数据库
	db, err := connectDatabase()
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	// 创建转换服务
	conversionService := services.NewPPTConversionService(db)

	// 根据参数执行不同操作
	switch {
	case *single != "":
		// 转换单个文件
		convertSingleFile(conversionService, *single)
	case *courseID > 0 && *single != "":
		// 为指定课程导入幻灯片
		importToCourse(conversionService, *courseID, *single)
	case *scanOnly:
		// 只扫描文件
		scanFiles(*pptDir)
	case *newOnly:
		// 只处理新文件
		processNewFiles(conversionService, *pptDir)
	default:
		// 批量转换
		convertBatchFiles(conversionService, *pptDir)
	}
}

// showHelp 显示帮助信息
func showHelp() {
	fmt.Println("PPT转换工具 - 将HTML格式的PPT文件转换为数据库中的幻灯片记录")
	fmt.Println()
	fmt.Println("使用方法:")
	fmt.Println("  go run main.go [选项]")
	fmt.Println()
	fmt.Println("选项:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  # 批量转换PPT目录下的所有文件")
	fmt.Println("  go run main.go -dir=./ppt")
	fmt.Println()
	fmt.Println("  # 转换单个PPT文件")
	fmt.Println("  go run main.go -file=./ppt/ppt_4_file_20250723_093200.html")
	fmt.Println()
	fmt.Println("  # 为指定课程导入幻灯片")
	fmt.Println("  go run main.go -course=4 -file=./ppt/ppt_4_file_20250723_093200.html")
	fmt.Println()
	fmt.Println("  # 只扫描文件，不进行转换")
	fmt.Println("  go run main.go -scan -dir=./ppt")
	fmt.Println()
	fmt.Println("  # 只处理新文件")
	fmt.Println("  go run main.go -new -dir=./ppt")
}

// connectDatabase 连接数据库
func connectDatabase() (*gorm.DB, error) {
	// 使用命令行参数或配置文件
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		*dbUser, *dbPassword, *dbHost, *dbPort, *dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移（确保表结构存在）
	if err := db.AutoMigrate(&models.Course{}, &models.Slide{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}

// convertSingleFile 转换单个文件
func convertSingleFile(service services.PPTConversionService, filePath string) {
	fmt.Printf("🔄 开始转换文件: %s\n", filePath)

	result, err := service.ConvertSinglePPT(filePath)
	if err != nil {
		log.Fatalf("❌ 转换失败: %v", err)
	}

	if result.Success {
		fmt.Printf("✅ 转换成功!\n")
		fmt.Printf("   课程ID: %d\n", result.CourseID)
		fmt.Printf("   幻灯片数: %d\n", result.SlidesCount)
		fmt.Printf("   处理时间: %s\n", result.ProcessedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("❌ 转换失败: %s\n", result.Error)
	}
}

// importToCourse 为指定课程导入幻灯片
func importToCourse(service services.PPTConversionService, courseID uint, filePath string) {
	fmt.Printf("🔄 为课程 %d 导入幻灯片: %s\n", courseID, filePath)

	err := service.ImportSlidesToCourse(courseID, filePath)
	if err != nil {
		log.Fatalf("❌ 导入失败: %v", err)
	}

	fmt.Printf("✅ 导入成功!\n")
}

// scanFiles 扫描文件
func scanFiles(dir string) {
	fmt.Printf("🔍 扫描目录: %s\n", dir)

	var count int
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".html" {
			if filepath.Base(path)[:4] == "ppt_" {
				fmt.Printf("   📄 %s (大小: %d bytes, 修改时间: %s)\n",
					path, info.Size(), info.ModTime().Format("2006-01-02 15:04:05"))
				count++
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("❌ 扫描失败: %v", err)
	}

	fmt.Printf("📊 扫描完成，找到 %d 个PPT文件\n", count)
}

// processNewFiles 处理新文件
func processNewFiles(service services.PPTConversionService, dir string) {
	fmt.Printf("🔄 扫描并处理新文件: %s\n", dir)

	result, err := service.ScanAndImportNewPPTs(dir)
	if err != nil {
		log.Fatalf("❌ 处理失败: %v", err)
	}

	fmt.Printf("📊 处理完成:\n")
	fmt.Printf("   新文件总数: %d\n", result.TotalFiles)
	fmt.Printf("   成功转换: %d\n", result.SuccessfulFiles)
	fmt.Printf("   转换失败: %d\n", result.FailedFiles)
	fmt.Printf("   处理时间: %s\n", result.ProcessedAt.Format("2006-01-02 15:04:05"))

	// 显示详细结果
	if len(result.Results) > 0 {
		fmt.Println("\n📋 详细结果:")
		for _, res := range result.Results {
			status := "❌"
			if res.Success {
				status = "✅"
			}
			fmt.Printf("   %s %s (课程ID: %d, 幻灯片: %d)\n",
				status, filepath.Base(res.FilePath), res.CourseID, res.SlidesCount)
			if !res.Success && res.Error != "" {
				fmt.Printf("      错误: %s\n", res.Error)
			}
		}
	}
}

// convertBatchFiles 批量转换文件
func convertBatchFiles(service services.PPTConversionService, dir string) {
	fmt.Printf("🔄 批量转换目录: %s\n", dir)

	result, err := service.ConvertBatchPPT(dir)
	if err != nil {
		log.Fatalf("❌ 批量转换失败: %v", err)
	}

	fmt.Printf("📊 批量转换完成:\n")
	fmt.Printf("   文件总数: %d\n", result.TotalFiles)
	fmt.Printf("   成功转换: %d\n", result.SuccessfulFiles)
	fmt.Printf("   转换失败: %d\n", result.FailedFiles)
	fmt.Printf("   处理时间: %s\n", result.ProcessedAt.Format("2006-01-02 15:04:05"))

	// 显示详细结果
	if len(result.Results) > 0 {
		fmt.Println("\n📋 详细结果:")
		for _, res := range result.Results {
			status := "❌"
			if res.Success {
				status = "✅"
			}
			fmt.Printf("   %s %s (课程ID: %d, 幻灯片: %d)\n",
				status, filepath.Base(res.FilePath), res.CourseID, res.SlidesCount)
			if !res.Success && res.Error != "" {
				fmt.Printf("      错误: %s\n", res.Error)
			}
		}
	}

	// 显示统计信息
	if result.TotalFiles > 0 {
		successRate := float64(result.SuccessfulFiles) / float64(result.TotalFiles) * 100
		fmt.Printf("\n📈 转换成功率: %.1f%%\n", successRate)
	}
}
