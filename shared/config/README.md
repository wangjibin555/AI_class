# 配置文件说明

## 重要提示 ⚠️

**`default.json` 文件包含敏感信息（密码、密钥等），已被 `.gitignore` 忽略，不会提交到 Git。**

## 使用方法

### 1. 复制示例文件

首次使用时，请复制示例文件：

```bash
cp shared/config/default.json.example shared/config/default.json
```

### 2. 修改配置

编辑 `default.json`，将敏感信息替换为实际值：

- `DB_PASSWORD`: PostgreSQL 数据库密码
- `RABBITMQ_PASSWORD`: RabbitMQ 密码
- `MINIO_SECRET_KEY`: MinIO 密钥
- `JWT_SECRET`: JWT 签名密钥

### 3. 使用环境变量（推荐）

配置文件支持环境变量替换。设置以下环境变量：

```bash
export DB_PASSWORD=your_actual_password
export RABBITMQ_PASSWORD=your_actual_password
export MINIO_SECRET_KEY=your_actual_secret_key
export JWT_SECRET=your_actual_jwt_secret
```

然后在配置文件中使用 `${ENV_VAR_NAME:-default_value}` 格式。

## 安全最佳实践

1. ✅ **不要**将 `default.json` 提交到 Git
2. ✅ **使用**环境变量存储敏感信息
3. ✅ **使用** `.env` 文件（已添加到 `.gitignore`）
4. ✅ **定期**更换密码和密钥
5. ✅ **使用**强密码（至少 16 位，包含大小写字母、数字和特殊字符）

## 环境变量列表

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DB_USER` | PostgreSQL 用户名 | `admin` |
| `DB_PASSWORD` | PostgreSQL 密码 | `changeme` |
| `RABBITMQ_USER` | RabbitMQ 用户名 | `admin` |
| `RABBITMQ_PASSWORD` | RabbitMQ 密码 | `changeme` |
| `MINIO_ROOT_USER` | MinIO 用户名 | `admin` |
| `MINIO_ROOT_PASSWORD` | MinIO 密码 | `changeme` |
| `MINIO_ACCESS_KEY` | MinIO Access Key | `admin` |
| `MINIO_SECRET_KEY` | MinIO Secret Key | `changeme` |
| `JWT_SECRET` | JWT 签名密钥 | `your-secret-key-change-in-production` |













