#!/bin/bash

# 性能测试启动脚本

echo "🚀 服务器性能测试工具"
echo "========================"
echo ""

# 检查Node.js是否安装
if ! command -v node &> /dev/null; then
    echo "❌ 错误: 未找到Node.js"
    echo "请先安装Node.js: https://nodejs.org/"
    exit 1
fi

echo "✅ Node.js版本: $(node --version)"
echo ""

# 显示菜单
echo "请选择测试模式:"
echo "1) 快速测试 (简化版，200个请求)"
echo "2) 完整测试 (全功能版，60秒压测)"
echo "3) 高耗时API测试 (音频生成、PPT生成等)"
echo "4) 音频PPT生成流程测试 (完整生成流程)"
echo "5) 高耗时API性能测试 (20并发客户端，60秒压测)"
echo "6) 查看使用说明"
echo ""

read -p "请输入选择 (1-6): " choice

case $choice in
    1)
        echo ""
        echo "🏃‍♂️ 启动快速测试..."
        echo "目标: 20个并发客户端，每客户端10个请求"
        echo ""
        node simple_performance_test.js
        ;;
    2)
        echo ""
        echo "🏃‍♂️ 启动完整测试..."
        echo "目标: 20个并发客户端，60秒压力测试"
        echo ""
        node performance_test.js
        ;;
    3)
        echo ""
        echo "🧪 启动高耗时API测试..."
        echo "测试: 音频生成、PPT生成、AI分析等高耗时接口"
        echo ""
        node test_high_cost_apis.js
        ;;
    4)
        echo ""
        echo "🎨 启动音频生成和DashScope PPT生成流程测试..."
        echo "测试: 完整的课件创建->音频生成->DashScope PPT生成流程"
        echo ""
        node test_audio_ppt_generation.js
        ;;
    5)
        echo ""
        echo "💪 启动高耗时API性能测试..."
        echo "测试: 20个并发客户端，60秒高强度压测DashScope和音频相关API"
        echo ""
        node test_high_cost_apis_performance.js
        ;;
    6)
        echo ""
        echo "📖 查看使用说明..."
        if command -v cat &> /dev/null; then
            cat "性能测试使用说明.md"
        else
            echo "请手动查看文件: 性能测试使用说明.md"
        fi
        ;;
    *)
        echo "❌ 无效选择，请重新运行脚本"
        exit 1
        ;;
esac

echo ""
echo "✅ 测试完成！"
