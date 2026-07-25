# 豫鑫 API 中转站 — 变更日志

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
