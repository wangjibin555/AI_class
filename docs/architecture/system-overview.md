# 系统架构概览

## 整体架构

文档转PPT系统采用微服务架构，将系统划分为多个独立的服务单元，每个服务专注于特定的业务功能。

## 核心服务

### 1. 文档上传服务 (Upload Service)
**职责**: 
- 接收用户上传的文档
- 验证文件格式和大小
- 将文件保存到对象存储
- 推送解析任务到消息队列

**技术栈**: Node.js/Express

### 2. 内容解析服务 (Parse Service)
**职责**:
- 从消息队列获取解析任务
- 解析不同格式的文档（Word, PDF, Markdown等）
- 提取文档结构和内容
- 将结构化数据传递给分块服务

**技术栈**: Python/FastAPI

### 3. 内容分块服务 (Chunk Service)
**职责**:
- 接收解析后的内容
- 根据内容结构智能分块
- 生成PPT页面结构
- 调用PPT生成服务

**技术栈**: Python/FastAPI

### 4. PPT生成服务 (PPT Generation Service)
**职责**:
- 根据分块内容生成PPT
- 应用模板和样式
- 将生成的PPT保存到对象存储
- 更新任务状态到数据库

**技术栈**: Python/FastAPI + python-pptx

### 5. 文件存储服务 (File Service)
**职责**:
- 提供文件上传/下载接口
- 管理文件元数据
- 处理文件权限

**技术栈**: Node.js/Express

### 6. 认证授权服务 (Auth Service)
**职责**:
- 用户注册和登录
- JWT令牌生成和验证
- 权限管理

**技术栈**: Node.js/Express

## 数据流

```
用户上传文档 
  → API Gateway 
  → Upload Service 
  → OSS存储 
  → RabbitMQ队列
  
RabbitMQ队列 
  → Parse Service 
  → 内容解析 
  → Chunk Service 
  → 内容分块
  
Chunk Service 
  → PPT Gen Service 
  → 生成PPT 
  → OSS存储 
  → 通知用户
```

## 通信方式

1. **同步通信**: API Gateway与各服务之间使用HTTP/REST
2. **异步通信**: 服务间使用RabbitMQ进行异步任务处理
3. **缓存**: 使用Redis缓存热点数据和会话信息
4. **存储**: 使用MinIO存储文件，PostgreSQL存储元数据

## 扩展性

- 每个服务可独立部署和扩展
- 使用容器化（Docker）和编排（Kubernetes）
- 支持水平扩展
- 消息队列解耦服务依赖

## 可靠性

- 消息队列保证任务不丢失
- 数据库持久化关键数据
- 对象存储提供文件冗余
- 服务健康检查和自动重启


