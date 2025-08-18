#!/bin/bash

# 生产环境配置检查脚本
echo "🔍 检查生产环境配置..."

CONFIG_FILE="configs/config.yaml"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ 配置文件不存在: $CONFIG_FILE"
    exit 1
fi

echo "✅ 配置文件存在: $CONFIG_FILE"

# 检查关键配置项
echo ""
echo "📋 检查关键配置项:"

# 检查阿里云DashScope API配置
DASHSCOPE_API_KEY=$(grep -o 'api_key: "[^"]*"' $CONFIG_FILE | head -1 | sed 's/api_key: "\(.*\)"/\1/')
if [ -z "$DASHSCOPE_API_KEY" ] || [ "$DASHSCOPE_API_KEY" = "your_api_key_here" ]; then
    echo "❌ DashScope API Key 未配置或使用默认值"
    echo "   请在 $CONFIG_FILE 中设置 aliyun.dashscope.api_key"
else
    echo "✅ DashScope API Key 已配置 (${DASHSCOPE_API_KEY:0:8}...)"
fi

# 检查阿里云TTS配置
ACCESS_KEY_ID=$(grep -o 'access_key_id: "[^"]*"' $CONFIG_FILE | sed 's/access_key_id: "\(.*\)"/\1/')
ACCESS_KEY_SECRET=$(grep -o 'access_key_secret: "[^"]*"' $CONFIG_FILE | sed 's/access_key_secret: "\(.*\)"/\1/')
TTS_APP_KEY=$(grep -o 'app_key: "[^"]*"' $CONFIG_FILE | sed 's/app_key: "\(.*\)"/\1/')

if [ -z "$ACCESS_KEY_ID" ] || [ "$ACCESS_KEY_ID" = "your_access_key_id" ]; then
    echo "❌ 阿里云 Access Key ID 未配置或使用默认值"
else
    echo "✅ 阿里云 Access Key ID 已配置 (${ACCESS_KEY_ID:0:8}...)"
fi

if [ -z "$ACCESS_KEY_SECRET" ] || [ "$ACCESS_KEY_SECRET" = "your_access_key_secret" ]; then
    echo "❌ 阿里云 Access Key Secret 未配置或使用默认值"
else
    echo "✅ 阿里云 Access Key Secret 已配置 (${ACCESS_KEY_SECRET:0:8}...)"
fi

if [ -z "$TTS_APP_KEY" ] || [ "$TTS_APP_KEY" = "your_app_key" ]; then
    echo "❌ 阿里云 TTS App Key 未配置或使用默认值"
else
    echo "✅ 阿里云 TTS App Key 已配置: $TTS_APP_KEY"
fi

# 检查数据库配置
DB_HOST=$(grep -A 10 "database:" $CONFIG_FILE | grep -o 'host: "[^"]*"' | sed 's/host: "\(.*\)"/\1/')
DB_NAME=$(grep -A 10 "database:" $CONFIG_FILE | grep -o 'name: "[^"]*"' | sed 's/name: "\(.*\)"/\1/')

if [ -z "$DB_HOST" ]; then
    echo "❌ 数据库主机未配置"
else
    echo "✅ 数据库主机已配置: $DB_HOST"
fi

if [ -z "$DB_NAME" ]; then
    echo "❌ 数据库名称未配置"
else
    echo "✅ 数据库名称已配置: $DB_NAME"
fi

echo ""
echo "🎯 生产环境配置检查完成!"
echo ""
echo "⚠️  注意事项:"
echo "   1. 确保所有API密钥都是有效的生产环境密钥"
echo "   2. 数据库连接信息指向生产环境数据库"
echo "   3. 服务器端口配置正确"
echo "   4. 存储路径和URL配置正确"
echo ""
echo "🚀 如果所有配置都正确，可以启动生产环境服务"