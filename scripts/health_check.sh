#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🏥 系统健康检查...${NC}"
echo ""

# 检查基础设施
echo -e "${BLUE}📊 基础设施状态：${NC}"

# PostgreSQL
echo -n "PostgreSQL: "
if docker exec doc2ppt-postgres pg_isready -U admin > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 健康${NC}"
else
    echo -e "${RED}❌ 异常${NC}"
fi

# Redis
echo -n "Redis: "
if docker exec doc2ppt-redis redis-cli ping > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 健康${NC}"
else
    echo -e "${RED}❌ 异常${NC}"
fi

# RabbitMQ
echo -n "RabbitMQ: "
if curl -s -u admin:admin123 http://localhost:15672/api/overview > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 健康${NC}"
else
    echo -e "${RED}❌ 异常${NC}"
fi

# MinIO
echo -n "MinIO: "
if curl -s http://localhost:9000/minio/health/live > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 健康${NC}"
else
    echo -e "${RED}❌ 异常${NC}"
fi

# 检查微服务
echo ""
echo -e "${BLUE}🔧 微服务状态：${NC}"

check_service() {
    local name=$1
    local port=$2
    
    echo -n "$name (端口 $port): "
    if nc -z localhost $port 2>/dev/null; then
        echo -e "${GREEN}✅ 运行中${NC}"
    else
        echo -e "${RED}❌ 未运行${NC}"
    fi
}

check_service "API Gateway" 3000
check_service "Upload Service" 3001
check_service "Parse Service" 3002
check_service "Chunk Service" 3003
check_service "PPT Gen Service" 3004
check_service "File Service" 3005
check_service "Auth Service" 3006
check_service "Frontend" 5173

# 数据库连接测试
echo ""
echo -e "${BLUE}💾 数据库连接测试：${NC}"
TABLES=$(docker exec doc2ppt-postgres psql -U admin -d doc2ppt -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | xargs)
if [ -n "$TABLES" ]; then
    echo -e "${GREEN}✅ 数据库连接成功，表数量: $TABLES${NC}"
else
    echo -e "${RED}❌ 数据库连接失败${NC}"
fi

# MinIO存储桶检查
echo ""
echo -e "${BLUE}🗂️  MinIO存储桶：${NC}"
if command -v mc &> /dev/null; then
    mc alias set local http://localhost:9000 admin admin123 2>/dev/null
    mc ls local/ 2>/dev/null | while read -r line; do
        echo "  $line"
    done
else
    echo -e "${YELLOW}⚠️  MinIO Client 未安装，跳过检查${NC}"
fi

# RabbitMQ队列检查
echo ""
echo -e "${BLUE}📬 RabbitMQ队列：${NC}"
docker exec doc2ppt-rabbitmq rabbitmqctl list_queues 2>/dev/null | while read -r line; do
    echo "  $line"
done

# 资源使用情况
echo ""
echo -e "${BLUE}💻 资源使用情况：${NC}"
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep doc2ppt

echo ""
echo -e "${BLUE}================================${NC}"
echo -e "${GREEN}✅ 健康检查完成！${NC}"
echo -e "${BLUE}================================${NC}"


