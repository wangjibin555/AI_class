# 文档转PPT系统

## 项目简介

本项目是一个基于微服务架构的文档转PPT系统，支持将各种格式的文档（Word、PDF、Markdown等）自动转换为精美的PPT演示文稿。

## 系统架构

采用微服务架构，主要包含以下组件：

- **前端 (Frontend)**: 提供用户交互界面
- **API网关 (Gateway)**: 统一入口，负责路由和认证
- **微服务层**:
  - 文档上传服务 (Upload Service)
  - 内容解析服务 (Parse Service)
  - 内容分块服务 (Chunk Service)
  - PPT生成服务 (PPT Generation Service)
  - 文件存储服务 (File Service)
  - 认证授权服务 (Auth Service)
- **基础设施层**:
  - 消息队列 (RabbitMQ)
  - 缓存 (Redis)
  - 对象存储 (MinIO)
  - 数据库 (PostgreSQL)

## 项目结构

```
doc-to-ppt-system/
├── frontend/                    # 前端应用
│   ├── public/                  # 静态资源
│   └── src/                     # 源代码
│       ├── components/          # React组件
│       ├── pages/              # 页面
│       ├── services/           # API调用
│       └── utils/              # 工具函数
│
├── gateway/                     # API网关
│   ├── src/
│   │   ├── routes/             # 路由配置
│   │   ├── middlewares/        # 中间件
│   │   └── config/             # 配置
│   └── tests/                  # 测试
│
├── services/                    # 微服务
│   ├── upload-service/         # 文档上传服务
│   ├── parse-service/          # 内容解析服务
│   ├── chunk-service/          # 内容分块服务
│   ├── ppt-gen-service/        # PPT生成服务
│   ├── file-service/           # 文件存储服务
│   └── auth-service/           # 认证授权服务
│   
│   每个服务包含:
│   ├── src/
│   │   ├── controllers/        # 控制器
│   │   ├── services/          # 业务逻辑
│   │   ├── models/            # 数据模型
│   │   ├── routes/            # 路由
│   │   ├── config/            # 配置
│   │   └── utils/             # 工具
│   └── tests/                 # 测试
│
├── infrastructure/              # 基础设施配置
│   ├── rabbitmq/               # 消息队列配置
│   ├── redis/                  # 缓存配置
│   ├── minio/                  # 对象存储配置
│   └── postgres/               # 数据库配置
│
├── shared/                      # 共享代码
│   ├── types/                  # 类型定义
│   ├── utils/                  # 公共工具
│   └── config/                 # 公共配置
│
├── deployments/                 # 部署配置
│   ├── docker/                 # Docker配置
│   └── k8s/                    # Kubernetes配置
│
└── docs/                        # 文档
    ├── api/                    # API文档
    ├── architecture/           # 架构文档
    └── guides/                 # 使用指南
```

## 技术栈

### 前端
- React / Vue.js
- TypeScript
- Axios
- Ant Design / Material-UI

### 后端
- Node.js / Python / Go (可根据服务选择)
- Express / FastAPI / Gin
- TypeScript / Python / Go

### 基础设施
- RabbitMQ - 消息队列
- Redis - 缓存
- MinIO - 对象存储
- PostgreSQL - 数据库
- Docker - 容器化
- Kubernetes - 容器编排

## 📚 文档导航

| 文档 | 说明 | 适合人群 |
|-----|------|---------|
| **[GETTING_STARTED.md](./GETTING_STARTED.md)** | 🚀 项目启动与操作总览（⭐从这里开始） | 所有人 |
| **[QUICK_REFERENCE.md](./QUICK_REFERENCE.md)** | ⚡ 快速参考手册 | 日常开发 |
| **[STARTUP_GUIDE.md](./STARTUP_GUIDE.md)** | 📖 详细启动指南 | 首次部署 |
| **[OPERATIONS.md](./OPERATIONS.md)** | 🔧 运维操作手册 | 运维人员 |
| **[PROJECT_STRUCTURE.md](./PROJECT_STRUCTURE.md)** | 🏗️ 项目结构说明 | 新成员 |

## 🚀 快速开始

### 环境要求
- ✅ Docker 20.10+ & Docker Compose 2.0+
- ✅ Node.js 18+
- ✅ Python 3.10+
- ✅ Git

### 5分钟启动

```bash
# 1. 进入项目目录
cd doc-to-ppt-system

# 2. 检查环境
./scripts/check_env.sh

# 3. 启动基础设施
docker-compose up -d

# 4. 初始化配置
./scripts/init_infrastructure.sh

# 5. 安装依赖（首次运行）
./scripts/install_deps.sh

# 6. 健康检查
./scripts/health_check.sh

# ✅ 开始开发！
```

### 访问地址

- 🌐 **前端**: http://localhost:5173
- 🚪 **API网关**: http://localhost:3000
- 🐰 **RabbitMQ管理**: http://localhost:15672 (admin/admin123)
- 🗄️ **MinIO控制台**: http://localhost:9001 (admin/admin123)

## 开发指南

详见 `docs/guides/` 目录下的开发文档。

## API文档

详见 `docs/api/` 目录或访问在线API文档。

## 贡献指南

欢迎贡献代码、报告问题和提出建议。

## 许可证

MIT License

