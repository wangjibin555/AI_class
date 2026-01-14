# 项目启动与操作指南

## 📋 目录
- [环境准备](#环境准备)
- [首次启动](#首次启动)
- [启动顺序](#启动顺序)
- [验证检查](#验证检查)
- [开发工作流](#开发工作流)
- [常见问题](#常见问题)
- [注意事项](#注意事项)
- [停止服务](#停止服务)

---

## 🔧 环境准备

### 必需软件版本

| 软件 | 最低版本 | 推荐版本 | 安装验证命令 |
|-----|---------|---------|------------|
| Docker | 20.10+ | 24.0+ | `docker --version` |
| Docker Compose | 2.0+ | 2.20+ | `docker-compose --version` |
| Node.js | 18.0+ | 20.0+ | `node --version` |
| npm | 9.0+ | 10.0+ | `npm --version` |
| Python | 3.10+ | 3.11+ | `python3 --version` |
| pip | 22.0+ | 23.0+ | `pip3 --version` |

### 安装检查脚本

```bash
# 创建环境检查脚本
cat > check_env.sh << 'EOF'
#!/bin/bash

echo "🔍 检查开发环境..."

# 检查 Docker
if command -v docker &> /dev/null; then
    echo "✅ Docker: $(docker --version)"
else
    echo "❌ Docker 未安装"
fi

# 检查 Docker Compose
if command -v docker-compose &> /dev/null; then
    echo "✅ Docker Compose: $(docker-compose --version)"
else
    echo "❌ Docker Compose 未安装"
fi

# 检查 Node.js
if command -v node &> /dev/null; then
    echo "✅ Node.js: $(node --version)"
else
    echo "❌ Node.js 未安装"
fi

# 检查 Python
if command -v python3 &> /dev/null; then
    echo "✅ Python: $(python3 --version)"
else
    echo "❌ Python 未安装"
fi

# 检查端口占用
echo ""
echo "🔍 检查端口占用情况..."
for port in 3000 3001 3002 3003 3004 3005 3006 5432 6379 5672 15672 9000 9001; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "⚠️  端口 $port 已被占用"
    else
        echo "✅ 端口 $port 可用"
    fi
done

echo ""
echo "✅ 环境检查完成！"
EOF

chmod +x check_env.sh
./check_env.sh
```

---

## 🚀 首次启动

### 第一步：克隆或进入项目

```bash
cd /Users/wepie/Desktop/AI-Class/doc-to-ppt-system
```

### 第二步：启动基础设施

```bash
# 启动 PostgreSQL、Redis、RabbitMQ、MinIO
docker-compose up -d

# 查看启动日志（确保所有服务正常启动）
docker-compose logs -f

# 按 Ctrl+C 退出日志查看
```

**⏱️ 等待时间：** 首次启动需要下载镜像，可能需要 5-10 分钟

### 第三步：验证基础设施

```bash
# 检查所有容器是否正常运行
docker-compose ps

# 应该看到 4 个容器都是 "Up" 状态：
# - doc2ppt-postgres
# - doc2ppt-redis
# - doc2ppt-rabbitmq
# - doc2ppt-minio
```

### 第四步：初始化MinIO存储桶

**方式一：通过Web控制台（推荐）**

1. 访问 MinIO 控制台：http://localhost:9001
2. 登录凭证：
   - 用户名：`admin`
   - 密码：`admin123`
3. 点击左侧 "Buckets"
4. 创建两个存储桶：
   - `documents` - 用于存储上传的文档
   - `presentations` - 用于存储生成的PPT
5. 设置存储桶访问权限为 `public`（开发环境）

**方式二：使用MinIO客户端**

```bash
# 安装 mc (MinIO Client)
brew install minio/stable/mc  # macOS
# 或
# apt-get install minio-client  # Ubuntu

# 配置别名
mc alias set local http://localhost:9000 admin admin123

# 创建存储桶
mc mb local/documents
mc mb local/presentations

# 设置公共访问权限（仅开发环境）
mc anonymous set public local/documents
mc anonymous set public local/presentations
```

### 第五步：初始化数据库

```bash
# 连接到 PostgreSQL
docker exec -it doc2ppt-postgres psql -U admin -d doc2ppt

# 在 psql 中执行以下 SQL（创建基础表）
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    filename VARCHAR(255) NOT NULL,
    file_type VARCHAR(10) NOT NULL,
    file_size BIGINT NOT NULL,
    storage_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    document_id UUID REFERENCES documents(id),
    status VARCHAR(20) DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    result_url TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_tasks_status ON tasks(status);

# 退出 psql
\q
```

### 第六步：配置环境变量

为每个服务创建 `.env` 文件：

**gateway/.env**
```bash
cat > gateway/.env << 'EOF'
PORT=3000
NODE_ENV=development

# JWT配置
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_EXPIRY=7d

# 服务地址
UPLOAD_SERVICE_URL=http://localhost:3001
PARSE_SERVICE_URL=http://localhost:3002
CHUNK_SERVICE_URL=http://localhost:3003
PPT_GEN_SERVICE_URL=http://localhost:3004
FILE_SERVICE_URL=http://localhost:3005
AUTH_SERVICE_URL=http://localhost:3006

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
EOF
```

**services/upload-service/.env**
```bash
cat > services/upload-service/.env << 'EOF'
PORT=3001
NODE_ENV=development

# MinIO配置
MINIO_ENDPOINT=localhost
MINIO_PORT=9000
MINIO_ACCESS_KEY=admin
MINIO_SECRET_KEY=admin123
MINIO_USE_SSL=false
MINIO_BUCKET_DOCUMENTS=documents

# RabbitMQ配置
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=admin
RABBITMQ_PASSWORD=admin123
RABBITMQ_QUEUE_PARSE=doc-parse-queue

# 文件上传限制
MAX_FILE_SIZE=52428800
ALLOWED_TYPES=docx,pdf,md,txt
EOF
```

**services/auth-service/.env**
```bash
cat > services/auth-service/.env << 'EOF'
PORT=3006
NODE_ENV=development

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=doc2ppt
DB_USER=admin
DB_PASSWORD=admin123

# JWT配置
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_EXPIRY=7d

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
EOF
```

**services/parse-service/.env**
```bash
cat > services/parse-service/.env << 'EOF'
PORT=3002
ENV=development

# RabbitMQ配置
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=admin
RABBITMQ_PASSWORD=admin123
RABBITMQ_QUEUE_PARSE=doc-parse-queue
RABBITMQ_QUEUE_CHUNK=doc-chunk-queue

# MinIO配置
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=admin
MINIO_SECRET_KEY=admin123
MINIO_USE_SSL=false
MINIO_BUCKET_DOCUMENTS=documents
EOF
```

### 第七步：安装服务依赖

**安装 Gateway**
```bash
cd gateway
npm install
cd ..
```

**安装 Upload Service**
```bash
cd services/upload-service
npm install
cd ../..
```

**安装 Auth Service**
```bash
cd services/auth-service
npm install
cd ../..
```

**安装 Parse Service**
```bash
cd services/parse-service
python3 -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt
deactivate
cd ../..
```

**安装 PPT Gen Service**
```bash
cd services/ppt-gen-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
deactivate
cd ../..
```

**安装前端**
```bash
cd frontend
npm install
cd ..
```

---

## 📊 启动顺序

### 正确的启动顺序（重要！）

```
1. 基础设施层（已启动）
   ├── PostgreSQL
   ├── Redis
   ├── RabbitMQ
   └── MinIO

2. 认证服务（优先启动）
   └── Auth Service (端口 3006)

3. 核心服务（并行启动）
   ├── Upload Service (端口 3001)
   ├── Parse Service (端口 3002)
   ├── Chunk Service (端口 3003)
   ├── PPT Gen Service (端口 3004)
   └── File Service (端口 3005)

4. 网关层
   └── API Gateway (端口 3000)

5. 前端应用
   └── Frontend (端口 5173)
```

### 启动脚本

创建启动脚本便于管理：

```bash
cat > start_all.sh << 'EOF'
#!/bin/bash

echo "🚀 启动文档转PPT系统..."

# 1. 检查基础设施
echo "📊 检查基础设施状态..."
docker-compose ps

# 2. 启动认证服务
echo ""
echo "🔐 启动认证服务..."
cd services/auth-service
npm run dev &
AUTH_PID=$!
sleep 3
cd ../..

# 3. 启动上传服务
echo "📤 启动上传服务..."
cd services/upload-service
npm run dev &
UPLOAD_PID=$!
sleep 2
cd ../..

# 4. 启动解析服务
echo "📝 启动解析服务..."
cd services/parse-service
source venv/bin/activate
uvicorn src.main:app --reload --port 3002 &
PARSE_PID=$!
deactivate
sleep 2
cd ../..

# 5. 启动PPT生成服务
echo "🎨 启动PPT生成服务..."
cd services/ppt-gen-service
source venv/bin/activate
uvicorn src.main:app --reload --port 3004 &
PPT_PID=$!
deactivate
sleep 2
cd ../..

# 6. 启动网关
echo "🚪 启动API网关..."
cd gateway
npm run dev &
GATEWAY_PID=$!
sleep 3
cd ..

# 7. 启动前端
echo "🌐 启动前端应用..."
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

echo ""
echo "✅ 所有服务已启动！"
echo ""
echo "📋 服务地址："
echo "   - 前端: http://localhost:5173"
echo "   - API网关: http://localhost:3000"
echo "   - RabbitMQ管理: http://localhost:15672"
echo "   - MinIO控制台: http://localhost:9001"
echo ""
echo "⚠️  按 Ctrl+C 停止所有服务"

# 保存 PID 用于清理
echo "$AUTH_PID $UPLOAD_PID $PARSE_PID $PPT_PID $GATEWAY_PID $FRONTEND_PID" > .pids

wait
EOF

chmod +x start_all.sh
```

---

## ✅ 验证检查

### 检查清单

```bash
# 1. 检查基础设施容器
docker-compose ps
# 应该看到 4 个容器都是 "Up" 状态

# 2. 检查服务端口
netstat -an | grep LISTEN | grep -E '3000|3001|3002|3003|3004|3005|3006'

# 3. 测试 MinIO 连接
curl http://localhost:9000/minio/health/live

# 4. 测试 RabbitMQ 连接
curl -u admin:admin123 http://localhost:15672/api/overview

# 5. 测试 Redis 连接
redis-cli -h localhost -p 6379 ping
# 应该返回 PONG

# 6. 测试 PostgreSQL 连接
docker exec doc2ppt-postgres pg_isready -U admin
```

### 健康检查脚本

```bash
cat > health_check.sh << 'EOF'
#!/bin/bash

echo "🏥 系统健康检查..."

# 基础设施检查
echo ""
echo "📊 基础设施："

# PostgreSQL
if docker exec doc2ppt-postgres pg_isready -U admin > /dev/null 2>&1; then
    echo "✅ PostgreSQL: 健康"
else
    echo "❌ PostgreSQL: 异常"
fi

# Redis
if redis-cli -h localhost -p 6379 ping > /dev/null 2>&1; then
    echo "✅ Redis: 健康"
else
    echo "❌ Redis: 异常"
fi

# RabbitMQ
if curl -s -u admin:admin123 http://localhost:15672/api/overview > /dev/null 2>&1; then
    echo "✅ RabbitMQ: 健康"
else
    echo "❌ RabbitMQ: 异常"
fi

# MinIO
if curl -s http://localhost:9000/minio/health/live > /dev/null 2>&1; then
    echo "✅ MinIO: 健康"
else
    echo "❌ MinIO: 异常"
fi

# 服务检查
echo ""
echo "🔧 微服务："

for port in 3000 3001 3006; do
    service_name=""
    case $port in
        3000) service_name="API Gateway" ;;
        3001) service_name="Upload Service" ;;
        3006) service_name="Auth Service" ;;
    esac
    
    if nc -z localhost $port 2>/dev/null; then
        echo "✅ $service_name (端口 $port): 运行中"
    else
        echo "⚠️  $service_name (端口 $port): 未运行"
    fi
done

echo ""
echo "✅ 健康检查完成！"
EOF

chmod +x health_check.sh
```

---

## 💻 开发工作流

### 推荐的开发流程

1. **启动基础设施**（只需启动一次）
   ```bash
   docker-compose up -d
   ```

2. **开发单个服务**
   ```bash
   # 示例：开发上传服务
   cd services/upload-service
   npm run dev
   ```

3. **查看实时日志**
   ```bash
   # 基础设施日志
   docker-compose logs -f [服务名]
   
   # 示例：查看 RabbitMQ 日志
   docker-compose logs -f rabbitmq
   ```

4. **重启单个服务**
   ```bash
   # 找到服务进程
   lsof -i :3001
   
   # 杀死进程
   kill -9 [PID]
   
   # 重新启动
   npm run dev
   ```

### 使用 tmux/screen 管理多个服务

```bash
# 安装 tmux
brew install tmux  # macOS
# 或
# apt-get install tmux  # Ubuntu

# 创建会话管理脚本
cat > dev_session.sh << 'EOF'
#!/bin/bash

SESSION="doc2ppt"

tmux new-session -d -s $SESSION

# 窗口0: Gateway
tmux rename-window -t $SESSION:0 'Gateway'
tmux send-keys -t $SESSION:0 'cd gateway && npm run dev' C-m

# 窗口1: Upload Service
tmux new-window -t $SESSION:1 -n 'Upload'
tmux send-keys -t $SESSION:1 'cd services/upload-service && npm run dev' C-m

# 窗口2: Auth Service
tmux new-window -t $SESSION:2 -n 'Auth'
tmux send-keys -t $SESSION:2 'cd services/auth-service && npm run dev' C-m

# 窗口3: Parse Service
tmux new-window -t $SESSION:3 -n 'Parse'
tmux send-keys -t $SESSION:3 'cd services/parse-service && source venv/bin/activate && uvicorn src.main:app --reload --port 3002' C-m

# 窗口4: Frontend
tmux new-window -t $SESSION:4 -n 'Frontend'
tmux send-keys -t $SESSION:4 'cd frontend && npm run dev' C-m

# 窗口5: Logs
tmux new-window -t $SESSION:5 -n 'Logs'
tmux send-keys -t $SESSION:5 'docker-compose logs -f' C-m

tmux attach-session -t $SESSION
EOF

chmod +x dev_session.sh
```

---

## ⚠️ 注意事项

### 🔴 关键注意事项

1. **启动顺序很重要**
   - ⚠️ 必须先启动基础设施，再启动服务
   - ⚠️ 认证服务应该优先于其他业务服务启动
   - ⚠️ 网关应该最后启动（在所有服务之后）

2. **端口冲突**
   - ⚠️ 启动前检查端口是否被占用
   - ⚠️ 如有冲突，修改配置文件中的端口号

3. **环境变量**
   - ⚠️ 每个服务必须配置对应的 `.env` 文件
   - ⚠️ 不要将 `.env` 文件提交到 Git
   - ⚠️ 生产环境必须更换所有密钥和密码

4. **数据持久化**
   - ✅ Docker volumes 会保留数据
   - ⚠️ `docker-compose down -v` 会删除所有数据
   - ⚠️ 仅清理容器使用 `docker-compose down`

5. **Python虚拟环境**
   - ⚠️ 每次运行Python服务前必须激活虚拟环境
   - ⚠️ 记得先 `source venv/bin/activate`

6. **内存资源**
   - ⚠️ Docker至少分配 4GB 内存
   - ⚠️ 推荐 8GB 或更多用于流畅开发

### 🟡 最佳实践

1. **开发环境隔离**
   ```bash
   # 为每个服务使用独立的终端窗口
   # 便于查看实时日志和错误信息
   ```

2. **日志管理**
   ```bash
   # 定期清理Docker日志
   docker system prune -a
   ```

3. **依赖更新**
   ```bash
   # 定期更新依赖
   npm outdated  # Node.js项目
   pip list --outdated  # Python项目
   ```

4. **代码质量**
   ```bash
   # 提交前运行检查
   npm run lint  # ESLint
   npm run test  # 单元测试
   ```

### 🔵 安全建议

1. **开发环境**
   - ✅ 使用简单密码方便开发
   - ⚠️ 不要在公网暴露开发环境

2. **生产环境**
   - 🔴 必须更换所有默认密码
   - 🔴 使用强密码和密钥
   - 🔴 启用 HTTPS
   - 🔴 配置防火墙规则
   - 🔴 定期备份数据库

---

## 🛑 停止服务

### 方式一：停止单个服务

```bash
# 查找进程
lsof -i :[端口号]

# 杀死进程
kill -9 [PID]
```

### 方式二：停止所有Node.js/Python服务

```bash
# 停止所有node进程（谨慎使用）
pkill -f "node.*dev"

# 停止所有uvicorn进程
pkill -f "uvicorn"
```

### 方式三：使用停止脚本

```bash
cat > stop_all.sh << 'EOF'
#!/bin/bash

echo "🛑 停止所有服务..."

# 读取保存的 PID
if [ -f .pids ]; then
    PIDS=$(cat .pids)
    for pid in $PIDS; do
        if ps -p $pid > /dev/null 2>&1; then
            kill -9 $pid
            echo "✅ 已停止进程: $pid"
        fi
    done
    rm .pids
else
    echo "⚠️  未找到 PID 文件，尝试通过端口停止..."
    
    for port in 3000 3001 3002 3003 3004 3005 3006 5173; do
        PID=$(lsof -ti:$port)
        if [ ! -z "$PID" ]; then
            kill -9 $PID
            echo "✅ 已停止端口 $port 的服务"
        fi
    done
fi

echo "✅ 所有服务已停止"
EOF

chmod +x stop_all.sh
```

### 方式四：停止基础设施

```bash
# 停止容器但保留数据
docker-compose stop

# 停止并删除容器（保留数据）
docker-compose down

# 停止并删除容器和数据（危险！）
docker-compose down -v

# 完全清理（包括镜像）
docker-compose down --rmi all -v
```

---

## 🔍 常见问题

### Q1: 端口已被占用

**症状：** `Error: listen EADDRINUSE: address already in use :::3000`

**解决：**
```bash
# 查找占用端口的进程
lsof -i :3000

# 杀死进程
kill -9 [PID]

# 或更改配置文件中的端口
```

### Q2: Docker容器无法启动

**症状：** `docker-compose up -d` 后容器状态为 "Exit"

**解决：**
```bash
# 查看详细日志
docker-compose logs [服务名]

# 重新构建
docker-compose down
docker-compose up -d --build

# 检查Docker内存分配
```

### Q3: MinIO无法创建存储桶

**症状：** 访问 MinIO 控制台报错

**解决：**
```bash
# 重启MinIO容器
docker-compose restart minio

# 检查容器日志
docker-compose logs minio

# 确认端口9000和9001未被占用
```

### Q4: RabbitMQ连接失败

**症状：** 服务日志显示 `Connection refused`

**解决：**
```bash
# 检查RabbitMQ状态
docker-compose ps rabbitmq

# 等待RabbitMQ完全启动（可能需要30秒）
docker-compose logs -f rabbitmq

# 访问管理界面验证
open http://localhost:15672
```

### Q5: Python虚拟环境问题

**症状：** `ModuleNotFoundError: No module named 'fastapi'`

**解决：**
```bash
cd services/parse-service

# 确认虚拟环境已激活
source venv/bin/activate

# 重新安装依赖
pip install -r requirements.txt

# 验证安装
pip list
```

### Q6: 数据库连接失败

**症状：** `connection to server at "localhost" failed`

**解决：**
```bash
# 检查PostgreSQL容器
docker-compose ps postgres

# 测试连接
docker exec -it doc2ppt-postgres psql -U admin -d doc2ppt

# 检查.env文件配置是否正确
```

---

## 📈 性能优化建议

1. **Docker资源配置**
   - 分配 8GB+ 内存
   - 启用多核 CPU

2. **开发时只启动需要的服务**
   ```bash
   # 只启动基础设施
   docker-compose up -d
   
   # 只开发某个服务
   cd services/upload-service
   npm run dev
   ```

3. **使用缓存加速构建**
   - npm: 使用 `npm ci` 而非 `npm install`
   - Docker: 优化 Dockerfile 层缓存

---

## 📞 获取帮助

- 查看日志：`docker-compose logs -f [服务名]`
- 健康检查：`./health_check.sh`
- 查看文档：`docs/` 目录
- 问题反馈：提交 Issue

---

**✅ 准备就绪！开始你的开发之旅吧！** 🚀


