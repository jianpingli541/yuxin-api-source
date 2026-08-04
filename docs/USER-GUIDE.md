# 豫鑫 API 中转站 · 用户操作手册

> 版本: v1.0.0-yuxin · 2026-08-03
> 访问地址(本环境): https://103.55.131.130
> 文档对象: 终端用户(调用方)、渠道管理员、系统管理员

---

## 目录

- [角色定义](#角色定义)
- [一、终端用户(API 调用方)](#一终端用户api-调用方)
- [二、渠道管理员](#二渠道管理员)
- [三、系统管理员](#三系统管理员)
- [四、安全与合规](#四安全与合规)
- [五、FAQ](#五faq)

---

## 角色定义

| 角色 | 入口 | 权限范围 |
|---|---|---|
| 终端用户(普通) | https://103.55.131.130/login | 查看公开定价、注册账户、生成 API Key、调用模型、查询用量 |
| 渠道管理员 | https://103.55.131.130/admin | 渠道列表、渠道测试、状态查询、用户管理 |
| 系统管理员 | 同上 + 管理员设置页 | 系统设置、模型管理、订阅套餐、审计日志 |

> 超级管理员 `role=100`,普通管理员 `role=10`(系统设置页可见性受限),普通用户 `role=1`。

---

## 一、终端用户(API 调用方)

### 1.1 注册与登录

1. 打开 `https://103.55.131.130/register`
2. 填写用户名(>=3 字符)、邮箱(可留空)、密码(>=8 字符,需含字母+数字)
3. 提交后系统跳转首页 → 已自动登录
4. **登录**:`https://103.55.131.130/login` → 输入用户名/邮箱 + 密码 → 进入控制台

> **可选**:开启两步验证(`设置 → 安全 → 两步验证`),扫码绑定 Authenticator;开启后每次登录除密码外还需输入 6 位动态码。

### 1.2 生成 API Key

1. 登录后进入 `控制台 → API Keys`
2. 点击 "新建 Key",填写:
   - 名称(如 `chatbot-prod`)
   - 配额(留空 = 无限)
   - 过期时间(留空 = 永不过期)
   - 允许调用的模型列表(留空 = 全部)
   - 允许的来源 IP(留空 = 不限制)
3. 点击 "提交",**复制显示的 Key(以 `sk-yuxin-` 开头,只显示一次)**
4. 妥善保存,丢失需重新生成

### 1.3 调用模型(OpenAI 兼容)

任选一种方式:

**A. cURL**

```bash
curl https://103.55.131.130/v1/chat/completions \
  -H "Authorization: Bearer sk-yuxin-XXXX" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"你好"}],"stream":false}'
```

**B. OpenAI 官方 SDK**

```python
from openai import OpenAI
client = OpenAI(
    api_key="sk-yuxin-XXXX",
    base_url="https://103.55.131.130/v1",
)
resp = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role":"user","content":"你好"}],
)
print(resp.choices[0].message.content)
```

**C. 流式(SSE)**:将 `stream: true` 加入请求,客户端按行解析 `data: {...}` 事件;以 `data: [DONE]` 结束。

### 1.4 查询用量

- Web:`控制台 → 用量`
- API:

```bash
curl https://103.55.131.130/api/user/tokenlog \
  -H "Authorization: Bearer sk-yuxin-XXXX" \
  -H "Content-Type: application/json" \
  -d '{"start_timestamp":1754000000,"end_timestamp":1754100000}'
```

### 1.5 充值与订阅

**当前未开启在线充值**(见 ACCEPTANCE 报告),如需充值请联系管理员通过后台手动加配额,或订阅下列套餐之一:

| 套餐 | 价格 | 配额 |
|---|---|---|
| 试用 7 天 | ¥0 | 100K tokens |
| 月度标准 | ¥99 | 5M tokens |
| 月度专业 | ¥499 | 30M tokens |
| 企业定制 | 联系商务 | 议定 |

### 1.6 安全设置(可选)

- **修改密码**:`设置 → 安全 → 修改密码`
- **两步验证**:`设置 → 安全 → 启用 2FA`
- **Passkey**:`设置 → 安全 → 添加 Passkey`(指纹/Windows Hello)
- **查看登录设备**:`设置 → 安全 → 登录会话` → 可远程踢出

---

## 二、渠道管理员

### 2.1 添加上游渠道

1. 进入 `管理后台 → 渠道`
2. 点击 "新建渠道",填写:
   - **类型**:OpenAI / Anthropic / Gemini / Azure / 自定义 等
   - **名称**:内部可识别名(如 `OpenAI-官方`,`DeepSeek-V4`)
   - **API Key**:上游厂商提供的密钥(**当前明文存储,R1 整改中**)
   - **Base URL**:上游接口地址
   - **代理分组**:用户组(`default`/`vip` 等),决定哪些用户可用
   - **权重**:整数,加权轮询用
   - **优先级**:数字越小越优先
   - **模型列表**:支持哪些模型,逗号分隔
3. 提交后,点击"测试"按钮验证连通性

### 2.2 渠道健康监控

- `渠道 → 健康` 页查看所有渠道实时状态(绿/黄/红)
- `管理 → 状态 → 测试` 主动 ping 所有渠道
- `Observability` → Prometheus/Grafana 看板(默认 admin/admin,**首次登录必须改密码**)

### 2.3 路由策略

支持四种智能路由(`管理 → 路由 → 策略`):

| 策略 | 说明 |
|---|---|
| 优先级权重 | 默认。权重高的渠道被选中概率大 |
| 成本优化 | 选最便宜的可用渠道 |
| 延迟优化 | 选响应最快的渠道(过去 60s 平均) |
| 质量优先 | 选历史成功率最高的渠道 |

用户调用时可动态覆盖:`X-Routing-Strategy: cost`/`latency`/`quality`/`priority` 请求头。

### 2.4 渠道熔断

- 渠道错误率超过阈值自动熔断 5 分钟(可在 `设置 → 渠道` 调整)
- 熔断期间用户请求自动路由到下一个可用渠道
- 手动解除:`渠道 → 操作 → 恢复`

---

## 三、系统管理员

### 3.1 系统设置

`管理后台 → 系统设置`

- **基础设置**:站点名称、Logo、备案号、用户协议、隐私政策
- **注册设置**:是否允许注册、是否需要邮箱验证、邀请码模式
- **登录设置**:登录尝试限制、是否开启 2FA 强制
- **计费设置**:充值汇率、订阅税率、自动签到积分
- **可观测性**:Prometheus / Grafana 地址、是否对外暴露 `/metrics`

### 3.2 用户管理

`管理 → 用户`

- 创建用户、调整角色、调整配额
- 启用/禁用/封禁账户(`status: 1=enabled, 2=disabled, 3=banned`)
- 查看用户用量明细

### 3.3 模型管理

`管理 → 模型`

- 调整模型定价(输入/输出 $/M tokens)
- 配置模型别名映射(对外名 ↔ 渠道实际名)
- 启用/禁用模型

### 3.4 订阅套餐

`管理 → 订阅`

- 增/改/删订阅套餐
- 查看订阅订单流水
- 手动退款(开发中)

### 3.5 审计日志

`管理 → 审计 → 操作日志`

- 记录所有管理员操作(增/删/改用户、渠道、配额等)
- 可按时间、用户、操作类型筛选
- 保留 90 天,过期归档至 ClickHouse

### 3.6 集群与任务

`管理 → 系统 → 任务`

- 查看后台任务运行状态(system_tasks 表)
- 节点 leader 选举(system_instances)
- 手动触发一次性任务

### 3.7 MCP 工具

`管理 → MCP`

- 启用/禁用内置工具(web_search / get_current_time / calculator)
- 编写自定义工具(高级用法,见 docs/mcp.md)

---

## 四、安全与合规

### 4.1 客户端安全

- HTTPS 强制(80 端口已 301 → 443)
- Cookie HttpOnly + SameSite=Strict(`SESSION_COOKIE_SECURE=true` 在生产 .env 已开)
- HSTS 头(`max-age=31536000`)
- CSP 默认 `self` 来源
- CORS 白名单(`middleware/cors.go` 内置)
- 全局限流:360 次 / 180 秒 / IP(可按端点分级,见 R9 整改)

### 4.2 数据安全

- 用户密码 bcrypt 哈希(cost=12)
- 上游渠道 Key **当前明文存 DB**(R1 整改中,加密落地后此节会更新)
- 备份:每日 02:00 cron 自动备份 PostgreSQL → `backups/auto/db_<时间>.sql`
- 备份保留:默认 7 天,需归档请联系运维

### 4.3 合规

- 隐私政策:首次访问时弹出,可在管理后台编辑
- 用户协议:同上
- Cookie 同意:登录态属于必要 Cookie,不做弹窗
- AGPL-3.0 商业合规:详见 `PRD.md §7`,客户需在 A/B/C 三选项中签字

### 4.4 日志与审计

- API 调用日志:ClickHouse 长期存储(`logs` 表同步)
- 管理员操作日志:`admin_audit_logs` 表
- Prometheus metrics:`/metrics`(内网可达,Grafana 可视化)

---

## 五、FAQ

### 5.1 调用时返回 401

- 检查 `Authorization: Bearer sk-yuxin-XXXX` 头格式
- 检查 Key 是否过期或被封禁(控制台 → API Keys 查看状态)
- 检查是否被全局限流(429 错误,见 4.1)

### 5.2 调用时返回 429

- 已超出全局限流(360 req/180s/IP)
- 等待限流窗口过期或联系运维放宽
- 集成方建议申请专属限流策略

### 5.3 调用时返回 502 / 503 / 504

- 上游渠道全部不可用或网络抖动
- 检查 `管理 → 状态 → 测试` 查看渠道状态
- 切换路由策略为 `cost` / `priority` 跳过故障渠道

### 5.4 流式响应中断

- 检查 nginx `proxy_read_timeout` 默认 300s
- 客户端断开后服务端也会停止发送
- 长上下文请使用 `stream: false` 或调高超时

### 5.5 看不到某些模型

- 模型未被加入你的 API Key 的"允许模型列表"
- 模型未被加入你的用户组
- 模型本身被管理后台禁用

### 5.6 修改密码 / 找回密码

- 已知密码改密码:`设置 → 安全 → 修改密码`
- 忘记密码:登录页 → "忘记密码" → 输入邮箱(若系统配置了 SMTP)→ 邮件链接重置
- 邮箱未配置:请联系管理员手动重置

### 5.7 性能/延迟调优

- 用户侧:使用延迟优化路由策略(`X-Routing-Strategy: latency`)
- 渠道侧:在管理后台配置渠道权重,把响应快的渠道权重调高
- 全局:Grafana 看板查看 P50/P95/P99 长尾渠道

### 5.8 商业合作/采购

- 商务对接:`商务@yuxin.com`(由客户方运营团队分发)
- 技术对接:`admin@yuxin.com`
- 安全事件:`security@yuxin.com`

---

## 附录 A · 角色权限矩阵(摘录)

| 操作 | 普通用户 | 普通管理员(role=10) | 超级管理员(role=100) |
|---|:-:|:-:|:-:|
| 调用模型 API | ✅ | ✅ | ✅ |
| 生成/管理自己的 API Key | ✅ | ✅ | ✅ |
| 查看自己的用量 | ✅ | ✅ | ✅ |
| 管理渠道 | ❌ | ✅ | ✅ |
| 管理用户(查询) | ❌ | ✅ | ✅ |
| 创建/封禁用户 | ❌ | ❌ | ✅ |
| 修改系统设置 | ❌ | 部分 | ✅ |
| 修改订阅套餐 | ❌ | ❌ | ✅ |
| 查看审计日志 | ❌ | 自己的 | ✅ |

## 附录 B · 常用 API 速查

| 用途 | 端点 |
|---|---|
| 模型列表 | `GET /v1/models` |
| 聊天补全 | `POST /v1/chat/completions` |
| 文本嵌入 | `POST /v1/embeddings` |
| 图像生成 | `POST /v1/images/generations` |
| 音频转写 | `POST /v1/audio/transcriptions` |
| 重排序 | `POST /v1/rerank` |
| 内容审查 | `POST /v1/moderations` |
| 系统状态 | `GET /api/status` |
| 公开定价 | `GET /api/public/pricing` |
| 公开模型 | `GET /api/marketplace/models` |

详细参数与响应见 `docs/openapi/api.json` 和 `docs/openapi/relay.json`。