#!/bin/bash

echo "🔄 IP地址批量更新脚本 (macOS兼容版)"
echo "================================"
echo OLD_IP="10.30.15.93"
echo NEW_IP="10.30.15.93"
echo "================================"

OLD_IP="10.30.15.93"
NEW_IP="10.30.15.93"
# 计数器
TOTAL_FILES=0
UPDATED_FILES=0

echo "📋 1. 查找包含旧IP地址的文件..."

# 查找所有包含旧IP的文件（排除.git目录和二进制文件）
FILES=$(grep -r -l "$OLD_IP" . --exclude-dir=.git --exclude-dir=node_modules --exclude="*.jpg" --exclude="*.png" --exclude="*.gif" --exclude="*.mp4" --exclude="*.zip" --exclude="*.tar" --exclude="*.gz" 2>/dev/null || true)

if [ -z "$FILES" ]; then
    echo "✅ 没有找到包含旧IP地址的文件"
    exit 0
fi

echo "找到以下文件包含旧IP地址:"
for file in $FILES; do
    echo "  📄 $file"
    ((TOTAL_FILES++))
done

echo ""
echo "📋 2. 开始批量替换..."

# 批量替换IP地址
for file in $FILES; do
    # 检查文件是否存在且可写
    if [ -f "$file" ] && [ -w "$file" ]; then
        # 备份原文件
        cp "$file" "$file.backup.$(date +%Y%m%d_%H%M%S)"
        
        # 使用macOS兼容的sed替换IP地址
        if sed -i '' "s/$OLD_IP/$NEW_IP/g" "$file" 2>/dev/null; then
            echo "✅ 已更新: $file"
            ((UPDATED_FILES++))
        else
            echo "❌ 更新失败: $file"
            # 恢复备份文件
            cp "$file.backup."* "$file" 2>/dev/null || true
        fi
    else
        echo "⚠️  跳过: $file (文件不存在或不可写)"
    fi
done

echo ""
echo "📋 3. 验证更新结果..."

# 验证是否还有旧IP地址
REMAINING=$(grep -r -l "$OLD_IP" . --exclude-dir=.git --exclude-dir=node_modules --exclude="*.backup.*" 2>/dev/null || true)

if [ -z "$REMAINING" ]; then
    echo "✅ 所有文件已成功更新，没有发现残留的旧IP地址"
else
    echo "⚠️  以下文件仍包含旧IP地址:"
    for file in $REMAINING; do
        echo "  📄 $file"
    done
fi

echo ""
echo "📋 4. 更新统计..."
echo "总文件数: $TOTAL_FILES"
echo "已更新文件数: $UPDATED_FILES"
if [ $TOTAL_FILES -gt 0 ]; then
    echo "更新成功率: $((UPDATED_FILES * 100 / TOTAL_FILES))%"
else
    echo "更新成功率: 0%"
fi

echo ""
echo "📋 5. 备份文件清理..."
echo "创建的备份文件可以通过以下命令清理:"
echo "  find . -name \"*.backup.*\" -type f"
echo "  find . -name \"*.backup.*\" -type f -delete  # 删除备份文件"

echo ""
echo "📋 6. 验证新IP地址..."
NEW_IP_FILES=$(grep -r -l "$NEW_IP" . --exclude-dir=.git --exclude-dir=node_modules --exclude="*.backup.*" 2>/dev/null || true)

if [ -n "$NEW_IP_FILES" ]; then
    NEW_IP_COUNT=$(echo "$NEW_IP_FILES" | grep -c '^' || echo "0")
    echo "✅ 发现 $NEW_IP_COUNT 个文件包含新IP地址:"
    for file in $NEW_IP_FILES; do
        OCCURRENCES=$(grep -c "$NEW_IP" "$file" 2>/dev/null || echo "0")
        echo "  📄 $file ($OCCURRENCES 处)"
    done
else
    echo "⚠️  没有发现包含新IP地址的文件"
fi

echo ""
echo "================================"
echo "✅ IP地址批量更新完成！"
echo "🔄 从 $OLD_IP 更新到 $NEW_IP"
echo "📊 处理了 $TOTAL_FILES 个文件，成功更新 $UPDATED_FILES 个"
echo "================================" 