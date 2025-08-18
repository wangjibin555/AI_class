#!/bin/bash

# 完整的WAV到MP3格式修复脚本
echo "🎵 开始修复音频格式问题..."

# 1. 重命名文件系统中的音频文件
echo ""
echo "📁 第一步：重命名文件系统中的音频文件"
cd ../storage/audio/slides 2>/dev/null || {
    echo "❌ 音频存储目录不存在，跳过文件重命名"
}

if [ -d "../../../storage/audio/slides" ]; then
    cd ../../../storage/audio/slides
    
    # 统计当前文件
    wav_count=$(ls -1 *.wav 2>/dev/null | wc -l)
    mp3_count=$(ls -1 *.mp3 2>/dev/null | wc -l)
    
    echo "   当前WAV文件数量: $wav_count"
    echo "   当前MP3文件数量: $mp3_count"
    
    if [ $wav_count -gt 0 ]; then
        echo "   开始重命名WAV文件..."
        for file in *.wav; do
            if [ -f "$file" ]; then
                new_name="${file%.wav}.mp3"
                mv "$file" "$new_name"
                echo "   ✅ $file -> $new_name"
            fi
        done
        echo "   📁 文件重命名完成"
    else
        echo "   ℹ️  没有找到WAV文件"
    fi
    
    cd - > /dev/null
fi

# 2. 更新数据库中的URL
echo ""
echo "🗄️ 第二步：更新数据库中的音频URL"

# 读取数据库配置
DB_HOST=$(grep -A 10 "database:" ../configs/config.yaml | grep -o 'host: "[^"]*"' | sed 's/host: "\(.*\)"/\1/')
DB_PORT=$(grep -A 10 "database:" ../configs/config.yaml | grep -o 'port: [0-9]*' | sed 's/port: //')
DB_USER=$(grep -A 10 "database:" ../configs/config.yaml | grep -o 'username: "[^"]*"' | sed 's/username: "\(.*\)"/\1/')
DB_PASS=$(grep -A 10 "database:" ../configs/config.yaml | grep -o 'password: "[^"]*"' | sed 's/password: "\(.*\)"/\1/')
DB_NAME=$(grep -A 10 "database:" ../configs/config.yaml | grep -o 'name: "[^"]*"' | sed 's/name: "\(.*\)"/\1/')

if [ -z "$DB_PORT" ]; then
    DB_PORT=3306
fi

echo "   数据库配置:"
echo "   - 主机: $DB_HOST"
echo "   - 端口: $DB_PORT" 
echo "   - 用户: $DB_USER"
echo "   - 数据库: $DB_NAME"

# 检查是否有mysql命令
if command -v mysql &> /dev/null; then
    echo "   正在连接数据库..."
    
    # 执行SQL更新
    mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" << EOF
-- 显示更新前的统计
SELECT 'Before Update:' as status;
SELECT 
    COUNT(*) as total_slides_with_audio,
    COUNT(CASE WHEN audio_url LIKE '%.mp3' THEN 1 END) as mp3_slides,
    COUNT(CASE WHEN audio_url LIKE '%.wav' THEN 1 END) as wav_slides
FROM slides 
WHERE audio_url IS NOT NULL AND audio_url != '';

-- 更新 slides 表中的 audio_url 字段
UPDATE slides 
SET audio_url = REPLACE(audio_url, '.wav', '.mp3')
WHERE audio_url LIKE '%.wav';

-- 显示更新后的统计
SELECT 'After Update:' as status;
SELECT 
    COUNT(*) as total_slides_with_audio,
    COUNT(CASE WHEN audio_url LIKE '%.mp3' THEN 1 END) as mp3_slides,
    COUNT(CASE WHEN audio_url LIKE '%.wav' THEN 1 END) as wav_slides
FROM slides 
WHERE audio_url IS NOT NULL AND audio_url != '';

-- 显示受影响的行数
SELECT ROW_COUNT() as updated_rows;
EOF

    if [ $? -eq 0 ]; then
        echo "   ✅ 数据库更新完成"
    else
        echo "   ❌ 数据库更新失败"
    fi
else
    echo "   ⚠️  未找到mysql命令，请手动执行SQL更新:"
    echo "   mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p$DB_PASS $DB_NAME < scripts/fix_audio_urls.sql"
fi

# 3. 清理可能的缓存
echo ""
echo "🧹 第三步：清理缓存"
if [ -d "../storage/cache" ]; then
    rm -rf ../storage/cache/*
    echo "   ✅ 缓存清理完成"
else
    echo "   ℹ️  没有找到缓存目录"
fi

# 4. 重启建议
echo ""
echo "🎯 修复完成！"
echo ""
echo "⚠️  重要提醒:"
echo "   1. 请重启后端服务以确保修改生效"
echo "   2. 清理前端缓存或强制刷新页面"
echo "   3. 如果问题仍然存在，请检查新生成的音频是否为MP3格式"
echo ""
echo "🚀 建议执行："
echo "   ./main  # 重启后端服务"
echo ""