#!/bin/bash

echo "=== 网页爬取与AI总结PPT生成器 - 使用示例 ==="
echo ""

# 检查API密钥
if [ -z "$QIANWEN_API_KEY" ]; then
    echo "⚠️  请先设置通义千问API密钥："
    echo "   export QIANWEN_API_KEY=\"your_api_key_here\""
    echo ""
    exit 1
fi

echo "✅ API密钥已设置"
echo ""

echo "=== 功能演示 ==="
echo ""

echo "1. 网页爬取模式："
echo "   go run *.go https://example.com"
echo ""

echo "2. 文件处理模式："
echo "   # 处理PDF文件"
echo "   go run *.go --file document.pdf"
echo ""
echo "   # 处理Word文档"
echo "   go run *.go --file report.docx"
echo ""
echo "   # 处理文本文件"
echo "   go run *.go --file notes.txt"
echo ""

echo "=== 快速测试 ==="
echo ""

# 创建示例文本文件
echo "创建示例文本文件..."
cat > example.txt << EOF
区块链技术应用报告

区块链是一种分布式数据库技术，具有去中心化、不可篡改、可追溯等特点。

主要应用领域：
1. 金融科技：数字货币、智能合约、跨境支付
2. 供应链管理：产品溯源、防伪认证
3. 数字身份：身份认证、隐私保护
4. 版权保护：数字版权、知识产权

技术优势：
- 数据安全性高
- 透明度强
- 降低信任成本
- 提高效率

发展趋势：
- 与AI、IoT技术融合
- 监管政策完善
- 企业级应用普及
- 性能优化提升
EOF

echo "✅ 示例文件已创建: example.txt"
echo ""

echo "开始处理示例文件..."
go run *.go --file example.txt

echo ""
echo "=== 处理完成 ==="
echo "请检查output目录中的生成文件"
echo ""

# 清理示例文件
rm -f example.txt
echo "✅ 示例文件已清理" 