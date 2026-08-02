# 豫鑫 API 中转站 — 变更日志

---

## v1.1.0-yuxin (2026-08-02)

### 🚀 上游大版本合并 + 商用化加固（生产级）

#### 1. 上游 new-api 大版本合并（779 commits）
- 合并 Calcium-Ion/new-api 自 2026-03-17 分叉点以来的 779 个提交（至 2026-08-01 HEAD），merge commit `cfc35dc0e`，473 文件 +19,964/-2,567 行
- 232 个冲突按规则分流 + 逐文件评审解决；保留全部定制功能（智能路由/MCP/合规/Canary/可观测性）
- 适配上游 relaykit 包抽取重构（dto/types/relayconvert/reasonmap 迁入 relaykit 子模块）
- 兼容性修复：`model.GetGroupEnabledAbilitiesByModel` 用上游 `getChannelQuery` 重新暴露（智能路由依赖）

#### 2. 合并回归修复（`eccd65616`）
- 恢复 hotfix.1 的 `stringifyEventData`（合并时误用 `checkout --theirs` 整文件覆盖丢失）——SSE 非 string 数据 panic 修复，`TestCustomEvent_NonStringDataMustNotPanic` 恢复通过
- 修复 relay/helper 计费测试双 dto import 冲突，测试恢复可编译

#### 3. Claude 渠道适配补齐（12 个渠道）
- 消灭全部 `panic("implement me")`：cloudflare/zhipu/baidu/xunfei/mistral/dify/palm/tencent/cohere 真转换（claude→openai 请求转换），jina/mokaai/replicate 改优雅报错（本不支持 chat）
- 前端 5 个生产 typecheck 错清零（hasToolSurcharge 导出/badge warning variant/yace 类型声明）

#### 4. 安全加固（M0 + M2，商用级）
- **Grafana**: 公网 `admin/yuxin2024` → 127.0.0.1 仅内网 + 31 位强随机密码（`.env` 的 `GF_SECURITY_ADMIN_PASSWORD`），访问走 SSH 隧道
- **Prometheus/Alertmanager**: 公网 → 127.0.0.1 仅内网，禁用 `--web.enable-lifecycle`
- **`/metrics`**: nginx 层 `allow 内网; deny all`，防业务指标泄漏
- **CORS**: `AllowAllOrigins=false` + 白名单
- **session refresh TTL**: 30 天 → 7 天
- **`/api/routing/config` `/api/compliance/config` GET**: 加 `UserAuth`（防匿名侦察限流阈值）
- **`.env` / DB 备份**: 权限 644 → 600

#### 5. 运维就绪（M4，商用级）
- **自动备份**: cron 每日 02:00（PG+Redis+配置，7 天保留），脚本 `scripts/auto-backup.sh`
- **告警**: 部署 alertmanager + 4 条规则（NewAPIDown/APIHighLatency/APIErrorRateHigh/DiskUsageHigh），prometheus 接线
- **监控修复**: observability 与 main 两套 Docker 网络打通（`api-gateway_gateway-net` 共享），prometheus scrape 恢复，yuxin_* 指标采集正常
- **Grafana**: 5 个 dashboard 已加载

#### 6. 商用 scaffold（M1，待填凭据）
- `.env` 加支付（Waffo/Stripe）/SMTP/Turnstile 占位键
- 法务三件套 AI 初稿（用户协议/隐私政策/退款条款）已起草，待律师复核+填占位

### 📊 验证证据
- `go build ./...` PASS，`go vet ./...` 0 警告，前端 typecheck 生产代码 0 错
- 端到端流程：注册→登录→令牌→定价→状态页全 `success:true`
- 外网暴露收敛：3001/9090/9093 外网 000，/metrics 403
- 端到端 Claude 渠道不 panic（0 活跃 panic）

### ⚠️ 遗留/待办（凭据/决策）
- 退款 cutoff 业务决策（2025-02-22 本地 vs 2026-02-22 上游，service 包唯一测试失败项）
- 支付/上游 API key/SMTP/Turnstile 凭据待填（`.env` 占位已留）
- 域名 + HTTPS（当前裸 IP+HTTP，明文凭据，商用 P0）
- 法务初稿待律师复核（ICP 许可证/算法备案/数据出境合规红线）
- 告警通知 webhook 待填（当前占位）
- 9 个 Claude 真转换渠道仅 mistral 端到端响应适配，其余仅请求侧（不 panic）

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

---

## v1.0.0-yuxin-hotfix.2 (2026-07-27)

### 🔍 完整自检验收（无功能变更）

> **触发原因**: 客户在交付前要求 `确保所有页面+功能+工作流都正常无误`。由 Claude Code 在 feifei 生产环境实地验收，未修改任何代码。

### ✅ 验证结果

| 维度 | 通过率 | 详情 |
|---|---|---|
| 容器健康 | 7/7 ✅ | new-api / postgres / redis / nginx / clickhouse / prometheus / grafana |
| 公开 API | 9/9 ✅ | 状态/定价/路由/MCP/合规/Canary 全部 200，响应 < 10ms |
| 鉴权分级 | ✅ 正确 | 无 token 401，普通用户访问 admin 接口返回 INSUFFICIENT_PRIVILEGE |
| 前端页面 | 5/5 ✅ | `/`、`/status`、`/pricing`、`/login`、`/register` 全部 200 + 正确 Title |
| 完整工作流 | ✅ 6/6 | 注册→登录→Bearer 鉴权→`/self`→合规检测 PII 拦截→合规放行 |
| Prometheus 指标 | ✅ 5/5 | `yuxin_api_{requests,tokens,cost,cache,uptime}` 全部抓到 |
| Go 编译 | ✅ | `go build ./...` 无错误 |
| Go 单测（关键包） | ✅ 6/6 | common + 5 个扩展 service 全过 |
| go vet（关键 4 个） | ✅ 0 警告 | hotfix.1 已修复 CustomEvent 锁值传递 |
| 前端 typecheck | ✅ | `bun run typecheck` exit 0 |
| 前端 build | ✅ | `bun run build` exit 0, 57MB |
| 前端 lint | ❌ 389 errors | **新增发现，不影响功能，需修** |

### 🆕 本次发现

1. **前端 lint 失败 389 errors / 83 warnings**
   - 不阻塞 build、不影响功能
   - 主要错误：nested ternary、no-array-index-key、`web/scripts/sync-i18n.mjs` 风格问题
   - 建议交付前修复以提升代码质量

### 📦 资源使用

- 磁盘：31GB / 915GB (4%)
- 内存：6.5GB / 62GB (10%)
- 容器运行时长：new-api 21h（hotfix.1 升级至今零停机）

### 🔄 与 hotfix.1 的关系

- **代码层面**：本次无任何 commit，纯文档+evidence 同步
- **运行层面**：与 hotfix.1 完全一致
- **结论**：hotfix.1 的修复在生产中持续有效（SSE panic 未复现、go vet 保持 0 关键警告）

### 📋 终验证据

完整原始输出已保存至 `evidence/final-verify-20260727.txt` (134 行)，所有命令可在 feifei 上重放。

