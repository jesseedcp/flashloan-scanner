# 前端控制台工程实施说明

> 文档角色：工程实施说明  
> 目标读者：负责继续编码实现的 AI / 工程同学  
> 文档用途：说明前端实时控制台如何接入现有 `flashloan-scanner` 代码体系

## 1. 文档目标

本说明不是产品需求文档，而是实现指引。目标是让实现方在不推翻现有 `flashloan-scanner` 结构的前提下，新增一套适合课程项目演示的实时前后端联动能力。

实现目标包括：

- 前端可发起扫描任务
- 一次任务同时启动 `Aave V3`、`Balancer V2`、`Uniswap V2`
- 扫描过程中通过 `WebSocket` 实时推送进度与结果
- 扫描结果继续落库
- 前端可通过 HTTP 查询结果详情

## 2. 实现原则

### 2.1 复用现有 scanner

不要重写协议识别逻辑。应尽量复用：

- `scanner/orchestrator`
- `scanner/extractor`
- `scanner/verifier`
- `scanner/aggregator`
- `database/scanner`
- `database/event`

### 2.2 新增一层“任务管理 + 实时推送”

新的前后端能力，不应直接侵入每个协议的识别细节，而应当在现有 scanner 之上增加：

- 任务管理层
- observer 事件层
- WebSocket 推送层
- HTTP 查询层

### 2.3 结果写库仍是主路径

扫描期间虽然会实时推送结果，但数据库仍然是系统记录结果的主来源。实时消息只是“展示层输出”，不替代落库。

## 3. 推荐目录结构

建议在 `flashloan-scanner` 下新增以下目录：

- `api/http/server.go`
- `api/http/routes.go`
- `api/http/handlers/summary.go`
- `api/http/handlers/transactions.go`
- `api/http/handlers/jobs.go`
- `api/ws/handler.go`
- `api/ws/messages.go`
- `api/service/job_manager.go`
- `api/service/job_state.go`
- `api/service/observer.go`
- `api/service/runner_bridge.go`

前端建议放在独立目录：

- `frontend/`
  - `src/pages/ScanConsole.tsx`
  - `src/pages/TransactionDetail.tsx`
  - `src/components/ScanControlPanel.tsx`
  - `src/components/ProtocolProgressCard.tsx`
  - `src/components/FindingsTable.tsx`
  - `src/components/LiveLogPanel.tsx`
  - `src/lib/ws.ts`
  - `src/lib/api.ts`
  - `src/store/useScanStore.ts`

## 4. 后端核心模块说明

## 4.1 Job Manager

建议实现一个内存任务管理器。

职责：

- 创建新任务
- 维护总任务状态
- 维护三个协议子任务状态
- 缓存实时 findings
- 为 WebSocket handler 提供事件广播

建议结构：

```go
type JobManager struct {
    mu sync.RWMutex
    jobs map[string]*ScanJobState
}
```

`ScanJobState` 建议包含：

- 基本参数：`jobID`, `chainID`, `startBlock`, `endBlock`, `traceEnabled`
- 总状态：`status`, `startedAt`, `finishedAt`
- 总统计：`totalCandidates`, `totalVerified`, `totalStrict`
- 子任务状态：`map[scanner.Protocol]*ProtocolJobState`
- findings 列表
- logs 列表
- websocket subscribers

### 4.2 ProtocolJobState

建议字段：

```go
type ProtocolJobState struct {
    Protocol        string
    Status          string
    StartBlock      uint64
    EndBlock        uint64
    CurrentBlock    uint64
    FoundCandidates int
    FoundVerified   int
    FoundStrict     int
    Error           string
}
```

## 4.3 Observer

当前 scanner 要支持“发现就推送”，关键在 observer。

建议新增接口：

```go
type ScanObserver interface {
    OnProtocolStarted(jobID string, protocol scanner.Protocol, startBlock, endBlock uint64)
    OnProtocolProgress(jobID string, protocol scanner.Protocol, currentBlock uint64)
    OnFinding(jobID string, protocol scanner.Protocol, finding FindingMessage)
    OnProtocolCompleted(jobID string, protocol scanner.Protocol, summary ProtocolSummary)
    OnProtocolFailed(jobID string, protocol scanner.Protocol, err error)
}
```

建议由 `api/service/observer.go` 实现一个默认 observer，并把事件转发到 job manager。

## 4.4 Runner Bridge

新增一层 bridge，把现有的各协议 runner 包起来，统一注入 observer。

作用：

- 接收前端任务参数
- 构建三个 protocol runner
- 统一管理上下文与并发
- 在 scanner 运行的关键节点调用 observer

## 5. 对现有 scanner 的改动建议

## 5.1 ProtocolRunner 增加 observer 支持

建议在 `scanner/orchestrator/protocol_runner.go` 中增加可选 observer。

可选方法：

```go
func (r *ProtocolRunner) WithObserver(observer ScanObserver) *ProtocolRunner
```

在以下位置调用 observer：

- `RunOnce` 开始时，触发 `OnProtocolStarted`
- 每批 block 完成时，触发 `OnProtocolProgress`
- 每个 interaction 验证完成并聚合后，触发 `OnFinding`
- 正常结束时，触发 `OnProtocolCompleted`
- 出错时，触发 `OnProtocolFailed`

### 5.2 finding 触发时机

`finding` 最好不要在 candidate 刚提取时就推送，而应在一条 interaction 已经完成验证并聚合后推送。这样前端收到的结果会更稳定。

推荐 finding 最小字段：

- `tx_hash`
- `protocol`
- `block_number`
- `candidate`
- `verified`
- `strict`

## 6. WebSocket 设计

## 6.1 路由建议

Gin 路由建议：

- `GET /ws/scan`

前端进入页面后连接该地址。

## 6.2 通信模式

采用单连接双向通信：

- 前端通过 WebSocket 发 `start_scan`
- 后端持续推送运行状态与结果

第一版不做：

- heartbeat 复杂策略
- reconnect 恢复任务
- 多任务 multiplex

## 6.3 消息结构

统一格式：

```go
type WSMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}
```

建议至少实现这些消息：

前端发送：

- `start_scan`

后端发送：

- `job_started`
- `job_progress`
- `protocol_progress`
- `finding`
- `protocol_completed`
- `job_completed`
- `job_failed`

## 7. HTTP API 设计

除 WebSocket 外，还要给前端详情页和汇总页提供 REST API。

建议路由：

- `GET /api/v1/summary`
- `GET /api/v1/transactions`
- `GET /api/v1/transactions/:txHash`
- `GET /api/v1/jobs/:jobId/results`

说明：

- `/summary`：首页汇总
- `/transactions`：结果列表
- `/transactions/:txHash`：单笔详情
- `/jobs/:jobId/results`：当前任务产生的结果集合

## 8. 数据读取建议

### 8.1 不要复用 CLI 文本输出

`flashloan-report` 的 CLI 适合终端，不适合作为 API 输出源。

正确做法是：

- 复用其查询思路
- 直接返回 JSON DTO

### 8.2 新增 API 专用 service 层

建议新增 `api/service/query_service.go`，封装以下查询：

- summary 聚合
- transaction list
- transaction detail
- job results

## 9. 前端工程建议

## 9.1 技术选型

建议：

- `React + Vite + TypeScript`
- `Tailwind`
- `React Router`
- `zustand`
- 原生 `WebSocket`

### 9.2 状态管理建议

前端建议维护以下状态：

- socket 连接状态
- 当前 job 状态
- 三协议 progress map
- live findings
- live logs
- 详情页选中的 tx

最小状态结构：

```ts
type ScanState = {
  socketStatus: 'disconnected' | 'connected'
  jobStatus: 'idle' | 'running' | 'completed' | 'failed'
  jobId?: string
  summary: {
    totalCandidates: number
    totalVerified: number
    totalStrict: number
  }
  protocolProgress: Record<string, ProtocolProgress>
  findings: FindingItem[]
  logs: string[]
}
```

## 9.3 页面拆分建议

### `ScanConsole.tsx`

负责：

- 建立 WebSocket
- 提交扫描参数
- 接收消息并更新 store
- 渲染控制台主视图

### `TransactionDetail.tsx`

负责：

- 根据 `txHash` 调 REST API
- 渲染 transaction / interaction / leg 明细

### `ScanControlPanel.tsx`

负责：

- 表单输入
- 默认填充演示参数
- 点击开始扫描

### `ProtocolProgressCard.tsx`

负责：

- 单协议进度卡片展示

### `FindingsTable.tsx`

负责：

- 实时结果表
- 点击某条结果进入详情

### `LiveLogPanel.tsx`

负责：

- 实时日志列表展示

## 10. 推荐默认参数

为保证演示稳定，建议前端页面默认写入：

- `chain_id = 1`
- `start_block = 22485844`
- `end_block = 22486844`
- protocols = 三协议全开
- `trace_enabled = false`

之后再允许用户手动修改。

## 11. 第一版实现边界

第一版只要求完成：

- 单个实时控制台页面
- 单个总任务
- 三协议并发扫描
- WebSocket 实时推送
- REST 详情接口
- 扫描结果写库并可查询

第一版不要求：

- 多任务调度
- 历史任务持久化
- 权限系统
- 取消任务
- 复杂错误恢复

## 12. 推荐实施顺序

建议按照以下顺序实现：

1. 给 `ProtocolRunner` 增加 observer 接口
2. 实现 `JobManager`
3. 实现 `runner bridge`，支持三协议并发
4. 实现 `WebSocket handler`
5. 实现 REST 查询接口
6. 初始化前端工程
7. 接通 WebSocket 与主页面
8. 接通详情页
9. 优化样式与演示流程

## 13. 实现方必须避免的错误

实现时应避免以下问题：

- 不要重写一套新的 scanner
- 不要把实时结果只做成数据库轮询
- 不要把三协议拆成用户手动点三次
- 不要只在任务结束时一次性返回所有结果
- 不要把 CLI 文本输出直接暴露给前端
- 不要第一版就引入过多平台化能力

## 14. 最终交付标准

如果实现完成，应该能够做到：

- 前端点一次开始扫描
- 后端同时跑三协议
- 前端实时看到进度和结果
- 扫描完成后可点开单笔详情
- 演示时可以稳定使用真实窗口 `22485844-22486844`

这就足以满足课程项目的第一版交付目标。
