#!/bin/bash

# Python依赖安装脚本
echo "=== 安装Python依赖 ==="
echo ""

# 检查Python3是否安装
if ! command -v python3 &> /dev/null; then
    echo "❌ 错误: Python3未安装"
    echo "请先安装Python3"
    exit 1
fi

echo "✅ Python3已安装: $(python3 --version)"
echo ""

# 检查pip3是否安装
if ! command -v pip3 &> /dev/null; then
    echo "❌ 错误: pip3未安装"
    echo "请先安装pip3"
    exit 1
fi

echo "✅ pip3已安装: $(pip3 --version)"
echo ""

# 安装必要的Python库
echo "📦 安装Python依赖库..."
echo ""

echo "安装 python-docx..."
pip3 install python-docx
if [ $? -ne 0 ]; then
    echo "❌ python-docx安装失败"
    exit 1
fi

echo "安装 openpyxl..."
pip3 install openpyxl
if [ $? -ne 0 ]; then
    echo "❌ openpyxl安装失败"
    exit 1
fi

echo "安装 python-pptx..."
pip3 install python-pptx
if [ $? -ne 0 ]; then
    echo "❌ python-pptx安装失败"
    exit 1
fi

echo ""
echo "✅ 所有Python依赖安装完成!"
echo ""

# 测试导入
echo "🧪 测试Python库导入..."
python3 -c "
try:
    import docx
    import openpyxl
    import pptx
    print('✅ 所有库导入成功')
except ImportError as e:
    print(f'❌ 导入失败: {e}')
    exit(1)
"

if [ $? -eq 0 ]; then
    echo ""
    echo "🎉 Python环境配置完成!"
    echo "现在可以使用PPT转换和DOCX处理功能了"
else
    echo ""
    echo "❌ Python环境配置失败"
    exit 1
fi 