#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🛑 停止所有服务...${NC}"
echo ""

# 停止Node.js和Python服务
echo -e "${BLUE}[1/2] 停止应用服务${NC}"

PORTS=(3000 3001 3002 3003 3004 3005 3006 5173)
SERVICE_NAMES=(
    "API Gateway"
    "Upload Service"
    "Parse Service"
    "Chunk Service"
    "PPT Gen Service"
    "File Service"
    "Auth Service"
    "Frontend"
)

for i in "${!PORTS[@]}"; do
    PORT=${PORTS[$i]}
    NAME=${SERVICE_NAMES[$i]}
    
    PID=$(lsof -ti:$PORT 2>/dev/null)
    if [ ! -z "$PID" ]; then
        kill -9 $PID 2>/dev/null
        echo -e "${GREEN}✅ 已停止 $NAME (端口 $PORT)${NC}"
    else
        echo -e "⏭️  $NAME (端口 $PORT) 未运行"
    fi
done

# 停止基础设施（可选）
echo ""
echo -e "${BLUE}[2/2] 基础设施管理${NC}"
read -p "是否停止基础设施容器（PostgreSQL, Redis等）？[y/N]: " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    docker-compose stop
    echo -e "${GREEN}✅ 基础设施已停止${NC}"
else
    echo -e "⏭️  保持基础设施运行"
fi

echo ""
echo -e "${GREEN}✅ 服务停止完成！${NC}"


