# 性能优化现状与改进建议（AI课堂）

本文系统梳理项目在课件/音频生成、工作流调用、前端播放、存储与数据库等链路的性能优化要点与边界，并提出可落地的改进方案。重点扩展“已做优化”的技术细节、实现方式与适用边界，辅以必要的代码参考与指标建议。

---

## 1. 已有优化点（现状，含实现细节）

### 1.1 异步任务模型（后端）
- 目标：用户发起“全课音频生成”时，接口快速返回，长耗时工作在后台执行，避免阻塞请求线程。
- 关键实现：
  - `go h.generateAudioAsync(courseID, request)` 启动后台任务；立即返回 `{status: processing}`。
  - 进度内存态：`progressMap map[uint]*AudioGenerationProgress + RWMutex`，同一个 `courseId` 若已有进行中任务直接返回进度，避免重复提交。
  - 串行批处理：遍历课程所有 `slides`，逐一生成音频与写库，期间每步更新进度（完成数、百分比、当前标题）。
  - 频率限制：每张间隔 `1s`（防止被上游 TTS 限流）。
- 代码参考：`backend/internal/handlers/audio_handler.go` → `GenerateCourseAudio`、`generateAudioAsync`、`getSlideAudioInfo`。
- 适用边界：
  - 单实例内存态进度；适合 DEMO 与单节点部署；重启丢进度、不可横向扩展。
  - 每张 1s 的固定节流是保守值，保障稳定但牺牲吞吐。

### 1.2 工作流调用稳定性（Coze）
- 目标：稳定拉起与完成 Coze 工作流（PPT/练习），降低超时/失败对用户的影响。
- 关键实现：
  - 直返/异步自适应：当 `run` 返回“直接结果”时解析 JSON；否则返回 `execute_id` 进入 `pollWorkflowResult` 轮询。
  - 指数退避（含封顶）：`retryDelay *= 2`，最大 30s；HTTP 客户端设定较长 `Timeout`，保证大任务能跑完；失败后打印上下文日志。
  - 输出解析健壮性：多层 JSON 解析+SSE 前缀清理（兼容 `data:`）、字符串/对象两种 data 形态。
- 代码参考：`backend/internal/coze/exercise_workflow_client.go` → `runExerciseWorkflow`、`pollWorkflowResult`、`parseWorkflowOutput`。
- 适用边界：
  - 仍依赖轮询；高并发下会有额外拉取成本；无服务端回推（Webhook）。

### 1.3 TTS 兼容性与健壮性（阿里云 NLS 增强版）
- 目标：提升语音合成的成功率与前端播放兼容性。
- 关键实现：
  - 固定输出 MP3：即使上游返回 WAV，亦做转换/兼容处理，保证小程序播放器可播（MP3 ID3 头+标准帧，最小帧数保证，避免“seek/pause fail”）。
  - Token 获取与连接管理：在每次合成前获取 Token，使用 WebSocket（`nls.NewSpeechSynthesis`）流式接收音频片段，写入 buffer；超时/错误有清晰日志链路。
  - 失败重试：多次重试（当前为“线性退避”，见“需改进”）。
  - 时长估算：统一估算/转换时长（ms→s），用于写库与前端展示。
- 代码参考：`backend/pkg/tts/aliyun_tts_enhanced.go` → `GenerateAudioEnhanced`、`onSynthesisResult`、`waitForCompletion`、`detectAudioFormat`、`convertWAVToMP3`。
- 适用边界：
  - 当前退避策略为线性，面对瞬时抖动/雪崩时效果不如指数退避+jitter。

### 1.4 节流与限频（批量 TTS）
- 目标：避免击穿上游速率限制，稳态生成。
- 关键实现：
  - 课程批量：默认“串行 + Sleep(1s)”；另一处基础服务为“串行 + 100ms”。
  - 优点：实现简单、稳定可预期。
- 适用边界：
  - 吞吐受限，无自适应；在弱网/抖动下容易整体变慢。

### 1.5 存储落地（本地）
- 目标：快速落盘并返回可访问 URL，降低依赖。
- 关键实现：
  - `storage.SaveAudio(filepath, data)` 将文件写入 `./storage/audio/slides/...`，并基于 `server.file_base_url` 返回直链。
  - 目录自动创建、错误抛出清晰。
- 适用边界：
  - 本地盘 + 直链，不适合多实例/横向扩展；缺少 CDN 与 Range 支持。

### 1.6 前端播放体验（音画同步）
- 目标：提升播放连贯性与用户观感。
- 关键实现：
  - AudioSyncManager：100ms 时间检查、200ms 容差；预加载“下一段音频”；以“音频结束事件”驱动切页，减少误跳。
  - 错误回调：`syncError` 事件；异常时不盲跳、避免二次错误。
- 代码参考：`frontend/utils/audio-sync.js`。
- 适用边界：
  - 目前关闭了“自动纠错”（基于独立音频文件的现实）；检查频率固定，setData 次数较多，主线程可能抖动。

### 1.7 数据库访问
- 目标：降低写入放大、保证序一致性。
- 关键实现：
  - 批量插入：`CreateInBatches(questions, 10)`（练习生成）；
  - 查询排序：`slide_number ASC`，前端播放一致。
- 适用边界：
  - 音频写入后的 `slide` 更新仍是“逐条更新”，大量页数时事务放大会拉高写入 QPS。

---

## 2. 痛点与不足（现状边界）

1) **TTS 重试为线性退避**：`delay = base * attempt`，缺少抖动与断路器，易在抖动期产生“惊群”。
2) **异步任务缺少持久化**：进度保存在内存；多实例/重启不可恢复；无统一队列，扩容困难。
3) **批量 TTS 串行 + Sleep 粗粒度**：无有界并发/自适应；CPU/IO 利用不足；整体时延偏大。
4) **DB 更新逐条写**：页数多时写入放大；无批量/分批事务提交。
5) **本地存储 + 直链**：无 CDN；多实例需共享存储；大文件缺 Range 支持。
6) **工作流轮询固定间隔**：缺少动态退避或 Webhook；高并发下轮询成本显著。
7) **前端 setData 频率高**：100ms/周期 + 多字段 setData，无批处理；弱机/弱网下主线程抖动。
8) **观测性弱**：缺少 Prometheus 指标与 pprof/trace；难以快速定位热点与容量边界。

---

## 3. 建议优化方案（含落地示例）

### 3.1 短期（1–2 周）

A) 将 TTS 重试改为“指数退避 + 抖动 + 封顶 + 断路器”
```go
// 以增强 TTS 为例（替换线性退避）
base := c.retryDelay                 // e.g. 1s
maxDelay := 30 * time.Second
for attempt := 1; attempt <= c.maxRetries; attempt++ {
  res, err := c.generateAudioWithRetry(text, voiceType, attempt)
  if err == nil { return res, nil }
  if attempt < c.maxRetries {
    delay := base * time.Duration(1<<(attempt-1)) // 2^(n-1)
    if delay > maxDelay { delay = maxDelay }
    jitter := time.Duration(rand.Int63n(int64(delay/2)))
    time.Sleep(delay + jitter)
  }
}
```
- 断路器：连续失败 N 次进入“冷却期”，快速失败，保护上下游。

B) 可观测性与压测基线
- 指标建议：
  - `tts_duration_ms`（P50/P95）、`tts_success_rate`、`workflow_latency_ms`、`workflow_error_rate`、`storage_write_ms`、`slides_update_qps`；
  - 前端曝光 `audio_preload_hit_rate`、`player_rebuffer_count`。
- 上线 `pprof`/`trace` 开关；压测 1×/周，保留基线图。

C) 前端 setData 合并 + 节流
- 将 100ms 改为：RAF 驱动 + 200ms 节流；将多字段 setData 合并为单次；失败预加载降速重试（指数退避）。

### 3.2 中期（2–4 周）

A) Worker Pool + 有界并发的 TTS 批量生成
```go
// 伪代码：N 个 worker 处理 slides，通道大小 = 队列长度
pool := NewWorkerPool(concurrencyN)
for _, slide := range slides { pool.Submit(func(){ genSlideAudio(slide) }) }
pool.Wait()
```
- 动态并发：根据错误率/HTTP 429 等信号 AIMD 调整并发；
- 连接/Token 复用；
- 失败任务进入“补偿队列”（次数封顶）。

B) 任务队列 + 进度持久化
- 使用 Redis Stream / RabbitMQ / NSQ：提交→可重试队列→Worker 消费；
- 进度写 Redis（`audio:progress:{courseId}`，TTL=24h），多实例共享、重启可恢复；
- 后端接口从 Redis 读取进度，取代进程内 `progressMap`。

C) 存储/CDN 与去重缓存
- 切换到对象存储（OSS/S3）+ CDN；音频 URL 加缓存头、支持 Range；
- 内容寻址缓存：`cacheKey = hash(text + voiceType)`，命中即短路合成；
- 结合“预热”逻辑（生成后直接 PUT CDN）。

D) 批量 DB 写入
- 将更新聚合为“每 10 条一次事务”或使用 `UPDATE ... WHERE id IN (...)`；
- 高频字段加索引校验：`course_id`, `slide_number`。

### 3.3 长期（4–8 周）

A) 流式 TTS / 分段拼接
- 长文本流式写入，边合成边落盘，降低峰值内存；必要时分段无缝拼接，提升吞吐。

B) Webhook / SSE 回推替代轮询
- 工作流完成后回推（签名校验+落库），后端不再固定轮询；
- 客户端使用事件订阅替代短轮询。

C) 全链路容量与 SLO
- 针对不同页数/并发/网络的容量曲线；
- 根据曲线设定默认并发、退避参数与超时阈值。

---

## 4. 风险与收益评估
- 指数退避 + Worker Pool：改动小、收益大（TTS 成功率上升、尾延迟收敛、后端稳定）；
- 队列化与共享进度：支撑多实例、灰度与弹性伸缩，避免“单机内存态”丢任务；
- CDN 与对象存储：首包改善显著，播放更稳；
- 可观测性：定位瓶颈更快，恢复更及时。

---

## 5. 落地清单（可执行）
- [ ] TTS 重试从线性 → 指数退避 + 抖动 + 断路器
- [ ] 批量生成改为 Worker Pool（并发=3，配置化，AIMD 自适应）
- [ ] 任务进度持久化到 Redis（key: `audio:progress:{courseId}`，TTL=24h）
- [ ] `slides` 批量更新（每 10 条一批/IN 子句批写）
- [ ] 切换 OSS/S3 + CDN，开启 Range/Cache-Control
- [ ] Prometheus 指标接入 + pprof/trace 开关
- [ ] 前端：RAF + 200ms 节流、合并 setData、预加载失败指数退避

---

### 参考代码路径
- 异步与进度：`backend/internal/handlers/audio_handler.go`
- 路由与注入：`backend/internal/routes/audio_routes.go`
- TTS 服务：`backend/internal/services/tts_service.go`
- 增强 TTS 客户端：`backend/pkg/tts/aliyun_tts_enhanced.go`
- 工作流客户端（退避样例）：`backend/internal/coze/exercise_workflow_client.go`
- 存储：`backend/pkg/storage/storage.go`
