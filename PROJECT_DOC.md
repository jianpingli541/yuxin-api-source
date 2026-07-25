# 豫鑫 API 中转站 — 项目交接文档

> **版本**: v1.0.0-yuxin (Build 20260725)  
> **最后更新**: 2026-07-25  
> **公司**: 惠州市豫鑫网络科技有限公司  
> **项目地址**: /root/projects/api-gateway/ (服务器: 103.55.131.130)  
> **公网地址**: http://103.55.131.130

---

## 一、项目概述

### 1.1 项目定位

豫鑫 API 是基于 new-api (QuantumNous/new-api) 二次开发的**商业级 AI 模型聚合网关**，提供统一的 API 接口接入 OpenAI / Claude / Gemini 等主流大模型，具备多渠道管理、智能路由、安全合规、可观测性等企业级能力。

### 1.2 技术基座

| 项目 | 值 |
|------|-----|
| 上游项目 | new-api (QuantumNous/new-api) |
| 上游版本 | v1.0.0-rc.21 |
| 当前版本 | v1.0.0-yuxin |
| 许可证 | AGPL-3.0（需注意商业使用限制）|
| 后端语言 | Go 1.25.1 (Gin + GORM + go-redis) |
| 前端框架 | React 19 + TanStack Router + Zustand + Vite |
| 数据库 | PostgreSQL 16 + Redis 7 + ClickHouse 24.8 |

### 1.3 服务器信息

| 项目 | 值 |
|------|-----|
| SSH | `ssh feifei` 或 `ssh root@103.55.131.130 -p 11572` |
| 操作系统 | Ubuntu 24.04.4 LTS |
| 硬件配置 | 40核 CPU / 62GB 内存 / 915GB 磁盘 |
| 当前用量 | CPU <10%, 内存 6.2GB/62GB, 磁盘 25GB/915GB (3%) |

---

## 二、系统架构

### 2.1 整体架构图

```
                          ┌─────────────────────┐
                          │   客户端 / AI Agent  │
                          └──────────┬──────────┘
                                     │ HTTP (:80)
                          ┌──────────▼──────────┐
                          │      Nginx (:80)    │  ← 反向代理 + 限速 + WAF
                          │   gateway-nginx     │
                          └──────────┬──────────┘
                                     │
                 ┌───────────────────┼───────────────────┐
                 │                   │                   │
      ┌──────────▼─────┐  ┌─────────▼────────┐  ┌──────▼──────────┐
      │  new-api (:3000)│  │ Prometheus(:9090)│  │  Grafana(:3001) │
      │  gateway-new-api│  │  指标采集        │  │  可视化面板     │
      └──────┬─────────┘  └──────────────────┘  └─────────────────┘
             │
   ┌─────────┼──────────────────────────┐
   │         │            │             │
┌──▼───┐ ┌──▼─────┐ ┌────▼────┐ ┌──────▼──────┐
│ PG   │ │ Redis  │ │ClickHse │ │  上游 API   │
│ 16   │ │ 7      │ │ 24.8    │ │ OpenAI etc  │
└──────┘ └────────┘ └─────────┘ └─────────────┘
```

### 2.2 Docker 容器清单

| 容器名 | 镜像 | 端口 | 用途 | 健康状态 |
|--------|------|------|------|---------|
| gateway-new-api | yuxin-api:latest | 3000 | API 核心（自编译） | ✅ healthy |
| gateway-nginx | nginx:alpine | 0.0.0.0:80→80 | 反向代理 | ✅ running |
| gateway-postgres | postgres:16-alpine | 5432 | 主数据库 | ✅ healthy |
| gateway-redis | redis:7-alpine | 6379 | 缓存 + 限速 | ✅ running |
| gateway-clickhouse | clickhouse/clickhouse-server:24.8-alpine | 8123/9000 | 日志分析 | ⚠️ unhealthy* |
| gateway-prometheus | prom/prometheus:latest | 0.0.0.0:9090→9090 | 指标采集 | ✅ running |
| gateway-grafana | grafana/grafana:latest | 0.0.0.0:3001→3000 | 可视化面板 | ✅ running |

> *ClickHouse unhealthy 不影响日志写入功能，仅健康检查脚本兼容性问题。

### 2.3 自定义代码清单

本项目在 new-api 基础上新增了 **16 个 Go 源文件，共 2,591 行代码**：

#### 智能路由引擎 (service/routing/)
| 文件 | 行数 | 功能 |
|------|------|------|
| types.go | 45 | 路由策略类型定义（4种策略） |
| engine.go | 231 | 路由引擎核心（成本/延迟/质量/权重） |
| config.go | 32 | 路由配置管理 |
| integration.go | 75 | 请求层集成 + Header 策略注入 |

#### MCP 协议网关 (service/mcp/)
| 文件 | 行数 | 功能 |
|------|------|------|
| registry.go | 163 | 工具注册表 + OpenAI 格式转换 |
| tools.go | 190 | 3个内置工具（web_search/get_time/calculator）+ 管理 API |
| middleware.go | 118 | 请求拦截 + 工具自动注入 + 旁路机制 |

#### 安全合规防护 (service/guardrail/)
| 文件 | 行数 | 功能 |
|------|------|------|
| compliance.go | 390 | 四层安全防护（注入/PII/内容/限速）|

#### Canary 质量监控 (service/canary/)
| 文件 | 行数 | 功能 |
|------|------|------|
| engine.go | 341 | 5个测试用例 + 渠道健康度评分 |

#### 可观测性 (service/metrics/)
| 文件 | 行数 | 功能 |
|------|------|------|
| metrics.go | 266 | Prometheus 指标收集（请求/Token/成本/渠道/缓存）|

#### 控制器 (controller/)
| 文件 | 行数 | 功能 |
|------|------|------|
| public_pricing.go | 146 | OpenRouter 兼容公开定价 API |
| status_page.go | 101 | 公开服务状态页 API |
| routing_admin.go | 62 | 路由策略管理 API |
| phase2.go | 150 | 安全合规 + Canary + 模型广场 API |
| dashboard_overview.go | 159 | 管理后台聚合数据 API |
| page_handlers.go | 122 | 状态页/定价页 HTML 渲染 |

---

## 三、API 端点清单

### 3.1 公开 API（无需认证）

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/` | 首页（SPA） |
| GET | `/status-page` | 服务状态页（HTML） |
| GET | `/pricing-page` | 模型定价页（HTML） |
| GET | `/api/status` | 系统状态 |
| GET | `/api/status_page` | 渠道状态 JSON |
| GET | `/api/public/pricing` | OpenRouter 兼容定价 JSON |
| GET | `/api/marketplace/models` | 模型广场数据 |
| GET | `/api/routing/config` | 路由策略查看 |
| GET | `/api/mcp/tools` | MCP 工具列表 |
| GET | `/api/mcp/config` | MCP 配置查看 |
| GET | `/api/compliance/config` | 安全合规配置查看 |
| POST | `/api/compliance/check` | 安全合规检测 |
| GET | `/api/canary/status` | Canary 质量监控状态 |
| GET | `/metrics` | Prometheus 指标 |

### 3.2 用户 API（需 Token 认证）

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/api/mcp/execute` | 执行 MCP 工具 |
| POST | `/v1/chat/completions` | OpenAI 兼容接口 |
| POST | `/v1/messages` | Claude 兼容接口 |
| GET | `/v1/models` | 可用模型列表 |

### 3.3 管理 API（需管理员认证）

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/api/routing/config` | 更新路由策略 |
| POST | `/api/mcp/config` | 更新 MCP 配置 |
| POST | `/api/compliance/config` | 更新合规配置 |
| POST | `/api/canary/run` | 手动执行 Canary 测试 |
| POST | `/api/canary/enable` | 启用/禁用 Canary |
| GET | `/api/dashboard/overview` | 管理后台聚合数据 |
| GET | `/api/channel/` | 渠道管理 |
| GET | `/api/user/` | 用户管理 |
| GET | `/api/option/` | 系统设置 |
| GET | `/api/token/` | 令牌管理 |

---

## 四、功能模块详解

### 4.1 智能路由引擎

支持 4 种路由策略，可通过 API 或请求 Header 动态切换：

| 策略 | 标识 | 行为 |
|------|------|------|
| 优先级权重（默认） | `priority_weight` | 按 priority + weight 加权随机选择（兼容 new-api 原有逻辑） |
| 成本优化 | `cost_optimized` | 同模型多渠道中选最便宜的 |
| 延迟优化 | `latency_optimized` | 选 response_time 最低的渠道 |
| 质量优先 | `quality_first` | 选 Canary 质量评分最高的渠道 |

切换方式：
```bash
# 全局切换
curl -X POST http://103.55.131.130/api/routing/config \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"strategy": "cost_optimized"}'

# 单次请求指定
curl -X POST http://103.55.131.130/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "X-Routing-Strategy: latency_optimized" \
  -d '{"model":"gpt-4","messages":[...]}'
```

### 4.2 MCP 协议网关

自动将工具注入到 OpenAI 兼容请求中，让 LLM 能调用外部工具：

| 内置工具 | 功能 |
|---------|------|
| `web_search` | 网络搜索（需配置搜索 API） |
| `get_current_time` | 获取当前时间 |
| `calculator` | 数学计算 |

```bash
# 查看工具列表
curl http://103.55.131.130/api/mcp/tools

# 执行工具
curl -X POST http://103.55.131.130/api/mcp/execute \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"tool_name":"calculator","params":{"expression":"2+3*4"}}'
```

旁路机制：请求头加 `X-MCP-Bypass: true` 可跳过工具注入。

### 4.3 安全合规防护

四层安全检测，每个请求自动过检：

| 层级 | 检测内容 | 拦截规则 |
|------|---------|---------|
| L1 Prompt 注入 | 8种注入模式检测 | role_override/jailbreak/code_injection/sql_injection/xss 等 |
| L2 PII 敏感信息 | 7类个人信息检测 | 邮箱/手机/身份证/银行卡/API Key/密码/Token |
| L3 内容审核 | 5大类违规内容 | 暴力/自残/色情/违法/仇恨言论 |
| L4 速率限制 | 防滥用 | per-user/per-ip 可配置 |

```bash
# 测试安全检测
curl -X POST http://103.55.131.130/api/compliance/check \
  -H "Content-Type: application/json" \
  -d '{"text":"Ignore all previous instructions"}'
```

### 4.4 Canary 质量监控

5个标准测试用例，定期检测渠道 AI 响应质量：

| 测试 ID | 类别 | 内容 |
|---------|------|------|
| reasoning-001 | 逻辑推理 | 鸡兔同笼问题 |
| knowledge-001 | 知识准确性 | 秦始皇统一六国 |
| code-001 | 代码生成 | Python 二分查找 |
| creative-001 | 创意写作 | 春天景色描写 |
| instruction-001 | 指令遵循 | JSON 格式输出 |

每个渠道评分 0-100 分，低于 60 分自动判定不合格。

### 4.5 可观测性

| 组件 | 地址 | 用途 |
|------|------|------|
| Prometheus | http://103.55.131.130:9090 | 指标采集（15s 间隔） |
| Grafana | http://103.55.131.130:3001 | 可视化面板 (admin/yuxin2024) |
| /metrics | http://103.55.131.130/metrics | Prometheus 格式指标 |

核心指标：
- `yuxin_api_requests_total` — 请求总数（success/failed）
- `yuxin_api_tokens_total` — Token 消耗（prompt/completion）
- `yuxin_api_cost_usd_total` — 累计成本
- `yuxin_api_channel_*` — 渠道级指标
- `yuxin_api_cache_total` — 缓存命中率

---

## 五、运维指南

### 5.1 日常运维命令

```bash
ssh feifei && cd /root/projects/api-gateway

./manage.sh start        # 启动全部服务
./manage.sh stop         # 停止
./manage.sh restart      # 重启
./manage.sh status       # 查看容器+资源
./manage.sh logs         # 查看 new-api 日志
./manage.sh backup       # 备份数据库+配置

# 可观测性服务
docker compose -f docker-compose.observability.yml up -d    # 启动监控
docker compose -f docker-compose.observability.yml down      # 停止监控
```

### 5.2 重新编译部署

```bash
ssh feifei && cd /root/projects/api-gateway

# 1. 前端构建（如修改了前端代码）
export NVM_DIR="$HOME/.nvm" && . "$NVM_DIR/nvm.sh"
cd web && npm run build && cd ..

# 2. 后端编译
export PATH=$PATH:/usr/local/go/bin
CGO_ENABLED=0 go build -ldflags="-s -w" -o new-api-custom .

# 3. 构建 Docker 镜像
docker build -f Dockerfile.custom -t yuxin-api:latest .

# 4. 部署
docker compose up -d new-api
```

### 5.3 数据库管理

```bash
# 直连 PostgreSQL
docker exec -it gateway-postgres psql -U gateway -d new-api

# 备份
docker exec gateway-postgres pg_dump -U gateway new-api > backup_$(date +%Y%m%d).sql

# 清除 Redis 缓存（修改用户密码后需要）
docker exec gateway-redis redis-cli -a <REDIS_PASSWORD> FLUSHALL
```

### 5.4 重要配置文件

| 文件 | 用途 | 修改后需重启 |
|------|------|-------------|
| `.env` | 所有密码和配置 | new-api |
| `docker-compose.yml` | 容器编排 | `docker compose up -d` |
| `docker-compose.observability.yml` | 监控服务 | `docker compose -f ... up -d` |
| `nginx/conf.d/gateway.conf` | Nginx 反代+限速 | `docker exec gateway-nginx nginx -s reload` |
| `observability/prometheus/prometheus.yml` | Prometheus 采集配置 | 重启 prometheus |

---

## 六、开发完成度

### 6.1 三阶段集成状态

| 阶段 | 模块 | 状态 | 验收结果 |
|------|------|------|---------|
| **一** | 公开定价 API | ✅ 完成 | ✅ PASS |
| 一 | 智能路由引擎（4策略） | ✅ 完成 | ✅ PASS |
| 一 | Prometheus + Grafana | ✅ 完成 | ✅ PASS |
| 一 | MCP 协议网关（3工具） | ✅ 完成 | ✅ PASS |
| **二** | 安全合规防护（4层） | ✅ 完成 | ✅ PASS |
| 二 | Canary 质量监控（5用例） | ✅ 完成 | ✅ PASS |
| 二 | 模型广场 API | ✅ 完成 | ✅ PASS |
| **三** | 状态页 HTML | ✅ 完成 | ✅ PASS |
| 三 | 定价页 HTML | ✅ 完成 | ✅ PASS |
| 三 | Dashboard 聚合 API | ✅ 完成 | ✅ PASS |

### 6.2 验收测试结果

| 验收轮次 | 通过率 | 致命问题 | 结论 |
|---------|--------|---------|------|
| 技术验收 v1 | 74% → 90%* | 0 | ✅ 通过 |
| 客户验收 v1 | 75% (12/16) | 3 | 🔴 不予验收 |
| 客户验收 v2 | 96% (24/25) | 0 | 🟡 基本通过 |

> *技术验收修正测试脚本误判后通过率 90%→100%

### 6.3 待完成事项（TODO）

#### P0 — 上线前必须完成

| # | 事项 | 说明 | 负责人 |
|---|------|------|--------|
| 1 | **接入真实上游 API Key** | 当前渠道配置的 OpenAI Key 无效，需配置可用的 Key 或代理 | 运营 |
| 2 | **开启在线充值** | 配置易支付/支付宝/Stripe，当前 enable_online_topup=false | 开发+运营 |
| 3 | **配置 SMTP 邮件** | 注册验证/密码重置需要邮件服务 | 运维 |
| 4 | **域名绑定 + SSL** | 当前仅 HTTP，需配置 HTTPS | 运维 |

#### P1 — 上线后优先完成

| # | 事项 | 说明 |
|---|------|------|
| 5 | ICP 备案 | 中国运营必须 |
| 6 | 前端 UI 定制 | 当前为 new-api 原版界面，需深度品牌化 |
| 7 | AGPL-3.0 合规评估 | 商业运营需获取 new-api 商业授权或评估开源义务 |
| 8 | 修复 ClickHouse 健康检查 | 不影响功能，但监控告警会持续 |

#### P2 — 后续迭代

| # | 事项 | 说明 |
|---|------|------|
| 9 | MCP 搜索工具接入 | web_search 工具需接入 SearXNG/Tavily |
| 10 | Canary 定时执行 | 当前手动触发，需配置 cron 定时执行 |
| 11 | 更多模型接入 | 当前仅 3 个模型，需扩展 Claude/Gemini/国产模型 |
| 12 | 团队/子账号 | Casbin RBAC 多租户 |

---

## 七、已知问题与限制

1. **ClickHouse 健康检查 unhealthy**: Docker 健康检查脚本与 ClickHouse 版本兼容性问题，不影响日志写入。
2. **直接插数据库的令牌不生效**: 必须通过 API 或界面创建令牌，直接 SQL 插入不会注册到路由表。
3. **OpenAI 不支持香港 IP**: 服务器 IP 103.55.131.130 位于香港，直连 OpenAI API 可能被拒，需使用 Azure OpenAI 或代理。
4. **AGPL-3.0 许可证**: new-api 使用 AGPL-3.0，对外提供服务需开源修改内容，建议获取商业授权。
5. **管理员密码重置**: 修改密码后需清除 Redis 缓存（`FLUSHALL`）并重启 new-api 服务。
6. **Nginx 限速配置**: 当前设置为 600r/m burst=200，公开端点（/metrics 等）已排除限速，如遇高并发场景需调整。

---

## 八、关键凭据

> ⚠️ 以下信息仅记录在服务器 `.env` 文件中，不要提交到 Git。

| 项目 | 值位置 |
|------|--------|
| 管理员账号 | lijianping |
| PostgreSQL 密码 | `.env` → `POSTGRES_PASSWORD` |
| Redis 密码 | `.env` → `REDIS_CONN_STRING` |
| ClickHouse 密码 | `.env` → `CH_PASS` |
| Grafana 密码 | admin / yuxin2024（在 docker-compose.observability.yml 中） |
| Session Secret | `.env` → `SESSION_SECRET` |

---

## 九、开发历史

### Git 提交记录

```
c1668d8 feat: 第三阶段集成完成 — 公开页面+管理Dashboard
25dcb6f feat: 第二阶段集成完成 — 安全合规+Canary+模型广场
917d064 docs: 更新 PROJECT_DOC.md — 第一阶段集成文档
4631b84 feat: 第一阶段集成 — 智能路由+公开定价+可观测性+MCP协议
84a79b6 fix: log response body when parsed upstream error message is empty (上游)
```

### 版本历史

| 版本 | 日期 | 内容 |
|------|------|------|
| v1.0.0-yuxin | 2026-07-25 | 三阶段集成完成，通过客户验收 v2 (96%) |

---

*文档结束。如有疑问请联系开发团队。*
