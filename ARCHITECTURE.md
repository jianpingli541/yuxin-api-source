# 豫鑫 API 网关 系统架构文档

> 版本：v1.2.21-yuxin-guide-drawer · 更新日期：2026-08-19
> 访问地址：https://ai.yuxin.yun
> 本文所有结论均经真实命令/浏览器实测验证（实测日期 2026-08-19），非文档推断。

---

## 一、系统定位

豫鑫 API 网关是一个 **AI 模型统一接入平台**（基于 new-api 二次开发，AGPL-3.0）。
一个账号 + 一个 API Key，通过 OpenAI 兼容接口调用多家上游大模型，统一计费、统一监控。

---

## 二、部署架构（14 容器，docker-compose 编排）

```
                        ┌─────────────────────────────────────────┐
   终端用户/管理员浏览器 │           https://ai.yuxin.yun           │
                        └──────────────┬──────────────────────────┘
                                       │ 443 (TLS)
                        ┌──────────────▼──────────────────────────┐
                        │  gateway-nginx (nginx:alpine)            │
                        │  80→443 / TLS终止 / 静态缓存 / :8081监控  │
                        └──────────────┬──────────────────────────┘
                                       │ proxy_pass http://new-api
                        ┌──────────────▼──────────────────────────┐
                        │  gateway-new-api (yuxin-api:v1.2.21)     │
                        │  Go(gin) 单体 · :3000 · 前端embed进二进制 │
                        └──┬───┬───┬───┬───┬──────────────────────┘
            ┌──────────────┘   │   │   │   └──────────────┐
   ┌────────▼──────┐  ┌────────▼┐  │   │            ┌─────▼──────────┐
   │ postgres:16   │  │ redis:7 │  │   │            │ 外部上游模型     │
   │ 业务数据       │  │缓存/限流│  │   │            │ DeepSeek/GLM/   │
   └───────────────┘  └─────────┘  │   │            │ Qwen/Volc 等    │
                        ┌──────────▼┐  │
                        │clickhouse │  │  ┌────────────────────┐
                        │ 调用日志   │  └──│ wechat-server      │
                        └───────────┘     │ justsong三方微信登录 │
                                          └────────────────────┘
   ┌─ 观测栈 (docker-compose.observability.yml, 独立 gateway-net) ─┐
   │ prometheus ← node/redis/postgres/nginx 4 exporter + new-api   │
   │ grafana(看板) · alertmanager → alert-forwarder(webhook)       │
   └────────────────────────────────────────────────────────────────┘
```

### 容器清单（14 个，全部实测在跑）

| 容器 | 镜像 | 职责 |
|---|---|---|
| gateway-new-api | yuxin-api:v1.2.21-yuxin-guide-drawer | 核心业务单体（前端+API+计费+调度）|
| gateway-nginx | nginx:alpine | TLS 终止 / 反代 / 静态缓存 |
| gateway-postgres | postgres:16-alpine | 业务数据（用户/密钥/渠道/订单）|
| gateway-redis | redis:7-alpine | 缓存 / 限流 |
| gateway-clickhouse | clickhouse-server:24.8-alpine | 调用日志（明细对账）|
| gateway-wechat-server | justsong/wechat-server | 微信登录 OAuth 组件 |
| gateway-prometheus | prom/prometheus | 指标采集 |
| gateway-grafana | grafana/grafana | 可视化看板 |
| gateway-alertmanager | prom/alertmanager | 告警分发 |
| gateway-alert-forwarder | api-gateway-alert-forwarder | 告警 webhook 转发 |
| gateway-node-exporter | prom/node-exporter | 主机指标 |
| gateway-redis-exporter | oliver006/redis_exporter | Redis 指标 |
| gateway-postgres-exporter | postgres-exporter | PG 指标 |
| gateway-nginx-exporter | nginx-prometheus-exporter | Nginx 指标 |

### 网络与存储
- 网络：`api-gateway_gateway-net`（业务）+ `gateway-net`（观测，独立）
- 数据卷：pg_data / redis_data / clickhouse_data / prometheus_data / grafana_data / alertmanager_data / wechat_server_data

---

## 三、核心服务机制（已实证到代码级）

| 层 | 机制 | 证据 |
|---|---|---|
| 前端 | React19 + rsbuild SPA，`go:embed web/dist` 编进 Go 二进制，new-api:3000 直接服务 | main.go:42 |
| API | 同进程 gin 路由，OpenAI 兼容 `/v1/*` + 管理 `/api/*` | router/*-router.go |
| 认证 | **双制**：浏览器 session（httpOnly cookie JWT）+ API PAT（Bearer sk-）；`authHelper(minRole)` 分级 TryUser/User/Admin/Root | middleware/auth.go:78-104 |
| 角色 | `role` 字段：1=用户，10=管理员，100=超管(Root)；侧边栏+路由守卫按此分流 | use-sidebar-view.ts:56 |
| 反代 | nginx :443 多 location 全 proxy_pass new-api；:8081 stub_status 供监控 | nginx/conf.d/gateway.conf |
| 微信 | 登录 OAuth `/api/oauth/wechat` + 支付 Native 扫码 + 回调 `/api/wechat/notify`，在 new-api 内；wechat-server 是独立三方登录组件 | controller/wechat.go |

---

## 四、请求处理流程（一条 API 调用完整链路）

```
客户端(sk- 或 session)
 → nginx :443 (TLS)
 → new-api gin 中间件链:
     限流(Redis) → 认证(authHelper) → 分组/配额校验(distributor)
     → 智能调度(model:auto → balanced双维选模型+渠道, X-Yuxin-Route头)
     → 渠道选择(优先级/权重/故障转移/重试) → 模型映射
 → 转发上游模型厂商
 → 响应回传(带 X-Yuxin-Model 实际生效模型头)
 → 计费(输入token×单价 + 输出token×单价) ×分组倍率 扣额度
 → 日志写 ClickHouse + 指标进 Prometheus
```

---

## 五、客户端 vs 管理端（同进程同入口，按 role 分流）

| 维度 | 客户端（role=1） | 管理端（role≥10/100） |
|---|---|---|
| 入口 | 同一 ai.yuxin.yun | 同一入口，登录后多 Admin 组 |
| 侧边栏 | 聊天(游乐场/聊天) / 常规(概览/数据看板/API密钥/使用日志/任务日志) / 个人(钱包/个人资料/使用指南) | +Admin 组(渠道/模型/用户/兑换码/订阅/系统信息*/系统设置) |
| 使用指南 | 抽屉 → USER-HELP（9章）| 抽屉 → ADMIN-HELP（13章）|
| API 权限 | 调 /v1/*、查自己数据 | +管理 /api/* 全量(AdminAuth/RootAuth) |
| 系统信息页 | 不可见 | 仅 role=100（Root 标记）|

> 使用指南角色分流抽屉：v1.2.21 实现——点侧边栏「使用指南」不跳新页面，右侧抽屉按角色内嵌对应文档（GuideDrawer + zustand + Sheet）。

---

## 六、页面 / 功能 / API 实测清单

> 以下实测结果由真实浏览器(Playwright) + curl 于 2026-08-19 验证。

### 客户端页面（普通用户 guideverify1341 实测，Playwright）

| 页面 | 路径 | 状态 | 实测要点 |
|---|---|---|---|
| 游乐场 | /playground | ⚠️部分 | 渲染正常、30+模型可选；**$0 余额发消息被 403 拦截**（计费前置有效，但新用户无法体验模型）|
| 概览 | /dashboard/overview | ✅ | 各卡渲染、$0 空态正常 |
| 数据看板 | /dashboard/models | ✅ | 5统计卡+图表区渲染（空数据态）|
| API 密钥 | /keys | ✅ | 列表+创建表单正常；⚠️ 1条key却显示"Go to page 11"分页异常 |
| 使用日志 | /usage-logs/common | ✅ | 筛选/表格渲染正常 |
| 任务日志 | /usage-logs/task | ✅ | 双tab+空态文案正常 |
| 钱包 | /wallet | ✅ | 充值档位(10-500 USD×7.3)/微信支付宝/兑换码/订阅卡/推荐奖励全渲染 |
| 个人资料 | /profile | ✅ | 5卡（绑定/2FA/Passkey/会话/语言）全渲染 |
| 使用指南抽屉 | 侧边栏 | ✅ | 抽屉弹出、URL不变、9章用户文档 |
| 首页/模型广场/排行榜 | / /pricing /rankings | ✅ | 公开页全部 200 渲染 |
| 登录/注册 | /sign-in /sign-up | ✅ | 表单完整，已登录302跳overview |

### 管理端页面（管理员 lijianping role=100 实测，Playwright）

| 页面 | 路径 | 状态 | 实测要点 |
|---|---|---|---|
| 渠道 | /channels | ✅ | 2渠道（阿里云#11/联通云#1）、测试/筛选/批量齐全；⚠️ 联通云上游 504、单模型测试~127s |
| 模型 | /models/metadata | ✅ | 73条元数据表、双tab |
| 用户 | /users | ✅ | 14用户、搜索、编辑抽屉正常 |
| 兑换码 | /redemption-codes | ✅ | 2码在库、创建抽屉字段齐全 |
| 订阅 | /subscriptions | ✅ | 基础套餐$99在列（仅1套餐，无低价引流）|
| 系统信息 | /system-info | ⚠️ | 心跳/Root标记/系统任务正常；**版本显示1.2.20-yuxin（应1.2.21，见版本漂移）**|
| 系统设置(7板块) | /system-settings/* | ✅ | 站点/身份验证/计费/模型路由/安全/控制台内容/运维 各抽1子页全加载 |
| 使用指南抽屉 | 侧边栏 | ✅ | 抽屉弹出、URL不变、**13章管理文档**（与用户端不同）|
| 管理指南入口 | 侧边栏 | ✅ | 已删除（v1.2.21 预期行为）|

### API 实测（curl 真实调用）

| 类别 | 端点 | 状态 | 结论 |
|---|---|---|---|
| 公开 | / /pricing /rankings /sign-in /sign-up /user-agreement /privacy-policy | 200 | 全部正常 |
| 公开 | /api/status | 200 | 50+配置可读；version 上报 1.2.20（漂移）|
| 鉴权 | /v1/models 无token | **401** | 拦截有效 |
| 鉴权 | /api/user/self 无token | **401** | 拦截有效 |
| 鉴权 | /api/channel/ 用户token | **403** | 越权拦截有效 |
| 鉴权 | /api/channel/ 管理员token | 200 | 管理接口正常 |
| 鉴权 | /api/user/1 用户token查他人 | **403** | 水平越权拦截有效 |
| 鉴权 | PAT打管理面 /api/user/1 | **401** | 令牌类型隔离有效 |
| 计费 | /v1/chat/completions $0用户 | **403** insufficient_user_quota | **计费前置拦截有效** |
| 计费 | /v1/chat/completions model=auto 充值用户 | 200 | **智能调度真实成功** auto→deepseek-v4-flash，X-Yuxin-Model头存在 |
| 计费 | /api/log/ 计费核验 | 200 | quota=45(9+21 token)与usage完全一致 |
| 模型 | /v1/models 带token | 200 | 返回**239个模型**（含内部测试模型，见安全隐患）|

---

## 七、代码流 / 数据流 / 运维流

### 代码流（唯一真相源）
```
feifei /root/projects/api-gateway (开发源, main)
  → scripts/release.sh <tag>
     (bun构建前端 → go build → docker镜像 → AGPL披露推GitHub → compose换tag → 健康检查)
  → 本机 /root/work/api-gateway (git fetch feifei && merge --ff-only)
  → git push uxin (GitHub备份 jianpingli541/uxin-api-gateway)
```
- AGPL 披露仓：jianpingli541/yuxin-api-source（release.sh 自动推送）
- ⚠️ 仓库陷阱：`/root/yuxin-api-source-snapshot`（feifei）是 AGPL 单层披露快照，**非开发源**，改它无效。

### 数据存储
| 存储 | 内容 |
|---|---|
| postgres | 用户/密钥/渠道/订单/兑换码/订阅/系统设置 |
| redis | 缓存/限流计数 |
| clickhouse | API 调用日志（token/费用明细）|
| prometheus | 时序指标 |

### 运维入口
- `cd /root/projects/api-gateway && docker compose ps/logs`
- `./manage.sh`（运维脚本）
- 备份：BACKUP.md · 运维手册：OPERATIONS.md · 部署：DEPLOYMENT.md

---

## 八、安全现状

实测安全观察（2026-08-19，已抽核坐实）：

| 级别 | 问题 | 实测证据 |
|---|---|---|
| 🔴 高 | **管理员（Root role=100）登录未强制 2FA** | `POST /api/user/login?turnstile=` 仅凭密码返回 200+role=100，无 require_2fa 字段；turnstile 空 token 也放行 |
| 🔴 高 | **Root 密码 = Gmail 地址同串**（弱凭据）| 历史遗留，已存档 secrets；叠加 2FA 未强制 = 单弱密码即可全控 |
| 🟡 中 | **注册无人机/邮箱验证** | `turnstile_check:false`+`email_verification:false`+`register_enabled:true`，存在批量注册 farming 面 |
| 🟡 中 | **/v1/models 暴露 239 个模型** | 含 `sre-gpu-auto-handle`/`test-sre-gpu-auto-handle` 等内部运维模型，信息泄露+误调用计费风险 |
| 🟡 中 | **登录限流按 IP，同机 secondary IP 可旁路** | 实测 172.252.225.219 可绕过已限流 IP 继续尝试 |
| 🟢 低 | /api/rankings 公开全站用量份额 | `requireAuth:false`，竞对可免费读经营数据 |
| 🟢 低 | /api/status 信息面偏大 | 50+ 配置项（汇率/单价/rp_id）对外可读 |
| ✅ 良好 | 计费前置拦截 / 鉴权三级(401/403) / PAT与dashboard-JWT类型隔离 / token key掩码+scope取回 / CSP+HSTS+COOP+COEP | 均实测通过，未发现可利用越权 |

**版本漂移（流程缺陷，已定位根因）**：部署镜像 v1.2.21，但 `/api/status` 与系统信息页上报 `1.2.20-yuxin`。根因：go build 经 `-ldflags -X common.Version=$(cat VERSION)` 注入版本，上次发布**先 build 后改 VERSION**，顺序颠倒。VERSION 文件必须在 build 前定稿（release.sh 应内化此约束）。

---

## 九、优化方案（更便利 / 更智能 / 更自动化）

> 每条锚定 2026-08-19 实测发现的真实痛点 + 开源对标，按优先级排序。

### 🔴 P0 安全底线
1. **强制管理员 2FA + 开 Cloudflare Turnstile** — 实测 Root 弱密码即可全控；后端对 role>=10 强制 require_2fa + 开 turnstile_check 配真实 sitekey + root 改强密码（new-api 原生，零开发）
2. **收敛 /v1/models 对外暴露面** — 实测 239 模型含内部 sre-gpu-auto-handle；用模型分组把对外 default 组收敛到约 10 个商用模型（原生 group 机制，纯配置）

### 🟠 P1 转化与体验
3. **新用户免费体验额度** — 实测 $0 用户游乐场 403 阻断转化链；quota_for_new_user 赠送小额体验额度 + 充值 CTA（原生配置项）
4. **使用指南公开化** — 实测未登录无帮助入口(docs:false)；首页/登录页挂 /help 公开路由（已有路由，仅挂导航）

### 🟡 P2 运维自动化
5. **渠道健康自动巡检+告警** — 实测联通云 504/测试127s；利用已有定期渠道测试 + alertmanager 加失败率/延迟告警规则（prometheus 原生）
6. **自动对账+异常计费告警** — Grafana 已有收入/用量看板，加单用户突增/渠道失败率告警（Grafana Alerting 原生）
7. **release.sh 内化版本纪律** — 实测版本漂移(部署1.2.21/上报1.2.20)；release.sh 开头先写 VERSION 再 build，build 后校验二进制版本==tag（流程修正一行）

### 🟢 P3 智能化进阶
8. **智能调度效果可视化+自动调优** — auto 调度已上线但质量数据只在日志；Grafana 加 auto 调度分布图(各模型量/成功率/延迟/成本)驱动别名池调优（X-Yuxin-Model 头 + ClickHouse，零新组件）

**执行顺序**：第一波配置级零开发(#1/#2/#3/#7) → 第二波运维自动化(#5/#6) → 第三波智能化(#4/#8)
