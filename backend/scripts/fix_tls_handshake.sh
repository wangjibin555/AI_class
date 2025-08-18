#!/bin/bash

# TLS握手错误诊断与修复脚本
# 用于解决 "tls: unknown certificate" 和协议不匹配错误

echo "🔍 TLS握手错误诊断与修复工具"
echo "========================================"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查是否在正确目录
if [ ! -d "certs" ]; then
    echo -e "${RED}❌ 错误：未找到 certs 目录，请在项目根目录执行此脚本${NC}"
    exit 1
fi

echo "📊 第1步：服务器信息收集"
echo "----------------------------------------"

# 获取服务器公网IP
echo -n "🌐 获取服务器公网IP..."
SERVER_IP=$(curl -s ifconfig.me)
if [ $? -eq 0 ] && [ -n "$SERVER_IP" ]; then
    echo -e " ${GREEN}✅ $SERVER_IP${NC}"
else
    echo -e " ${RED}❌ 获取失败${NC}"
    echo "尝试备用方法..."
    SERVER_IP=$(curl -s ipinfo.io/ip)
    if [ $? -eq 0 ] && [ -n "$SERVER_IP" ]; then
        echo -e "🌐 服务器公网IP: ${GREEN}$SERVER_IP${NC}"
    else
        echo -e "${RED}❌ 无法获取服务器公网IP，请手动输入：${NC}"
        read -p "请输入服务器公网IP: " SERVER_IP
    fi
fi

echo ""
echo "🔐 第2步：当前SSL证书分析"
echo "----------------------------------------"

# 检查证书文件是否存在
if [ ! -f "certs/cert.pem" ]; then
    echo -e "${RED}❌ 证书文件不存在: certs/cert.pem${NC}"
    NEED_NEW_CERT=true
else
    echo -e "📄 检查当前证书内容..."
    
    # 检查证书中的Subject Alternative Name
    echo "🔍 证书中的域名和IP:"
    SAN_INFO=$(openssl x509 -in certs/cert.pem -text -noout | grep -A 5 "Subject Alternative Name")
    
    if [ $? -eq 0 ]; then
        echo -e "${BLUE}$SAN_INFO${NC}"
        
        # 检查当前服务器IP是否在证书中
        if echo "$SAN_INFO" | grep -q "$SERVER_IP"; then
            echo -e "${GREEN}✅ 服务器IP ($SERVER_IP) 已包含在证书中${NC}"
            SERVER_IP_IN_CERT=true
        else
            echo -e "${YELLOW}⚠️  服务器IP ($SERVER_IP) 未包含在证书中${NC}"
            SERVER_IP_IN_CERT=false
        fi
        
        # 检查域名是否在证书中
        if echo "$SAN_INFO" | grep -q "wangjibin-sc.wepie.com"; then
            echo -e "${GREEN}✅ 域名 (wangjibin-sc.wepie.com) 已包含在证书中${NC}"
            DOMAIN_IN_CERT=true
        else
            echo -e "${YELLOW}⚠️  域名 (wangjibin-sc.wepie.com) 未包含在证书中${NC}"
            DOMAIN_IN_CERT=false
        fi
    else
        echo -e "${RED}❌ 无法读取证书的SAN信息${NC}"
        NEED_NEW_CERT=true
    fi
    
    # 检查证书有效期
    echo ""
    echo "📅 证书有效期检查:"
    CERT_DATES=$(openssl x509 -in certs/cert.pem -dates -noout)
    echo -e "${BLUE}$CERT_DATES${NC}"
    
    # 检查证书是否过期
    if openssl x509 -in certs/cert.pem -checkend 0 > /dev/null; then
        echo -e "${GREEN}✅ 证书未过期${NC}"
    else
        echo -e "${RED}❌ 证书已过期${NC}"
        NEED_NEW_CERT=true
    fi
fi

echo ""
echo "🌍 第3步：DNS解析检查"
echo "----------------------------------------"

echo -n "🔍 检查域名DNS解析..."
DNS_IP=$(nslookup wangjibin-sc.wepie.com | grep "Address:" | tail -1 | awk '{print $2}')

if [ -n "$DNS_IP" ]; then
    echo -e " ${GREEN}✅ $DNS_IP${NC}"
    
    if [ "$DNS_IP" = "$SERVER_IP" ]; then
        echo -e "${GREEN}✅ DNS解析正确 (域名指向服务器IP)${NC}"
        DNS_CORRECT=true
    else
        echo -e "${YELLOW}⚠️  DNS解析不匹配${NC}"
        echo -e "   域名解析IP: ${BLUE}$DNS_IP${NC}"
        echo -e "   服务器实际IP: ${BLUE}$SERVER_IP${NC}"
        DNS_CORRECT=false
    fi
else
    echo -e " ${RED}❌ DNS解析失败${NC}"
    DNS_CORRECT=false
fi

echo ""
echo "📋 第4步：问题诊断结果"
echo "----------------------------------------"

NEED_NEW_CERT=false

# 判断是否需要重新生成证书
if [ ! -f "certs/cert.pem" ]; then
    echo -e "${RED}❌ 证书文件不存在${NC}"
    NEED_NEW_CERT=true
elif [ "$SERVER_IP_IN_CERT" = false ]; then
    echo -e "${YELLOW}⚠️  证书缺少服务器IP ($SERVER_IP)${NC}"
    NEED_NEW_CERT=true
elif [ "$DOMAIN_IN_CERT" = false ]; then
    echo -e "${YELLOW}⚠️  证书缺少域名 (wangjibin-sc.wepie.com)${NC}"
    NEED_NEW_CERT=true
fi

if [ "$DNS_CORRECT" = false ]; then
    echo -e "${YELLOW}⚠️  DNS解析问题需要修复${NC}"
fi

# 如果需要重新生成证书
if [ "$NEED_NEW_CERT" = true ]; then
    echo ""
    echo "🔧 第5步：重新生成SSL证书"
    echo "----------------------------------------"
    
    # 备份旧证书
    if [ -f "certs/cert.pem" ]; then
        echo "📦 备份旧证书..."
        cp certs/cert.pem certs/cert.pem.backup.$(date +%Y%m%d_%H%M%S)
        cp certs/key.pem certs/key.pem.backup.$(date +%Y%m%d_%H%M%S)
        echo -e "${GREEN}✅ 旧证书已备份${NC}"
    fi
    
    echo "🔐 生成新的SSL证书..."
    echo "   包含域名: wangjibin-sc.wepie.com"
    echo "   包含IP: $SERVER_IP, 127.0.0.1"
    
    # 生成新证书
    openssl req -x509 -newkey rsa:4096 \
        -keyout certs/key.pem \
        -out certs/cert.pem \
        -days 365 -nodes \
        -subj "/C=CN/ST=Beijing/L=Beijing/O=AI-Classroom/CN=wangjibin-sc.wepie.com" \
        -addext "subjectAltName=DNS:wangjibin-sc.wepie.com,DNS:localhost,DNS:127.0.0.1,IP:127.0.0.1,IP:$SERVER_IP"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ 新证书生成成功${NC}"
        
        # 验证新证书
        echo ""
        echo "🔍 验证新证书内容:"
        NEW_SAN=$(openssl x509 -in certs/cert.pem -text -noout | grep -A 5 "Subject Alternative Name")
        echo -e "${BLUE}$NEW_SAN${NC}"
        
        # 检查权限
        chmod 600 certs/key.pem
        chmod 644 certs/cert.pem
        echo -e "${GREEN}✅ 证书权限设置完成${NC}"
        
    else
        echo -e "${RED}❌ 证书生成失败${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✅ 当前证书配置正确，无需重新生成${NC}"
fi

echo ""
echo "🚀 第6步：重启建议"
echo "----------------------------------------"

echo -e "${YELLOW}⚠️  重要提醒：${NC}"
echo "1. 重新生成证书后，需要重启后端服务"
echo "2. 重启命令: sudo systemctl restart your-service 或手动重启"
echo "3. 确保防火墙开放9000端口: sudo ufw allow 9000"

echo ""
echo "🔍 第7步：验证检查清单"
echo "----------------------------------------"

echo "请重启服务后，验证以下内容："
echo ""
echo "1. 🌐 使用域名访问测试:"
echo "   curl -k https://wangjibin-sc.wepie.com:9000/api/v1/health"
echo ""
echo "2. 🌐 使用IP访问测试:"
echo "   curl -k https://$SERVER_IP:9000/api/v1/health"
echo ""
echo "3. 📱 前端连接测试:"
echo "   检查微信小程序是否能正常登录"
echo ""
echo "4. 📋 查看服务器日志:"
echo "   tail -f your-service.log | grep TLS"

echo ""
echo "💡 常见问题解决："
echo "----------------------------------------"
echo "• 如果仍有 'unknown certificate' 错误："
echo "  - 检查客户端是否使用了正确的域名"
echo "  - 确认DNS解析是否生效 (可能需要等待)"
echo ""
echo "• 如果有 'HTTP request to HTTPS server' 错误："
echo "  - 检查客户端配置，确保使用 https:// 协议"
echo "  - 确认前端配置文件使用正确的HTTPS地址"

echo ""
echo -e "${GREEN}🎉 TLS诊断与修复脚本执行完成！${NC}"
echo -e "${BLUE}💡 建议保存此脚本，以便将来使用${NC}"

# 显示当前配置摘要
echo ""
echo "📋 当前配置摘要："
echo "----------------------------------------"
echo "服务器IP: $SERVER_IP"
echo "域名: wangjibin-sc.wepie.com"
echo "HTTPS端口: 9000"
echo "证书位置: certs/cert.pem"
echo "私钥位置: certs/key.pem"