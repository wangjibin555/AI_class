#!/bin/bash

# 网页爬取与AI总结PPT生成器 - 示例使用脚本

echo "=== 网页爬取与AI总结PPT生成器 ==="
echo ""

# 检查是否设置了API密钥
if [ -z "$QIANWEN_API_KEY" ]; then
    echo "❌ 错误: 未设置通义千问API密钥"
    echo "请设置环境变量: export QIANWEN_API_KEY='your_api_key_here'"
    echo "或者编辑 config.json 文件"
    exit 1
fi

echo "✅ API密钥已设置"
echo ""

# 检查命令行参数
if [ $# -eq 0 ]; then
    echo "使用方法: $0 <URL>"
    echo "示例: $0 https://example.com"
    echo ""
    echo "或者直接运行:"
    echo "go run main.go https://example.com"
    exit 1
fi

URL=$1
echo "开始处理URL: $URL"
echo ""

# 运行程序
echo "🚀 启动程序..."
go run main.go "$URL"

echo ""
echo "=== 处理完成 ==="
echo "请检查 output 目录中的生成文件" 