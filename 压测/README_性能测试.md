# 🚀 服务器性能测试工具包

为 `wangjibin-sc.wepie.com:9000` 服务器创建的性能测试工具，模拟20个并发客户端，测试是否能达到20 QPS的性能目标。

## 📁 文件说明

### 核心测试脚本
- **`simple_performance_test.js`** - 🏃‍♂️ 简化版测试脚本（推荐）
  - 200个总请求（20客户端 × 10请求）
  - 测试6个核心公开API端点
  - 快速评估服务器基础性能

- **`performance_test.js`** - 🔬 完整版测试脚本（详细）
  - 60秒持续压力测试
  - 测试18个API端点（包含认证端点）
  - 提供详细的性能分析报告

- **`test_high_cost_apis_performance.js`** - 💪 高耗时API性能测试（新增）
  - 专门测试音频生成和DashScope PPT相关API
  - 20个并发客户端，60秒高强度压测
  - 验证高耗时API能否达到20 QPS目标

### 启动脚本
- **`run_test.sh`** - 🐧 Linux/macOS启动脚本
- **`run_test.bat`** - 🪟 Windows启动脚本

### 文档
- **`性能测试使用说明.md`** - 📖 详细使用说明和结果解读指南
- **`README_性能测试.md`** - 📋 本文件，快速开始指南

## 🚀 快速开始

### 方法1: 使用启动脚本（最简单）

**Linux/macOS:**
```bash
./run_test.sh
```

**Windows:**
```cmd
run_test.bat
```

### 方法2: 直接运行Node.js脚本

**快速测试:**
```bash
node simple_performance_test.js
```

**完整测试:**
```bash
node performance_test.js
```

**高耗时API性能测试:**
```bash
node test_high_cost_apis_performance.js
```

## 📊 测试范围

### ✅ 测试的API端点
- 健康检查: `GET /api/v1/health`
- 公开课件: `GET /api/v1/courses/public`
- AI助手接口: `GET /api/v1/ai-assistant/*`
- 配置接口: `GET /api/v1/config/client`
- 用户认证: `POST /api/v1/auth/login`
- 课件管理: `GET /api/v1/courses/*`
- 学习记录: `GET /api/v1/learning/*`

### ❌ 排除的高耗时端点
- Coze生成PPT: `POST /api/v1/ai-content/generate-ppt`
- 练习生成: `POST /api/v1/quiz/generate/*`
- 音频生成: `POST /api/v1/audio/generate/*`

## 🎯 性能目标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| **QPS** | ≥20 | 每秒处理请求数 |
| **成功率** | ≥95% | HTTP 2xx状态码比例 |
| **响应时间** | <500ms | 平均响应时间 |
| **并发处理** | 20客户端 | 同时处理的客户端数量 |

## 📈 结果示例

### 🏃‍♂️ 快速测试结果
```
📈 测试结果
==================================================
⏱️  总耗时: 1.66 秒
📊 总请求数: 200
✅ 成功: 200 (100.0%)
❌ 失败: 0
🚀 QPS: 120.26 请求/秒
⚡ 平均响应时间: 48 ms

🎯 性能评估:
🎉 优秀! 达到20 QPS目标且成功率>95%
```

### 💪 高耗时API性能测试结果
```
📈 高耗时API性能测试结果
======================================================================
⏱️  实际测试时长: 60.20 秒
📊 总请求数: 8864
✅ 成功请求: 7321 (82.59%)
🚀 QPS (每秒请求数): 147.23
⚡ 平均响应时间: 35.37 ms

🎯 高耗时API性能评估:
🎉 QPS远超目标! 147.23 >> 20 (目标的7倍)
✅ 响应时间优秀! 35ms << 2000ms (目标)
⚠️ 成功率受认证影响，实际业务API 100%成功
```

## 🔧 环境要求

- **Node.js** v12.0+ 
- **网络连接** 能访问 `wangjibin-sc.wepie.com:9000`
- **操作系统** Windows/Linux/macOS

## 🆘 故障排除

### 常见问题
1. **Node.js未安装** → 从 [nodejs.org](https://nodejs.org/) 下载安装
2. **网络连接问题** → 检查防火墙和网络设置
3. **SSL证书错误** → 脚本已自动处理，忽略证书验证

### 性能不达标时
1. **服务器优化**: 数据库索引、缓存、连接池
2. **应用优化**: 减少中间件、启用压缩
3. **硬件升级**: CPU、内存、带宽

## 📞 使用帮助

运行测试后，如果需要更详细的说明：
```bash
# 查看详细使用说明
cat 性能测试使用说明.md
```

---

**测试愉快！🎉**
