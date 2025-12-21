# 快速开始指南

## 环境准备

### 必要软件
- Docker & Docker Compose
- Node.js 18+ (用于前端和Gateway)
- Python 3.10+ (用于解析和生成服务)
- Git

### 可选软件
- Kubernetes (用于生产部署)
- Postman (API测试)

## 本地开发环境搭建

### 1. 启动基础设施

首先启动所有依赖的基础设施服务：

```bash
cd doc-to-ppt-system
docker-compose up -d
```

这将启动：
- PostgreSQL (端口 5432)
- Redis (端口 6379)
- RabbitMQ (端口 5672, 管理界面 15672)
- MinIO (端口 9000, 控制台 9001)

### 2. 验证基础设施

访问以下地址验证服务启动：

- RabbitMQ管理界面: http://localhost:15672
  - 用户名: admin
  - 密码: admin123

- MinIO控制台: http://localhost:9001
  - 用户名: admin
  - 密码: admin123

### 3. 配置MinIO

访问MinIO控制台，创建以下存储桶：
- `documents` - 存储上传的文档
- `presentations` - 存储生成的PPT

### 4. 安装依赖

#### 前端
```bash
cd frontend
npm install
npm run dev
```

#### API Gateway
```bash
cd gateway
npm install
npm run dev
```

#### 各个微服务

Node.js服务（Upload Service, File Service, Auth Service）:
```bash
cd services/upload-service
npm install
npm run dev
```

Python服务（Parse Service, Chunk Service, PPT Gen Service）:
```bash
cd services/parse-service
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt
uvicorn src.main:app --reload --port 3002
```

### 5. 配置环境变量

每个服务目录下创建 `.env` 文件，参考 `.env.example`。

示例配置：
```env
# 数据库
DB_HOST=localhost
DB_PORT=5432
DB_NAME=doc2ppt
DB_USER=admin
DB_PASSWORD=admin123

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# RabbitMQ
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=admin
RABBITMQ_PASSWORD=admin123

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=admin
MINIO_SECRET_KEY=admin123

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=7d
```

## 测试系统

### 1. 测试文档上传

```bash
curl -X POST http://localhost:3000/api/upload \
  -F "file=@test.docx" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 2. 查看任务状态

```bash
curl http://localhost:3000/api/tasks/{taskId} \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. 下载生成的PPT

```bash
curl http://localhost:3000/api/download/{taskId} \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o result.pptx
```

## 开发工具

### 推荐IDE插件

**VS Code:**
- ESLint
- Prettier
- Python
- Docker
- REST Client

### 日志查看

查看服务日志：
```bash
# Docker服务日志
docker-compose logs -f [service-name]

# 应用日志
# 各服务会将日志输出到控制台和日志文件
```

## 常见问题

### 1. 端口冲突
如果端口被占用，修改 `docker-compose.yml` 中的端口映射。

### 2. 权限问题
确保MinIO存储桶设置了正确的访问策略。

### 3. 消息队列连接失败
检查RabbitMQ是否正常运行，队列是否已创建。

## 下一步

- 查看 [API文档](../api/README.md)
- 了解 [架构设计](../architecture/system-overview.md)
- 学习 [开发规范](./development-guide.md)


