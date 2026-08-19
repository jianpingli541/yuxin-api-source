# 上线前认证安全动作清单（Pre-Launch Auth Security）

> 建立：2026-08-18（v1.2.20-yuxin-register-validation 发布日）
> 状态：**未执行**，上线前按本文档逐项落实并勾选
> 范围：登录 / 注册 / 会话 / 密码策略（yuxin API 网关，feifei hk5708）

## 0. 已完成基线（2026-08-18 前）

代码层已具备，无需重复建设：

- 密码 bcrypt 存储；登录统一报错（无用户名枚举）
- 登录锁定：用户名+IP 连续 5 次失败锁 15 分钟（Redis，`controller/login_lockout.go`）
- 双层限流：nginx 注册 5r/m（`reg_limit`）+ 应用 `CriticalRateLimit`
- 安全响应头全套：HSTS / CSP / X-Frame-Options / nosniff 等（`nginx/conf.d/security_headers.conf`）
- refresh cookie：HttpOnly + Secure + SameSite=Strict + Path 收窄到 `/api/user/auth` + Origin Guard
- 2FA(TOTP) 与 Turnstile 机制代码已接入（`controller/user.go`、`middleware/turnstile-check.go`），开关未启用
- 注册校验（v1.2.20）：用户名 3-20 位字母/数字/下划线/连字符；密码 8-20 位含大小写+数字+特殊符号（注册+改密同时生效）；邮箱选填但须合法格式；前后端规则已对齐

## 1. 当前风险快照（2026-08-18 实测生产配置）

| 项 | 现状 | 风险 |
|---|---|---|
| `turnstile_check` | **false**（key 为空） | 注册/登录无人机校验，换 IP 即可批量注册 |
| `email_verification` | **false** | 注册不验邮箱归属；无密码找回渠道 |
| SMTP | **未配置**（SMTPServer 为空） | 邮箱验证/找回密码均不可用 |
| root 管理员 | **2FA 未开 + 历史弱密码**（8-17 审计实锤） | 单点失陷即全盘失陷 |
| 存量用户 | 13 个，均建于"≥6 位"测试策略期 | 可能存在弱密码账号 |
| 登录锁定维度 | 用户名+IP | 撞库换 IP 可继续尝试（防锁定 DoS 的刻意取舍） |

## 2. 上线前必做（P0）

### P0-1 root 管理员改密 + 开启 2FA
- 动作：root 登录 → 个人设置 → 修改密码（新策略强制 8-20 位四类字符）→ 开启 TOTP 2FA
- 责任：**客户本人操作**（2FA 密钥须客户自己持有）
- 验收：用旧密码登录失败；新密码登录触发 2FA 流程（`require_2fa: true`）

### P0-2 关闭裸注册（三选一，推荐 a）
- a) 配置 SMTP + 开启邮箱验证（推荐）：系统设置填入 SMTP 服务器/账号/授权码 → 打开 `email_verification`；顺带解锁找回密码
  - 外部依赖：**需要一个可用 SMTP 账号**（QQ/163/企业邮均可）
  - 验收：`/api/status` 返回 `email_verification: true`；注册接口缺 `verification_code` 返回邮箱验证必填错误
- b) 开 Turnstile：填 `TURNSTILE_SITE_KEY` / `TURNSTILE_SECRET_KEY` → 注册+登录自动生效（代码已接线）
  - 验收：`/api/status` 返回 `turnstile_check: true`；无 token 注册返回人机校验错误
- c) 临时方案：关闭密码注册走邀请制（`RegisterEnabled=false` 或 `PasswordRegisterEnabled=false`）
- 注意：开邮箱验证后，注册提交的 email 才会落库（`Register` 中 `EmailVerificationEnabled` 分支）

### P0-3 存量弱密码账号强制改密
- 动作：上线前对 13 个存量用户执行强制重置（后台标记/逐个人工通知改密），或清库重开（视客户决策）
- 验收：抽查任一老账号，确认密码已按新策略重置

## 3. 上线前建议（P1）

### P1-1 账户级撞库补偿
- 现状：锁定按 用户名+IP，换 IP 即续试
- 动作：增加账户级计数——同一账户 24h 跨 IP 累计失败 ≥N 次 → 该账户登录强制 Turnstile（保持 IP 隔离设计，避免锁定 DoS）
- 涉及：`controller/login_lockout.go` + `controller/user.go` Login

### P1-2 access token TTL 复核
- 2026-08-18 审计未覆盖 access token 时长与吊销策略，上线前确认 TTL 处于业务可接受窗口、登出/改密后旧 token 失效（`AuthVersion` 机制已存在，需验证生效）

## 4. 卫生项（P2）

- `GENERATE_DEFAULT_TOKEN` 保持关闭（代码路径为注册送 50 万额度+unlimited token，2026-08-18 确认默认 false，严禁开启）
- 登录锁定/撞库事件接 alert-forwarder 告警（当前仅写 SysLog）
- 备份归档 `/root/backups/sec-hygiene-20260817/alipaytest.tgz`：支付稳定上线后删除（8-17 待办）

## 5. 执行顺序与依赖

```
P0-1（root 改密+2FA，客户本人，~10 分钟）
  → P0-2a（SMTP+邮箱验证，依赖：SMTP 账号）
  → P0-3（存量账号重置）
  → P1-1 / P1-2（代码改动，需排期）
```

外部依赖清单：① SMTP 账号（P0-2a）；② Turnstile key（P0-2b，如选）；③ 客户本人开 2FA（P0-1）。

## 6. 勾选记录

| 项 | 状态 | 执行人 | 日期 | 证据 |
|---|---|---|---|---|
| P0-1 root 改密+2FA | ☐ 未执行 | | | |
| P0-2 裸注册关闭 | ☐ 未执行 | | | |
| P0-3 存量账号重置 | ☐ 未执行 | | | |
| P1-1 账户级撞库补偿 | ☐ 未执行 | | | |
| P1-2 access token TTL 复核 | ☐ 未执行 | | | |
| P2 卫生项 | ☐ 未执行 | | | |
