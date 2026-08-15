# Changelog

## v1.2.10-yuxin (2026-08-15)

安全加固版: 支付密钥静态加密 + 工具链 CVE 修复 + 监控/健康检查补齐。

### 安全
- DB options 支付密钥静态加密: AlipayPrivateKey / WechatPrivateKey / WechatApiV3Key
  采用与 channels.key 一致的 AES-256-GCM 方案 (CRYPTO_SECRET 派生, enc1: 前缀)。
  启动迁移 MigrateSensitiveOptionsToEncrypted 幂等执行; 读路径统一解密,
  写路径统一加密; 内存 OptionMap 保持明文, 管理面板行为不变。
  新增单元测试 3 项 (round-trip / 幂等 / 明文透传)。
- Go 工具链 1.25.12 -> 1.26.6-alpine (digest pin), 修复 govulncheck 命中的
  7 项可达标准库漏洞 (net/url, crypto/tls, net/http, encoding/xml, encoding/asn1, idna)
- golang.org/x/image 升级, 修复 webp 解码漏洞 (GO-2026 系列)

### 运维
- 告警真实落地: alert-forwarder 容器化 (gateway-net), alertmanager webhook
  指向 alert-forwarder:5001; 告警 JSONL 持久化至 backups/alerts/ 并随每日
  异地备份拉取; 支持钉钉/企微/飞书 webhook 扇出 (webhooks.list 配置即生效)
- 监控覆盖 5/9 -> 9/9: 新增 prometheus/alertmanager/grafana 自监控 +
  ClickHouse 内置 Prometheus 指标 (:9363, config.d/prometheus.xml)
- 健康检查补齐: nginx/redis/wechat-server/prometheus/grafana/alertmanager
- 异地备份: Kali 接收端 systemd timer 每日 04:10 拉取 backups/ + 配置快照
- 恢复演练通过: pg dump 还原 35 表 (users 11/tokens 24/channels 1/abilities 10),
  ClickHouse BACKUP/RESTORE 全链路 (logs 591 行)
- admin access_token 生成 (id=1, API 验证 200), 凭据存 /root/.config/.secrets/

### 指纹
- nginx sub_filter 版本脱敏同步 1.2.10-yuxin

### 待办 (需外部输入)
- 告警外部接收地址 (钉钉/企微 webhook 或外部邮箱, 配置即生效)
- 注册开放策略决策 (Turnstile/邮箱验证, 需 Cloudflare/SMTP 凭据)
- HA 与 AGPL 商业授权谈判 (v1.3.0 backlog)


## v1.2.9-yuxin (2026-08-15)

市场上线就绪审计 + 修复版。

### 合规
- AGPL §13: 公开源码仓同步至 v1.2.9 快照 (GitHub yuxin-api-source, c5a75685)
- AGPL §7(b): 7 locale attribution 补英文原串 "Frontend design and development by New API contributors."
- 新增隐私政策与用户协议内容 (legal.privacy_policy / legal.user_agreement)

### 功能修复
- /docs 路由修复: docs/index.tsx 缺 createFileRoute 导出导致路由从未注册(404)；示例域名 your-domain.com → ai.yuxin.yun
- Pricing/模型广场/排行榜对未登录访客开放 (HeaderNavModules requireAuth=false)
- 清除测试公告 "111111"

### 安全/运维
- /api/status 版本指纹脱敏 sub_filter 同步 (v1.2.7 → 1.2.9-yuxin)
- ClickHouse 备份双根因修复: 脚本密码变量 CH_PASS 修正 + backups 磁盘配置持久化(observability/clickhouse/backup-disk.xml)
- 监控覆盖 1/9 → 5/9 组件 (postgres/redis/node/nginx exporter + prometheus 抓取)
- 告警规则 4 → 10 条 (新增 infra.yml: 各组件 down + 内存 + 磁盘预测)
- wechat-server 镜像 pin digest
- 归档 nginx/ssl 6 个自签 .bak
- docker prune 回收 4.46GB

### 待办 (需外部输入)
- 告警真实接收渠道 (alertmanager webhook 仍指 127.0.0.1:5001)
- 备份异地副本 (rclone/OSS 端点待提供)
- DB options 支付密钥应用层加密
- 注册开放策略决策 (Turnstile/邮箱验证)

# 豫鑫 API 中转站 — 变更日志

---

## v1.2.8-yuxin (2026-08-09) · 卫生清理与可用性验收版

### 背景

v1.2.7-yuxin 收款闭环加固后的交付后运维整理——全面侦查 + 卫生清理 + 真实浏览器可用性验收 + 版本封箱。

### 卫生清理（回收 26GB）

- /tmp 清理：删除 1.0GB 临时调试产物（17 个重复二进制 *.bin 共 ~1.9GB、80+ 个一次性脚本已归档、调试前端产物、小标记文件）
- 项目 .bak 清理：9 个 docker-compose bak、1 个 Dockerfile bak、3 个 new-api-custom 二进制 bak（378MB）、7 个 nginx config bak——全部 git 有历史
- 2 个野容器移除：tender_jemison / distracted_meninsky（yuxin-rebuild-20260807 / yuxin-paytweak-20260807，43h 未挂端口、未挂卷、不在 compose）
- 37 个历史 docker 镜像清理：v1.2.1~v1.2.6 系列、c1-fix-v4~v9、audit-fixes、reconcile、real、wallet-fix、clean、fix-types、freshdist、rebuild、wechat-cert、paytweak、merge 系列、rollback 系列、pre- 回滚、test、latest
- 保留：v1.2.7-yuxin（生产）+ v1.2.6-yuxin（回滚兜底）
- Docker build cache prune：回收 19.6GB（docker builder prune -f）
- 敏感文件 shred：.env.bak.20260803212653 + .env.bak.20260803213317（含密钥前驱值）、/tmp/pub_in.pem + /tmp/paytest-cookies.txt

### 可用性验收（Playwright + Chrome DevTools）

- 域名 https://ai.yuxin.yun 外网可达，TLS 链验证成功（ssl_verify_result=20）
- /api/status 200 OK，返回完整 HeaderNav/Sidebar/announcements/api_info/chats 数据
- 首页 200 OK：Home/Model Square/Rankings + Cherry Studio/CC Switch 入口 + PlayGround（Chat/Responses/Claude/Gemini） + 200 ok 实时状态
- Dashboard 已登录会话渲染：Overview 卡片 + Get started 引导 + Usage/Last 24h/Historical/Request Count + Credit remaining + API Info + Announcements + FAQ
- API Keys 页面：表格/筛选/Status/View/Create 按钮齐备
- Wallet 页面：Current Balance/Total Usage/API Requests + 充值套餐（10/20/50/100/200/500）+ 微信支付 + 支付宝 + 自定义金额 + 兑换码 + 基础套餐 $99 + 推荐计划
- Pricing：10 个模型（deepseek-v4-flash/pro、glm-5.1/5.2、MiniMax-M2.5、qwen3.5-plus/3.6-flash/3.6-plus 等）+ 完整筛选（group/vendor/tag/pricing type/endpoint type）
- 数据库真实存在：1 channel、11 users、24 tokens

### 安全观察（非阻断）

- Console error（1 条）：ccswitch.io/favicon.png ERR_BLOCKED_BY_RESPONSE（第三方 favicon 跨域策略，与业务无关）
- 工作区状态：git status 干净，无未提交改动
- 受保护镜像（v1.2.7/v1.2.6）生产容器仍 (healthy) Up 21h

### 归档与回滚

- 所有清理项已归档至 /root/backups/cleanup-20260809/（tmp-scripts/75 个脚本 + tmp-logs/72 个日志 + project-bak/10 个配置 bak + nginx-bak/）+ VERSION.pre-v1.2.8
- Docker 镜像回滚：yuxin-api:v1.2.6-yuxin 保留

---


## v1.2.1-yuxin (2026-08-04) · 交付前自检封箱版

### 背景

豫鑫 new-api 交付前全面自检（4 维度并行审计：代码与测试/安全合规/运维可靠性/交付物与业务），定位 13 项阻断问题。本版本封箱交付：提交所有未入 git 改动、重建二进制与镜像、加 USER 降权、文档版本对齐、重跑验收归档。

### 🔒 安全加固

- **B2**: `.gitignore` 补凭据边界红线（nginx/ssl 私钥/letsencrypt/node_modules）
- **B5**: Dockerfile 容器降权——新增 `newapi` 用户（uid/gid 65532）+ USER 指令，杜绝容器逃逸=宿主 root
- **B3**: 渠道密钥 AES-256-GCM 透明加密功能编入运行产物（此前二进制漂移 1.5 天未含）
- **新增**: `.env` 补 `CRYPTO_SECRET`（强随机值），避免重启随机化导致历史渠道密钥无法解密

### 📦 交付封箱

- **B1**: 工作区脏改动分类提交（6 个逻辑 commit：gitignore/依赖/渠道加密/HTTPS/测试/文档）
- **B4**: `docker-compose.yml` 镜像引用锁 `yuxin-api:v1.2.1-yuxin` 替代 `:latest` 漂移
- **B7**: 4 份核心文档版本号统一到 v1.2.1（QUICKSTART/DEPLOYMENT/OPERATIONS/ACCEPTANCE）
- **B8**: 重跑验收并归档 `evidence/2026-08-04-v1.2.1/`（10 PASS / 2 FAIL 均非代码 bug）

### 🔧 依赖与工具链

- Go 工具链 `1.25.1 → 1.25.12` 对齐 Dockerfile
- 前端 `dompurify 3.4.11 → ^3.4.12`、overrides `brace-expansion/fast-uri/hono` 安全升级
- CI 工作流优化：gofmt 独立 job、backend 拆分 vet/build/test、push main 触发

### 🐛 修复

- **Dockerfile**: `useradd` 前先 `groupadd` 建组避免 gid 不存在错误
- **middleware/audit.go**: PATCH 方法审计动作 `update → patch`（stash 恢复）

### ✅ 验证

- 全量 `go build` / `go vet` / `go test ./...` 通过
- Docker 镜像构建成功（含 B5 降权）
- 容器内 `id = uid=65532(newapi)` 非 root
- 健康端点 `/api/status` 200
- 验收脚本 10/12 PASS

---

## v1.2.0-yuxin (2026-08-03)

### 🛡️ 商用化深化 + 可靠性层（生产级）

#### 1. 统一可靠性层（熔断 + 重试 + fallback）
- 每上游 channel 一个熔断器（sony/gobreaker v1.0.0），连续失败 5 次开路、30 秒半开探测
- 同 channel 瞬时错误指数退避重试（默认 2 次、200ms→2000ms），瞬时判定与既有 shouldRetry 语义一致
- 熔断/重试失败时复用既有 GetRandomSatisfiedChannel 外层循环自动切下一优先级 channel，不另造路由
- config-gated：setting.Enabled=false 默认，OFF 路径与改造前字节级一致，三子开关独立
- 计费安全：重试只发生在 client.Do 失败（未调 DoResponse/未扣费），每次成功 client.Do 对应一次扣费
- 单点接入：relay/channel/api_request.go:doRequest（所有请求类型共用的上游 HTTP 调用收口点）

#### 2. Admin 操作审计日志
- 新增 model/admin_audit.go（AdminAuditLog 表，GORM 迁移）
- 扩展 middleware/audit.go：写操作记录管理员变更（action/target/旧值/新值/IP/状态码/成败），敏感字段脱敏，gopool 异步落库不阻塞请求
- 新增查询端点 GET /api/admin/audit-logs（admin 鉴权，分页/按 action/时间/操作者过滤）
- PATCH 动词修正：PATCH/PUT 统一映射为 update（原误为 patch）

#### 3. Grafana 运营看板
- 新增 dashboard-operations.json（11 面板：总成本/成功率/缓存命中率/每日成本按模型/渠道调用量/延迟 P50-P99/渠道错误率矩阵）
- ClickHouse 数据源 provisioning（明细粒度走 ClickHouse；Prom 仅供总量），SQL 全部实测通过

#### 4. CI/CD 强化
- ci.yml 加 gofmt / go vet / go test -race / -coverprofile / gitleaks / Trivy 闸口
- 新增 dependabot.yml（gomod + github-actions 月度）+ trivy.yml（fs + image CVE 扫描）
- 全仓 gofmt 格式化（15 个文件字段对齐/缩进统一，无逻辑变更）

#### 5. 凭据依赖项产物（客户拿即用）
- AGPL-3.0 合规方案与商业许可谈判 brief（Path A 对接 support@quantumnous.com，748 commits 回授筹码）
- HTTPS 上线 runbook + nginx 443 配置模板（域名落地即用）
- HA 迁移方案（多副本→双机→多 AZ 三阶段）
- 支付充值接入设计（支付宝/微信/Stripe 三通道 + 幂等对账）

### 📊 验证证据
- go build / go vet / gofmt -l / go test ./... 全量绿（race 无新增失败）
- 可靠性层 12 个测试函数全 PASS；审计测试全 PASS
- 34 项变更 diff 密钥扫描 0 命中（仅 env 占位与格式化）

### ⚠️ 遗留/待办
- 可靠性层默认 OFF，需显式开启（reliability.enabled=true）后生效
- Grafana dashboard 需 grafana-clickhouse-datasource 插件
- 9 渠道 Claude 响应侧端到端（客户凭据，staging 自测）
- alertmanager webhook 接通真实通知通道；CORS 裸 IP 移除（域名落地后）
- 智能路由升级（v1.3.0 backlog）；HA 多机实施；充值通道实接（均需外部依赖）

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

