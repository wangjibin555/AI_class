package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// 音频文件存储路径
	audioDir := "../storage/audio/slides"

	fmt.Println("🔍 开始扫描音频文件...")

	err := filepath.Walk(audioDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查是否是 .wav 文件
		if strings.HasSuffix(strings.ToLower(info.Name()), ".wav") {
			// 生成新的 .mp3 文件名
			newPath := strings.TrimSuffix(path, ".wav") + ".mp3"

			fmt.Printf("📁 重命名: %s -> %s\n", info.Name(), filepath.Base(newPath))

			// 重命名文件
			err := os.Rename(path, newPath)
			if err != nil {
				fmt.Printf("❌ 重命名失败: %v\n", err)
				return err
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("❌ 扫描失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 音频文件重命名完成!")
}
