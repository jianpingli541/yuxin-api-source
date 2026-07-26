# 豫鑫 API 中转站 — 变更日志

---

## v1.0.0-yuxin-hotfix.1 (2026-07-26)

### 🛡️ 安全/稳定性修复（生产级）

#### CustomEvent SSE 渲染器 panic 修复 (`common/custom-event.go`)

**问题**: `writeData` 函数中 `data.(string)` 未检查类型断言。`CustomEvent.Data` 类型为 `interface{}`，所有流式 AI 响应（SSE）都经过此路径。任何调用方传递非 string 类型（`[]byte` / `int` / `nil` / `struct`）都会触发 panic，中断整个流式响应。

**影响范围**: 15+ SSE 调用点（`relay/helper/common.go` + `relay/channel/*/relay-*.go`）

**修复**:
- 引入 `stringifyEventData` 类型分支处理（`nil` / `string` / `[]byte` / `default fmt.Sprint`）
- 保留原 SSE wire 契约（`data` 开头才追加 `\n\n` 终止符）
- 新增 137 行回归测试，覆盖 6 种数据类型 + `-race` 并发安全验证

#### sync.Mutex 锁值传递修复

**问题**: `CustomEvent` 结构体含 `sync.Mutex` 值字段，但 `Render/WriteContentType` 接收器为值类型（gin Renderer 契约要求）。每次调用都复制锁 —— go vet 报警 ×4，且复制的锁无法真正同步并发。

**修复**: 删除结构体的 `Mutex` 字段，改用包级 `sseHeaderMu sync.Mutex` 保护 header map 写操作。

### 📊 验证证据
- `go vet ./common/...`: 4 警告 → **0**
- `go test ./common/ -race -count=3`: **PASS**
- `go build ./...`: 通过
- 容器升级后 15s 内 healthy，**零停机**
- 升级过程持续外部访问无 5xx

### 🔄 数据完整性
- 用户: 1 → 1（无丢失）
- 渠道: 1 → 1
- Token: 3 → 3
- 日志: 6 → 6

### 📦 部署详情
- 升级前 HEAD: `c19ff672`
- 升级后 HEAD: `99ebd126`
- 镜像构建: 多阶段 Docker（前端 bun + Go 编译，6.5min）
- 回滚镜像: `yuxin-api:rollback-20260726-184657`
- 数据库备份: `backups/pre-upgrade-20260726-184657/new-api-db.sql` (112K)



---

## v1.0.0-yuxin (2026-07-25)

### 新增功能

#### 第一阶段 — 基础能力

- **智能路由引擎** (`service/routing/`): 支持 4 种路由策略（优先级权重/成本优化/延迟优化/质量优先），可通过 API 或 Header 动态切换
- **MCP 协议网关** (`service/mcp/`): 自动将工具注入到 LLM 请求中，内置 3 个工具（web_search/get_current_time/calculator），支持旁路机制
- **Prometheus 可观测性** (`service/metrics/`): 全量指标采集（请求/Token/成本/渠道/缓存），Prometheus + Grafana 部署
- **公开定价 API** (`controller/public_pricing.go`): OpenRouter 兼容格式，可被 AI Agent 自动拉取
- **公开状态页** (`controller/status_page.go`): 实时渠道状态 JSON API

#### 第二阶段 — 安全与质量

- **安全合规防护** (`service/guardrail/compliance.go`): 四层安全检测
  - L1: Prompt 注入检测（8 种模式）
  - L2: PII 敏感信息检测（7 类）
  - L3: 内容审核（5 大类别）
  - L4: 速率限制（per-user/per-ip）
- **Canary 质量监控** (`service/canary/engine.go`): 5 个标准测试用例，渠道健康度评分
- **模型广场 API** (`controller/phase2.go`): 模型列表+价格+能力

#### 第三阶段 — 前端与管理

- **状态页 HTML** (`controller/page_handlers.go`): 实时渠道状态可视化页面
- **定价页 HTML** (`controller/page_handlers.go`): 模型卡片展示+搜索过滤
- **Dashboard 聚合 API** (`controller/dashboard_overview.go`): 系统/渠道/路由/安全/MCP/Canary 聚合数据

### 基础设施

- Nginx 反向代理（限速 600r/m + WAF + 安全头）
- Docker Compose 7 容器编排
- Prometheus + Grafana 监控体系
- 运维管理脚本 (`manage.sh`)

### 验收结果

- 技术验收: 33/33 通过 (100%)
- 客户验收: 24/25 通过 (96%)

### 已知限制

- 在线充值未开启（需配置支付渠道）
- 无 HTTPS（需绑定域名 + SSL）
- 无 SMTP 邮件配置
- 渠道未配置有效的上游 API Key
- ClickHouse 健康检查显示 unhealthy（不影响功能）

---

## 上游版本

- 基于 new-api (QuantumNous/new-api) v1.0.0-rc.21
- 许可证: AGPL-3.0
