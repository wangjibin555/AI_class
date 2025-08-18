#!/bin/bash

echo "=== 文件处理功能测试 ==="

# 创建测试文件
echo "创建测试文件..."

# 创建测试TXT文件
cat > test_document.txt << EOF
人工智能技术发展报告

1. 概述
人工智能（AI）是计算机科学的一个分支，它企图了解智能的实质，并生产出一种新的能以人类智能相似的方式做出反应的智能机器。

2. 主要技术领域
- 机器学习：通过算法使计算机能够从数据中学习
- 深度学习：使用神经网络模拟人脑的学习过程
- 自然语言处理：让计算机理解和生成人类语言
- 计算机视觉：让计算机能够理解和分析图像

3. 应用领域
- 医疗健康：疾病诊断、药物研发
- 金融服务：风险评估、智能投顾
- 自动驾驶：环境感知、路径规划
- 教育：个性化学习、智能辅导

4. 发展趋势
- 大模型技术的突破
- 多模态AI的发展
- AI伦理和安全的重视
- 边缘AI的普及

5. 挑战与机遇
- 数据隐私保护
- 算法偏见问题
- 就业结构变化
- 技术监管需求
EOF

echo "✅ 测试TXT文件已创建: test_document.txt"

# 测试PDF文件处理
if [ -f "测试/Go基础笔记_20250721101908.pdf" ]; then
    echo "发现测试PDF文件: 测试/Go基础笔记_20250721101908.pdf"
    echo "测试PDF文件处理..."
    go run *.go --file 测试/Go基础笔记_20250721101908.pdf
    echo "✅ PDF文件处理完成"
else
    echo "⚠️  未找到测试PDF文件，跳过PDF测试"
fi

# 测试DOCX文件处理
if [ -f "测试/Go基础笔记.docx" ]; then
    echo "发现测试DOCX文件: 测试/Go基础笔记.docx"
    echo "测试DOCX文件处理..."
    go run *.go --file 测试/Go基础笔记.docx
    echo "✅ DOCX文件处理完成"
else
    echo "⚠️  未找到测试DOCX文件，跳过DOCX测试"
fi

# 测试TXT文件处理
echo "测试TXT文件处理..."
go run *.go --file test_document.txt
echo "✅ TXT文件处理完成"

echo "=== 测试完成 ==="
echo "请检查output目录中的生成文件"
echo ""
echo "生成的文件包括："
ls -la output/*.pptx 