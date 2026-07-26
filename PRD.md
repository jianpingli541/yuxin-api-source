# 豫鑫 API 中转站 — 产品需求文档（PRD）

> **版本**: v1.0.0-yuxin
> **客户**: 惠州市豫鑫网络科技有限公司
> **生产环境**: http://103.55.131.130 (feifei 服务器)
> **代码仓库**: https://github.com/jianpingli541/uxin-api-gateway (私有)
> **文档日期**: 2026-07-26
> **基线**: 基于 new-api v1.0.0-rc.21 二次开发
> **许可证**: AGPL-3.0（**必须告知客户**，详见"许可证风险"章节）

---

## 一、产品定位

豫鑫 API 是**商业级 AI 模型聚合网关**，提供统一 API 接口接入 OpenAI / Claude / Gemini
等主流大模型，具备多渠道管理、智能路由、安全合规、可观测性等企业级能力。

**目标用户**：通过 API 调用大模型的企业开发者、AI 应用开发者、SaaS 平台。

**商业模式**：按调用量/token 加价分销（下游用户充值 → 调用 → 上游渠道扣费 → 赚差价）。

---

## 二、技术基座

| 项目 | 值 |
|------|------|
| 上游项目 | new-api (QuantumNous/new-api) v1.0.0-rc.21 |
| 后端 | Go 1.25.1 (Gin + GORM + go-redis) |
| 前端 | React 19 + TanStack Router + Zustand + Vite |
| 数据库 | PostgreSQL 16 + Redis 7 + ClickHouse 24.8 |
| 容器 | Docker Compose（7 个服务，详见部署章节）|

---

## 三、功能范围（基于现存 ACCEPTANCE-REPORT 固化）

### L1 — 公开页面
- [L1.1] 首页加载
- [L1.2] 状态页加载（实时渠道状态）
- [L1.3] 定价页加载（模型卡片+价格）
- [L1.4] 首页无严重 JS 错误

### L2 — 认证流程
- [L2.1] 登录页可访问
- [L2.2] 注册页可访问
- [L2.3] 错误登录被正确拒绝

### L3 — 渠道管理（多上游模型渠道）
- [L3.1] 渠道列表 API 返回 200
- [L3.2] 渠道健康检查（每个渠道独立状态）
- [L3.3] 渠道优先级配置生效

### L4 — 智能路由（自定义扩展功能）
- [L4.1] 优先级权重路由（默认策略）
- [L4.2] 成本优化路由（选择最便宜可用渠道）
- [L4.3] 延迟优化路由（选择最快响应渠道）
- [L4.4] 质量优先路由（选择最高质量渠道）
- [L4.5] 通过 Header 动态切换策略

### L5 — API 链路
- [L5.1] 系统状态 API
- [L5.2] 状态页数据 API
- [L5.3] 公开定价 API（OpenRouter 兼容格式）
- [L5.4] OpenAI 兼容 `/v1/chat/completions` 端点
- [L5.5] OpenAI 兼容 `/v1/models` 端点
- [L5.6] Token 鉴权生效（无 token → 401，错 token → 401）
- [L5.7] 用量统计 API（已认证用户）
- [L5.8] 渠道余额查询 API（管理员）

### L6 — MCP 协议网关（自定义扩展）
- [L6.1] MCP 工具自动注入 LLM 请求
- [L6.2] 内置 3 个工具（web_search / get_current_time / calculator）
- [L6.3] 工具旁路机制（用户可禁用）

### L7 — 安全合规（自定义扩展）
- [L7.1] L1: Prompt 注入检测（8 种模式）
- [L7.2] L2: PII 敏感信息检测（7 类）
- [L7.3] L3: 内容审核（5 大类别）
- [L7.4] L4: 速率限制（per-user / per-ip）

### L8 — 可观测性（自定义扩展）
- [L8.1] Prometheus 指标端点 `/metrics`
- [L8.2] Grafana Dashboard 渲染正常
- [L8.3] 请求/Token/成本/渠道/缓存 五类指标采集

### L9 — Canary 质量监控（自定义扩展）
- [L9.1] 5 个标准测试用例定时跑
- [L9.2] 渠道健康度评分
- [L9.3] 异常告警

### L10 — 管理后台
- [L10.1] Dashboard 聚合 API（系统/渠道/路由/安全/MCP/Canary）
- [L10.2] 模型广场 API（列表+价格+能力）
- [L10.3] 在线充值（**当前未开启，需后续配置**）

---

## 四、验收标准（全部可机器执行）

> **每个验收项对应一条可重放命令**。客户验收时直接跑这些命令。

### 验收前置变量

```bash
# 客户/验收人员自行填入
export YUXIN_BASE_URL="http://103.55.131.130"
export YUXIN_USER_TOKEN="<填入普通用户 token>"
export YUXIN_ADMIN_TOKEN="<填入管理员 token>"
export YUXIN_TEST_MODEL="gpt-4o-mini"  # 或任一可用模型
```

### L1 公开页面

```bash
# L1.1 首页加载
curl -sS -o /dev/null -w "L1.1 HTTP=%{http_code} time=%{time_total}s\n" \
  $YUXIN_BASE_URL/
# 通过条件: HTTP=200, time<2s

# L1.2 状态页加载
curl -sS -o /dev/null -w "L1.2 HTTP=%{http_code}\n" \
  $YUXIN_BASE_URL/status
# 通过条件: HTTP=200

# L1.3 定价页加载
curl -sS -o /dev/null -w "L1.3 HTTP=%{http_code}\n" \
  $YUXIN_BASE_URL/pricing
# 通过条件: HTTP=200

# L1.4 首页无严重 JS 错误（用 Playwright）
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const errors = [];
  page.on('pageerror', e => errors.push(e.message));
  await page.goto('$YUXIN_BASE_URL/', { waitUntil: 'networkidle' });
  const severe = errors.filter(m => !m.includes('401'));
  console.log('L1.4 severe_errors=' + severe.length);
  if (severe.length > 0) { console.log(severe); process.exit(1); }
  await browser.close();
})();
"
# 通过条件: severe_errors=0
```

### L2 认证流程

```bash
# L2.1 登录页
curl -sS -o /dev/null -w "L2.1 HTTP=%{http_code}\n" $YUXIN_BASE_URL/login
# 通过条件: HTTP=200

# L2.2 注册页
curl -sS -o /dev/null -w "L2.2 HTTP=%{http_code}\n" $YUXIN_BASE_URL/register
# 通过条件: HTTP=200

# L2.3 错误登录被拒
curl -sS -X POST $YUXIN_BASE_URL/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"nouser12345","password":"wrongpass"}' \
  -w "\nL2.3 HTTP=%{http_code}\n"
# 通过条件: HTTP=4xx 且响应包含 "invalid"/"用户名或密码"
```

### L5 API 链路（核心）

```bash
# L5.1 系统状态 API
curl -sS $YUXIN_BASE_URL/api/status | jq '.success'
# 通过条件: true

# L5.3 公开定价 API
curl -sS $YUXIN_BASE_URL/api/pricing | jq '.data | length'
# 通过条件: ≥ 3 个模型

# L5.4 OpenAI 兼容 chat completions
curl -sS $YUXIN_BASE_URL/v1/chat/completions \
  -H "Authorization: Bearer $YUXIN_USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$YUXIN_TEST_MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"max_tokens\":10}"
# 通过条件: HTTP=200 且响应包含 choices[0].message.content

# L5.6 Token 鉴权（无 token）
curl -sS -o /dev/null -w "L5.6a HTTP=%{http_code}\n" \
  $YUXIN_BASE_URL/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$YUXIN_TEST_MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}]}"
# 通过条件: HTTP=401

# L5.6 Token 鉴权（错 token）
curl -sS -o /dev/null -w "L5.6b HTTP=%{http_code}\n" \
  $YUXIN_BASE_URL/v1/chat/completions \
  -H "Authorization: Bearer sk-invalid-token" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$YUXIN_TEST_MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}]}"
# 通过条件: HTTP=401

# L5.7 用量统计 API（已认证）
curl -sS -H "Authorization: Bearer $YUXIN_USER_TOKEN" \
  $YUXIN_BASE_URL/api/user/self/dashboard | jq '.data.quota'
# 通过条件: HTTP=200 且返回数值
```

### L8 可观测性

```bash
# L8.1 Prometheus 指标端点
curl -sS $YUXIN_BASE_URL/metrics | head -20
# 通过条件: 输出包含 yuxin_ 或 new_api_ 前缀的指标行

# L8.2 Grafana 可访问
curl -sS -o /dev/null -w "L8.2 HTTP=%{http_code}\n" \
  $YUXIN_BASE_URL:3000/api/health
# 通过条件: HTTP=200
```

---

## 五、不做清单（Out of Scope）

- ❌ 修改 new-api 上游核心计费逻辑
- ❌ 修改 new-api 上游数据库 schema（除非必要）
- ❌ 引入新的前端框架
- ❌ 直接操作生产数据库（必须通过迁移脚本）
- ❌ 部署期间不做停机（必须滚动/蓝绿）
- ❌ 在线充值（**v1.0 不做**，需后续开通第三方支付通道）

---

## 六、风险清单

| 风险 | 等级 | 缓解方案 |
|---|---|---|
| **AGPL-3.0 许可证传染** | 🔴 高 | 客户若对外提供 SaaS，必须开源修改后的源码。**必须告知客户**让其做知情决策（开源 / 购买商业授权 / 接受风险） |
| ClickHouse 容器 unhealthy | 🟡 中 | 已发现，需在动作 C 前排查（不影响主链路） |
| 上游 new-api 版本锁定 | 🟡 中 | 上游版本固定在 v1.0.0-rc.21，禁止未经测试升级 |
| 渠道 key 泄露风险 | 🟡 中 | 所有上游渠道 key 在生产 .env 中，禁止入 git |
| 单点服务器故障 | 🟡 中 | 单台 feifei，无高可用。需备份策略 |

---

## 七、许可证风险（**必须在动作 C 验收前摊牌**）

豫鑫 API 基于 new-api 二次开发，**上游 LICENSE 为 AGPL-3.0**。

AGPL-3.0 的关键约束：
1. **网络使用即分发**：任何通过网络提供服务的修改版本，必须向所有用户开放源码
2. **修改必须开源**：客户对 new-api 的所有修改，必须以 AGPL-3.0 开源
3. **传染性**：与修改代码链接的其他代码也可能被 AGPL 约束

**客户的三个选择**：
- **A. 接受 AGPL，开源豫鑫 API 修改部分**（成本最低，但代码公开）
- **B. 向 new-api 上游购买商业许可证**（成本最高，闭源合法）
- **C. 不公开源码、不买商业许可证**（成本为零，但存在法律风险）

**我方责任**：在交付物中**明确告知客户上述三个选项**，让客户做知情决策。**不可隐瞒**。

---

## 八、交付物清单（动作 C 收尾时全部产出）

| 类别 | 文件 | 状态 |
|---|---|---|
| 源码 | GitHub 私有仓库 `uxin-api-gateway` | ✅ |
| 部署手册 | `DEPLOYMENT.md` | ❌ 待补 |
| 运维手册 | `OPERATIONS.md` | ❌ 待补 |
| 备份恢复手册 | `BACKUP.md` | ❌ 待补 |
| 验收脚本 | `acceptance/run-acceptance.sh`（重放所有 L1-L10）| ❌ 待补 |
| evidence/ | 质量门禁证据 | ⏳ 动作 B 收集中 |
| delivery-manifest.md | 交付清单 | ❌ 待动作 B 末尾生成 |

---

## 九、验收流程（动作 C）

1. 客户填入验收前置变量（token / model）
2. 客户在生产服务器跑 `bash acceptance/run-acceptance.sh`
3. 全部 PASS → 客户签字
4. 我方跑 `/prd-retrospect` 复盘，记入 experience
5. AGPL 许可证摊牌，客户决策（A/B/C）

---

