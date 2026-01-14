#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 检查开发环境...${NC}"
echo ""

# 检查 Docker
echo -e "${BLUE}[1/6] 检查 Docker${NC}"
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    echo -e "${GREEN}✅ $DOCKER_VERSION${NC}"
else
    echo -e "${RED}❌ Docker 未安装${NC}"
    echo "   安装: https://docs.docker.com/get-docker/"
fi

# 检查 Docker Compose
echo -e "${BLUE}[2/6] 检查 Docker Compose${NC}"
if command -v docker-compose &> /dev/null; then
    COMPOSE_VERSION=$(docker-compose --version)
    echo -e "${GREEN}✅ $COMPOSE_VERSION${NC}"
else
    echo -e "${RED}❌ Docker Compose 未安装${NC}"
fi

# 检查 Node.js
echo -e "${BLUE}[3/6] 检查 Node.js${NC}"
if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version)
    NPM_VERSION=$(npm --version)
    echo -e "${GREEN}✅ Node.js $NODE_VERSION${NC}"
    echo -e "${GREEN}✅ npm $NPM_VERSION${NC}"
else
    echo -e "${RED}❌ Node.js 未安装${NC}"
    echo "   安装: https://nodejs.org/"
fi

# 检查 Python
echo -e "${BLUE}[4/6] 检查 Python${NC}"
if command -v python3 &> /dev/null; then
    PYTHON_VERSION=$(python3 --version)
    PIP_VERSION=$(pip3 --version | awk '{print $2}')
    echo -e "${GREEN}✅ $PYTHON_VERSION${NC}"
    echo -e "${GREEN}✅ pip $PIP_VERSION${NC}"
else
    echo -e "${RED}❌ Python3 未安装${NC}"
    echo "   安装: https://www.python.org/downloads/"
fi

# 检查 Git
echo -e "${BLUE}[5/6] 检查 Git${NC}"
if command -v git &> /dev/null; then
    GIT_VERSION=$(git --version)
    echo -e "${GREEN}✅ $GIT_VERSION${NC}"
else
    echo -e "${RED}❌ Git 未安装${NC}"
fi

# 检查端口占用
echo -e "${BLUE}[6/6] 检查端口占用情况${NC}"
PORTS=(3000 3001 3002 3003 3004 3005 3006 5173 5432 6379 5672 15672 9000 9001)
PORT_NAMES=(
    "API Gateway"
    "Upload Service"
    "Parse Service"
    "Chunk Service"
    "PPT Gen Service"
    "File Service"
    "Auth Service"
    "Frontend"
    "PostgreSQL"
    "Redis"
    "RabbitMQ"
    "RabbitMQ Admin"
    "MinIO API"
    "MinIO Console"
)

for i in "${!PORTS[@]}"; do
    PORT=${PORTS[$i]}
    NAME=${PORT_NAMES[$i]}
    
    if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  端口 $PORT ($NAME) 已被占用${NC}"
    else
        echo -e "${GREEN}✅ 端口 $PORT ($NAME) 可用${NC}"
    fi
done

echo ""
echo -e "${BLUE}================================${NC}"
echo -e "${GREEN}✅ 环境检查完成！${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo "📝 下一步："
echo "   1. docker-compose up -d        # 启动基础设施"
echo "   2. ./init_infrastructure.sh    # 初始化配置"
echo "   3. ./start_all.sh              # 启动所有服务"


