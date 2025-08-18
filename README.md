# AI课堂 - 智能课件生成系统

## 项目简介

AI课堂是一个基于AI技术的智能课件生成系统，支持从URL、文档、文本等多种输入方式自动生成结构化的PPT课件，并提供完整的学习记录管理和AI助手功能。

## 功能特性

### 🎯 核心功能
- **智能内容提取**: 支持知乎、微信公众号、CSDN等主流平台的内容爬取
- **AI课件生成**: 基于通义千问AI自动生成结构化PPT课件
- **多格式支持**: 支持URL、PDF、Word、TXT、Markdown等多种输入格式
- **语音合成**: 自动生成课程讲解音频
- **学习记录**: 完整的学习进度跟踪和统计分析
- **AI助手**: 智能问答和学习指导

### 📱 用户界面
- **微信小程序**: 原生微信小程序界面，用户体验优秀
- **响应式设计**: 适配各种屏幕尺寸
- **实时交互**: 支持实时进度更新和状态同步

### 🔧 技术架构
- **后端**: Go + Gin + GORM + Redis
- **前端**: 微信小程序原生开发
- **AI服务**: 阿里云通义千问API
- **数据库**: MySQL 8.0+
- **缓存**: Redis 6.0+

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   微信小程序     │    │   后端API服务    │    │   AI服务        │
│                 │    │                 │    │                 │
│ - 用户界面      │◄──►│ - 内容处理      │◄──►│ - 通义千问      │
│ - 课程播放      │    │ - 课件生成      │    │ - 语音合成      │
│ - 学习记录      │    │ - 用户管理      │    │ - 智能分析      │
│ - AI助手        │    │ - 数据存储      │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   数据存储       │
                       │                 │
                       │ - MySQL         │
                       │ - Redis         │
                       │ - 文件存储      │
                       └─────────────────┘
```

## 快速开始

### 环境要求

- Go 1.19+
- MySQL 8.0+
- Redis 6.0+
- 微信开发者工具
- 阿里云通义千问API密钥

### 1. 克隆项目

```bash
git clone https://github.com/your-username/ai-classroom.git
cd ai-classroom
```

### 2. 配置环境

```bash
# 复制配置文件
cp backend/configs/config.yaml.example backend/configs/config.yaml

# 编辑配置文件
vim backend/configs/config.yaml
```

### 3. 启动系统

```bash
# 使用启动脚本（推荐）
./start.sh

# 或手动启动
cd backend
go mod tidy
go run cmd/main.go
```

### 4. 前端配置

1. 打开微信开发者工具
2. 导入项目：选择 `frontend` 目录
3. 配置AppID（测试号或正式号）
4. 配置服务器域名白名单

## 项目结构

```
ai-classroom/
├── backend/                 # 后端服务
│   ├── cmd/                # 主程序入口
│   ├── configs/            # 配置文件
│   ├── internal/           # 内部包
│   │   ├── handlers/       # HTTP处理器
│   │   ├── middleware/     # 中间件
│   │   ├── models/         # 数据模型
│   │   ├── repositories/   # 数据访问层
│   │   ├── services/       # 业务逻辑层
│   │   └── websocket/      # WebSocket服务
│   ├── pkg/                # 公共包
│   │   ├── ai/             # AI服务
│   │   ├── crawler/        # 爬虫服务
│   │   ├── parser/         # 文档解析
│   │   ├── redis/          # Redis客户端
│   │   └── utils/          # 工具函数
│   └── go.mod              # Go模块文件
├── frontend/               # 前端小程序
│   ├── pages/              # 页面文件
│   │   ├── index/          # 首页
│   │   ├── course/         # 课程相关
│   │   ├── player/         # 播放器
│   │   ├── profile/        # 用户中心
│   │   └── ai-assistant/   # AI助手
│   ├── apis/               # API接口
│   ├── utils/              # 工具函数
│   ├── components/         # 组件
│   └── app.json            # 小程序配置
├── docs/                   # 文档
├── scripts/                # 脚本文件
├── start.sh               # 启动脚本
└── README.md              # 项目说明
```

## API文档

### 认证接口

```http
POST /api/v1/auth/wechat
POST /api/v1/auth/refresh
GET  /api/v1/auth/profile
```

### 内容处理

```http
POST /api/v1/content/validate-url
POST /api/v1/content/extract
POST /api/v1/content/text
POST /api/v1/content/upload
```

### 课程管理

```http
GET    /api/v1/courses
POST   /api/v1/courses
GET    /api/v1/courses/:id
PUT    /api/v1/courses/:id
DELETE /api/v1/courses/:id
```

### 学习记录

```http
GET    /api/v1/courses/learning-records
GET    /api/v1/courses/learning-records/stats
PUT    /api/v1/courses/:id/progress
POST   /api/v1/courses/:id/complete
```

### AI助手

```http
POST   /api/v1/ai-assistant/chat
GET    /api/v1/ai-assistant/history
DELETE /api/v1/ai-assistant/history
```

## 数据库设计

### 主要数据表

- `users`: 用户信息
- `courses`: 课程信息
- `slides`: 幻灯片内容
- `learning_records`: 学习记录
- `ai_assistant_chats`: AI助手对话记录

### 关系图

```
users (1) ──── (n) courses
courses (1) ──── (n) slides
users (1) ──── (n) learning_records
courses (1) ──── (n) learning_records
users (1) ──── (n) ai_assistant_chats
```

## 部署指南

### 开发环境

```bash
# 1. 安装依赖
go mod tidy

# 2. 配置环境变量
export DASHSCOPE_API_KEY="your_api_key"
export DB_HOST="localhost"
export REDIS_HOST="localhost"

# 3. 启动服务
go run cmd/main.go
```

### 生产环境

```bash
# 1. 构建二进制文件
go build -o ai-classroom cmd/main.go

# 2. 配置生产环境
cp configs/config.prod.yaml configs/config.yaml

# 3. 启动服务
./ai-classroom
```

### Docker部署

```bash
# 构建镜像
docker build -t ai-classroom .

# 运行容器
docker run -d -p 3000:3000 ai-classroom
```

## 开发指南

### 代码规范

- 使用Go官方代码规范
- 遵循RESTful API设计原则
- 使用语义化版本控制
- 编写单元测试和集成测试

### 提交规范

```
feat: 新功能
fix: 修复bug
docs: 文档更新
style: 代码格式调整
refactor: 代码重构
test: 测试相关
chore: 构建过程或辅助工具的变动
```

### 测试

```bash
# 运行单元测试
go test ./...

# 运行集成测试
go test -tags=integration ./...

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 性能优化

### 后端优化

- 使用Redis缓存热点数据
- 实现数据库连接池
- 优化SQL查询
- 使用异步处理耗时操作

### 前端优化

- 图片懒加载
- 分页加载数据
- 本地缓存策略
- 减少网络请求

## 监控和日志

### 日志配置

```yaml
# configs/config.yaml
logging:
  level: info
  format: json
  output: file
  file: logs/app.log
  max_size: 100MB
  max_age: 30d
```

### 监控指标

- API响应时间
- 错误率统计
- 系统资源使用
- 用户活跃度

## 常见问题

### Q: 如何配置AI服务？
A: 在配置文件中设置阿里云通义千问API密钥：
```yaml
aliyun:
  dashscope:
    api_key: "your_api_key_here"
```

### Q: 如何添加新的内容平台？
A: 在 `pkg/crawler/content_extractor.go` 中添加新的平台规则。

### Q: 如何自定义课件模板？
A: 在 `pkg/ai/dashscope.go` 中修改PPT生成提示词。

### Q: 如何扩展AI助手功能？
A: 在 `internal/handlers/content.go` 中扩展AI助手对话逻辑。

## 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 联系方式

- 邮箱: wjb18824872920@163.com

## 更新日志

### v1.0.0 (2024-01-01)
- 🎉 初始版本发布
- ✨ 支持URL内容提取和AI课件生成
- ✨ 实现学习记录管理
- ✨ 集成AI助手功能
- ✨ 完成微信小程序前端

---

**AI课堂** - 让学习更智能，让知识更生动！ 🚀