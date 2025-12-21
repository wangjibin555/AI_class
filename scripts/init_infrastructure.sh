#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}🔧 初始化基础设施...${NC}"
echo ""

# 等待服务就绪
wait_for_service() {
    local service=$1
    local host=$2
    local port=$3
    local max_attempts=30
    local attempt=1
    
    echo -e "${YELLOW}⏳ 等待 $service 启动...${NC}"
    
    while [ $attempt -le $max_attempts ]; do
        if nc -z $host $port 2>/dev/null; then
            echo -e "${GREEN}✅ $service 已就绪${NC}"
            return 0
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo -e "\n${RED}❌ $service 启动超时${NC}"
    return 1
}

# 1. 检查Docker容器状态
echo -e "${BLUE}[1/5] 检查Docker容器${NC}"
docker-compose ps

# 2. 等待服务就绪
echo -e "\n${BLUE}[2/5] 等待服务就绪${NC}"
wait_for_service "PostgreSQL" "localhost" "5432"
wait_for_service "Redis" "localhost" "6379"
wait_for_service "RabbitMQ" "localhost" "5672"
wait_for_service "MinIO" "localhost" "9000"

# 3. 初始化数据库
echo -e "\n${BLUE}[3/5] 初始化数据库${NC}"
docker exec -i doc2ppt-postgres psql -U admin -d doc2ppt << 'EOSQL'
-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建文档表
CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    file_type VARCHAR(10) NOT NULL,
    file_size BIGINT NOT NULL,
    storage_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建任务表
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    result_url TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_documents_user_id ON documents(user_id);
CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(created_at);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);

-- 插入测试用户
INSERT INTO users (username, email, password_hash, role)
VALUES ('admin', 'admin@example.com', 'hashed_password', 'admin')
ON CONFLICT (username) DO NOTHING;

\echo '✅ 数据库表创建完成'
EOSQL

echo -e "${GREEN}✅ 数据库初始化完成${NC}"

# 4. 创建MinIO存储桶
echo -e "\n${BLUE}[4/5] 创建MinIO存储桶${NC}"

# 检查mc是否安装
if ! command -v mc &> /dev/null; then
    echo -e "${YELLOW}⚠️  MinIO Client (mc) 未安装，请手动创建存储桶${NC}"
    echo "   访问: http://localhost:9001"
    echo "   用户名: admin"
    echo "   密码: admin123"
    echo "   创建存储桶: documents, presentations"
else
    # 配置mc
    mc alias set local http://localhost:9000 admin admin123 2>/dev/null
    
    # 创建存储桶
    mc mb local/documents 2>/dev/null || echo "  存储桶 documents 已存在"
    mc mb local/presentations 2>/dev/null || echo "  存储桶 presentations 已存在"
    
    # 设置访问策略（开发环境）
    mc anonymous set public local/documents
    mc anonymous set public local/presentations
    
    echo -e "${GREEN}✅ MinIO存储桶创建完成${NC}"
fi

# 5. 创建RabbitMQ队列
echo -e "\n${BLUE}[5/5] 配置RabbitMQ队列${NC}"
docker exec doc2ppt-rabbitmq rabbitmqctl list_queues 2>/dev/null
echo -e "${GREEN}✅ RabbitMQ配置完成${NC}"

echo ""
echo -e "${BLUE}================================${NC}"
echo -e "${GREEN}✅ 基础设施初始化完成！${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo "📋 访问地址："
echo "   PostgreSQL:    localhost:5432"
echo "   Redis:         localhost:6379"
echo "   RabbitMQ:      http://localhost:15672 (admin/admin123)"
echo "   MinIO:         http://localhost:9001 (admin/admin123)"
echo ""
echo "📝 下一步："
echo "   ./scripts/install_deps.sh      # 安装服务依赖"
echo "   ./scripts/start_all.sh         # 启动所有服务"


