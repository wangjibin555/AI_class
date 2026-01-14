#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}📦 安装项目依赖...${NC}"
echo ""

# 1. Gateway
echo -e "${BLUE}[1/6] 安装 API Gateway 依赖${NC}"
cd gateway
if [ -f "package.json" ]; then
    npm install
    echo -e "${GREEN}✅ Gateway 依赖安装完成${NC}"
else
    echo -e "${YELLOW}⚠️  package.json 未找到，跳过${NC}"
fi
cd ..

# 2. Frontend
echo -e "\n${BLUE}[2/6] 安装前端依赖${NC}"
cd frontend
if [ -f "package.json" ]; then
    npm install
    echo -e "${GREEN}✅ Frontend 依赖安装完成${NC}"
else
    echo -e "${YELLOW}⚠️  package.json 未找到，跳过${NC}"
fi
cd ..

# 3. Upload Service
echo -e "\n${BLUE}[3/6] 安装 Upload Service 依赖${NC}"
cd services/upload-service
if [ -f "package.json" ]; then
    npm install
    echo -e "${GREEN}✅ Upload Service 依赖安装完成${NC}"
else
    echo -e "${YELLOW}⚠️  package.json 未找到，跳过${NC}"
fi
cd ../..

# 4. Auth Service
echo -e "\n${BLUE}[4/6] 安装 Auth Service 依赖${NC}"
cd services/auth-service
if [ -f "package.json" ]; then
    npm install
    echo -e "${GREEN}✅ Auth Service 依赖安装完成${NC}"
else
    echo -e "${YELLOW}⚠️  package.json 未找到，跳过${NC}"
fi
cd ../..

# 5. Parse Service
echo -e "\n${BLUE}[5/6] 安装 Parse Service 依赖${NC}"
cd services/parse-service
if [ -f "requirements.txt" ]; then
    if [ ! -d "venv" ]; then
        echo "  创建Python虚拟环境..."
        python3 -m venv venv
    fi
    source venv/bin/activate
    pip install --upgrade pip
    pip install -r requirements.txt
    deactivate
    echo -e "${GREEN}✅ Parse Service 依赖安装完成${NC}"
else
    echo -e "${YELLOW}⚠️  requirements.txt 未找到，跳过${NC}"
fi
cd ../..

# 6. PPT Gen Service
echo -e "\n${BLUE}[6/6] 安装 PPT Gen Service 依赖${NC}"
cd services/ppt-gen-service
if [ -f "requirements.txt" ]; then
    if [ ! -d "venv" ]; then
        echo "  创建Python虚拟环境..."
        python3 -m venv venv
    fi
    source venv/bin/activate
    pip install --upgrade pip
    pip install -r requirements.txt
    deactivate
    echo -e "${GREEN}✅ PPT Gen Service 依赖安装完成${NC}"
else
    echo -e "${YELLOW}⚠️  requirements.txt 未找到，跳过${NC}"
fi
cd ../..

echo ""
echo -e "${BLUE}================================${NC}"
echo -e "${GREEN}✅ 所有依赖安装完成！${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo "📝 下一步："
echo "   ./scripts/start_all.sh         # 启动所有服务"


