# API 兼容矩阵（骨架）

> 生成方式：第 2 节路由表由脚本 `scripts/gen_api_compat.py` 从
> `router/api-router.go` 的路由注册行自动提取（2026-08-03 生成，勿手改表格）；
> 第 1 节 OpenAI 兼容清单人工对照 `router/relay-router.go` 整理。
>
> 局限（骨架版）：鉴权/限流仅识别路由注册行上显式出现的中间件，
> **组级 `.Use()` 挂载的中间件不逐行回溯**；标「待核」的条目以启动时
> 实际路由表为准。

## 1. OpenAI 兼容端点

本网关以 OpenAI 兼容接口为一等公民：客户端把 SDK/应用的 base URL
指向本网关地址即可。兼容端点挂载于 `/v1`（见 `router/relay-router.go`）：

| OpenAI 能力 | 方法 | 网关端点 | 状态 |
|---|---|---|---|
| Chat Completions | POST | `/v1/chat/completions` | 支持 |
| Completions（旧版补全） | POST | `/v1/completions` | 支持 |
| Responses API | POST | `/v1/responses`、`/v1/responses/compact` | 支持 |
| Embeddings | POST | `/v1/embeddings`（含 `/v1/engines/:model/embeddings`） | 支持 |
| Rerank | POST | `/v1/rerank` | 支持 |
| 图像生成 | POST | `/v1/images/generations` | 支持 |
| 图像编辑 | POST | `/v1/images/edits` | 支持 |
| 音频转写 | POST | `/v1/audio/transcriptions` | 支持 |
| 音频翻译 | POST | `/v1/audio/translations` | 支持 |
| 语音合成 | POST | `/v1/audio/speech` | 支持 |
| 内容审查 | POST | `/v1/moderations` | 支持 |
| Realtime（WebSocket） | GET | `/v1/realtime` | 支持 |
| 模型列表 / 模型详情 | GET | `/v1/models`、`/v1/models/:model` | 支持 |
| 搜索（alpha） | POST | `/v1/alpha/search` | 支持（alpha） |
| Edits（旧版） | POST | `/v1/edits` | 支持（旧协议） |
| Files / Fine-tunes | * | `/v1/files*`、`/v1/fine-tunes*` 等 | 未实现（占位路由，返回未实现错误） |

其他上游协议兼容：

- **Anthropic Claude**：`POST /v1/messages`（Claude 原生格式）。
- **Google Gemini**：`/v1beta/models/...` 原生格式；
  `/v1beta/openai/models` 提供 OpenAI 风格入口。
- **Midjourney**：`/mj/submit/*`、`/mj/task/*`（非 OpenAI 协议）。
- **Suno**：`/suno/submit/*`、`/suno/fetch`。
- **Playground**：`POST /pg/chat/completions`（内置调试页）。

## 2. 管理 API 路由表（router/api-router.go）

列说明：

- **鉴权要求**：取自路由注册行显式出现的 `middleware.AdminAuth()` /
  `middleware.UserAuth()`；未标注的为「无鉴权（待核）」。
- **速率限制**：仅列出注册行显式挂载的限流中间件。
- **业务类别**：admin=管理员、user=登录用户、public=公开、system=系统/监控。
| 端点 | HTTP 方法 | 鉴权要求 | 速率限制 | 业务类别 |
|---|---|---|---|---|
| `/metrics` | GET | 无鉴权（待核） | — | system |
| `/api/setup` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/setup` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/status` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/uptime/status` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/status_page` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/public/pricing` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/routing/config` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/routing/config` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/mcp/tools` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/mcp/execute` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/mcp/config` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/mcp/config` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/compliance/config` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/compliance/config` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/compliance/check` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/canary/status` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/canary/run` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/canary/enable` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/status/test` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/notice` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/user-agreement` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/privacy-policy` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/about` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/midjourney` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/home_page_content` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/pricing` | GET | HeaderNavModuleAuth | GlobalAPIRateLimit | public |
| `/api/perf-metrics/summary` | GET | HeaderNavModulePublicOrUserAuth | GlobalAPIRateLimit | system |
| `/api/perf-metrics` | GET | HeaderNavModulePublicOrUserAuth | GlobalAPIRateLimit | system |
| `/api/rankings` | GET | HeaderNavModuleAuth | GlobalAPIRateLimit | public |
| `/api/verification` | GET | 无鉴权（待核） | EmailVerificationRateLimit、GlobalAPIRateLimit | public |
| `/api/reset_password` | GET | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/reset` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/oauth/state` | POST | TryUserAuth | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/oauth/email/bind` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/oauth/wechat` | GET | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/oauth/wechat/bind` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/oauth/telegram/login` | GET | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/oauth/telegram/bind/start` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/oauth/telegram/bind/:flow_token` | GET | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/oauth/:provider` | GET | TryUserAuth | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/ratio_config` | GET | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/stripe/webhook` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/creem/webhook` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/waffo/webhook` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/waffo-pancake/webhook/:env` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/verify` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/user/auth/refresh` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/auth/logout` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/register` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/login` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/login/2fa` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/passkey/login/begin` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/passkey/login/finish` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/tokenlog` | POST | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/user/epay/notify` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/user/epay/notify` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/user/groups` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/user/sessions` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/sessions/:sid` | DELETE | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/sessions/revoke-others` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/self/groups` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/self` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/models` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/self` | PUT | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/user/self` | DELETE | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/token` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/passkey` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/passkey/register/begin` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/passkey/register/finish` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/passkey/verify/begin` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/passkey/verify/finish` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/passkey` | DELETE | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/aff` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/topup/info` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/topup/self` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/topup` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/user/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/user/amount` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/stripe/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/user/stripe/amount` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/creem/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/user/waffo/amount` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/waffo/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/user/waffo-pancake/amount` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/waffo-pancake/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/user/aff_transfer` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/setting` | PUT | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/2fa/status` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/2fa/setup` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/2fa/enable` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/2fa/disable` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/2fa/backup_codes` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/checkin` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/checkin` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/oauth/bindings` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/oauth/bindings/:provider_id` | DELETE | UserAuth | GlobalAPIRateLimit | user |
| `/api/user/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/topup` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/topup/complete` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/search` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/:id/oauth/bindings` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/:id/oauth/bindings/:provider_id` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/:id/bindings/:binding_type` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/:id` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/manage` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/` | PUT | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/:id` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/:id/reset_passkey` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/2fa/stats` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/user/:id/2fa` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/plans` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/subscription/self` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/subscription/self/preference` | PUT | UserAuth | GlobalAPIRateLimit | user |
| `/api/subscription/balance/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/subscription/epay/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/subscription/stripe/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/subscription/creem/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/subscription/waffo-pancake/pay` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/subscription/admin/plans` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/plans` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/plans/:id` | PUT | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/plans/:id` | PATCH | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/bind` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/plans/:id/subscriptions/reset` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/users/:id/subscriptions` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/users/:id/subscriptions` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/users/:id/subscriptions/reset` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/user_subscriptions/:id/invalidate` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/admin/user_subscriptions/:id` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/subscription/epay/notify` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/subscription/epay/notify` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/subscription/epay/return` | GET | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/subscription/epay/return` | POST | 无鉴权（待核） | GlobalAPIRateLimit | public |
| `/api/option/` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/` | PUT | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/payment_compliance` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/channel_affinity_cache` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/channel_affinity_cache` | DELETE | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/rest_model_ratio` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/waffo-pancake/catalog` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/waffo-pancake/pair` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/waffo-pancake/save` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/waffo-pancake/subscription-product` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/option/waffo-pancake/subscription-product-options` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/custom-oauth-provider/discovery` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/custom-oauth-provider/` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/custom-oauth-provider/:id` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/custom-oauth-provider/` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/custom-oauth-provider/:id` | PUT | RootAuth | GlobalAPIRateLimit | admin |
| `/api/custom-oauth-provider/:id` | DELETE | RootAuth | GlobalAPIRateLimit | admin |
| `/api/performance/stats` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/performance/disk_cache` | DELETE | RootAuth | GlobalAPIRateLimit | admin |
| `/api/performance/reset_stats` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/performance/gc` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/performance/logs` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/performance/logs` | DELETE | RootAuth | GlobalAPIRateLimit | admin |
| `/api/ratio_sync/channels` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/ratio_sync/fetch` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/token/` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/token/search` | GET | UserAuth | GlobalAPIRateLimit、SearchRateLimit | user |
| `/api/token/auto-groups` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/token/:id` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/token/:id/key` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/token/` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/token/` | PUT | UserAuth | GlobalAPIRateLimit | user |
| `/api/token/:id` | DELETE | UserAuth | GlobalAPIRateLimit | user |
| `/api/token/batch` | POST | UserAuth | GlobalAPIRateLimit | user |
| `/api/token/batch/keys` | POST | UserAuth | CriticalRateLimit、GlobalAPIRateLimit | user |
| `/api/usage/token/` | GET | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/redemption/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/redemption/search` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/redemption/:id` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/redemption/` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/redemption/` | PUT | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/redemption/invalid` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/redemption/:id` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/log/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/log/stat` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/log/self/stat` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/log/channel_affinity_usage_cache` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/log/search` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/log/self` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/log/self/search` | GET | UserAuth | GlobalAPIRateLimit、SearchRateLimit | user |
| `/api/system-task/log-cleanup` | POST | RootAuth | GlobalAPIRateLimit | admin |
| `/api/system-task/list` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/system-task/current` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/system-task/:task_id` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/system-info/instances` | GET | RootAuth | GlobalAPIRateLimit | admin |
| `/api/system-info/stale-instances` | DELETE | RootAuth | GlobalAPIRateLimit | admin |
| `/api/system-info/instances/:node_name` | DELETE | RootAuth | GlobalAPIRateLimit | admin |
| `/api/data/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/data/users` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/data/self` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/data/flow` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/data/flow/self` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/admin/audit-logs` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/log/token` | GET | 无鉴权（待核） | CriticalRateLimit、GlobalAPIRateLimit | public |
| `/api/group/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/prefill_group/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/prefill_group/` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/prefill_group/` | PUT | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/prefill_group/:id` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/mj/self` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/mj/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/task/self` | GET | UserAuth | GlobalAPIRateLimit | user |
| `/api/task/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/vendors/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/vendors/search` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/vendors/:id` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/vendors/` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/vendors/` | PUT | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/vendors/:id` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/sync_upstream/preview` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/sync_upstream` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/missing` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/search` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/:id` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/` | PUT | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/models/:id` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/settings` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/settings/test-connection` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/search` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/test-connection` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/hardware-types` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/locations` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/available-replicas` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/price-estimation` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/check-name` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/:id` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/:id/logs` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/:id/containers` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/:id/containers/:container_id` | GET | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/:id` | PUT | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/:id/name` | PUT | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/:id/extend` | POST | AdminAuth | GlobalAPIRateLimit | admin |
| `/api/deployments/:id` | DELETE | AdminAuth | GlobalAPIRateLimit | admin |

> 合计 **243** 条路由：admin 120、user 72、public 48（含待核）、system 3。
