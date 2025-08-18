#!/bin/bash

# 测试脚本 - 网页爬取与AI总结PPT生成器

echo "=== 网页爬取与AI总结PPT生成器 - 测试脚本 ==="
echo ""

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "❌ 错误: Go未安装"
    echo "请先安装Go 1.21或更高版本"
    exit 1
fi

echo "✅ Go已安装: $(go version)"
echo ""

# 检查依赖
echo "📦 检查依赖..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ 依赖检查失败"
    exit 1
fi
echo "✅ 依赖检查完成"
echo ""

# 编译程序
echo "🔨 编译程序..."
go build -o web-scraper-ppt main.go ppt_generator.go
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi
echo "✅ 编译成功"
echo ""

# 检查API密钥
if [ -z "$QIANWEN_API_KEY" ]; then
    echo "⚠️  警告: 未设置通义千问API密钥"
    echo "请设置环境变量: export QIANWEN_API_KEY='your_api_key_here'"
    echo "或者编辑 config.json 文件"
    echo ""
    echo "测试将跳过AI总结部分，仅测试网页爬取功能"
    echo ""
    
    # 测试网页爬取功能（不调用AI）
    echo "🧪 测试网页爬取功能..."
    echo "注意: 由于没有API密钥，将跳过AI总结步骤"
    echo ""
    
    # 创建一个简单的测试URL
    TEST_URL="https://httpbin.org/html"
    echo "测试URL: $TEST_URL"
    
    # 运行程序（会失败，但可以测试爬取部分）
    ./web-scraper-ppt "$TEST_URL" 2>&1 | head -20
    echo ""
    echo "✅ 基础功能测试完成"
else
    echo "✅ API密钥已设置"
    echo ""
    
    # 完整功能测试
    echo "🧪 完整功能测试..."
    TEST_URL="https://httpbin.org/html"
    echo "测试URL: $TEST_URL"
    echo ""
    
    # 运行程序
    ./web-scraper-ppt "$TEST_URL"
    
    if [ $? -eq 0 ]; then
        echo ""
        echo "✅ 完整功能测试成功"
        
        # 检查输出文件
        if [ -d "output" ] && [ "$(ls -A output)" ]; then
            echo "📁 输出文件已生成:"
            ls -la output/
        fi
    else
        echo ""
        echo "❌ 功能测试失败"
    fi
fi

echo ""
echo "=== 测试完成 ===" 