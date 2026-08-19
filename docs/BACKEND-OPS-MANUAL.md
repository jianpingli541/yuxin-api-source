# 豫鑫 API 网关 · 后端运维配置手册（0 基础版）

> 适用对象：完全没接触过本系统的运维人员。
> 读完你能做到：连上服务器 → 看懂部署 → 改任何一项配置并验证生效 → 发布/回滚 → 备份恢复 → 处理常见故障。
> 版本：v1.2.22-yuxin-wallet-pay · 实测日期 2026-08-19 · 所有命令均在 feifei 服务器实测通过。
>
> **使用方法**：按章节顺序读。每个配置项都按统一格式写——【在哪】【是什么】【怎么改】【怎么验证】【出错怎么办】。复制命令直接执行即可。

---

## 第 0 章 · 先搞懂：这台服务器上跑着什么

### 0.1 一句话
豫鑫 API 网关是一套 **AI 模型统一接入平台**：用户注册账号 → 充值 → 拿 API Key → 通过统一接口调用各家大模型，平台按 token 计费扣费。

### 0.2 服务器在哪、怎么连
- 服务器主机名：`hk5708`（香港），Ubuntu，物理机
- 从运维工作机连接（已配好免密）：
  ```bash
  ssh feifei
  ```
- 项目代码目录（**所有后端配置都在这里改**）：
  ```bash
  cd /root/projects/api-gateway
  ```
- 验证你连对了：
  ```bash
  ssh feifei 'docker ps --format "{{.Names}}" | head'
  ```
  应该看到 `gateway-new-api`、`gateway-postgres` 等一批 `gateway-` 开头的容器。

### 0.3 系统由哪些部件组成（16 个容器）
```
用户浏览器
   │  https://ai.yuxin.yun (443端口)
   ▼
gateway-nginx ──────────── 网站入口（TLS证书/反代/限流/缓存）
   │
   ▼
gateway-new-api ────────── 核心业务（Go程序，:3000，前端+API+计费+调度全在这）
   ├─► gateway-postgres ── 存用户/密钥/渠道/订单/配置（业务数据库）
   ├─► gateway-redis ───── 缓存+限流计数
   ├─► gateway-clickhouse ─ 存每一次API调用日志（对账用）
   └─► gateway-wechat-server ─ 微信公众号扫码登录组件
观测栈（监控用，不影响业务）：
   gateway-prometheus（采指标）→ gateway-grafana（看板）→ gateway-alertmanager（告警）
   → gateway-alert-forwarder（把告警发到钉钉/企微/飞书/邮件）
   + 4个 exporter（postgres/redis/node/nginx 指标采集）
```

**记住三个最常用的命令**（90% 日常操作靠它们）：
```bash
ssh feifei 'cd /root/projects/api-gateway && docker compose ps'        # 看所有容器状态
ssh feifei 'docker logs gateway-new-api --tail 50'                     # 看业务日志
ssh feifei 'curl -sk https://ai.yuxin.yun/api/status | head -c 200'    # 看网站是否活着
```

---

## 第 1 章 · 配置文件地图：改东西之前先知道去哪改

后端配置分**三层**，改之前先判断你要改的东西在哪一层：

| 层 | 位置 | 改完怎么生效 | 典型内容 |
|---|---|---|---|
| **文件层** | `.env`、`docker-compose.yml`、`nginx/conf.d/*.conf` | 重启对应容器 / nginx reload | 数据库密码、端口、密钥、TLS |
| **数据库层** | `options` 表（57 个键） | 后台改立即生效；直接改 SQL 60 秒内自动生效 | 站点名、支付、定价、开关 |
| **代码层** | `model/option.go`、`setting/` | 要重新编译发布（第 6 章） | 配置项的注册和默认值 |

> ⚠️ **最重要的一个坑**（0 基础必看）：
> `.env` 文件里有一段 `Waffo*/Stripe*/SMTP*/TURNSTILE*` 变量，**这些是占位符，没有任何代码读它们**！
> 你要配 SMTP 邮件、Stripe 支付、人机验证，**不能写 .env**，要去**管理后台**（网页）或改 `options` 表。
> 已实测：`grep` 全代码库，没有任何 Go 代码从环境变量读 SMTP/Stripe。

### 1.1 文件层清单
```
/root/projects/api-gateway/
├── .env                            ← 数据库密码/redis密码/会话密钥/加密密钥/时区等
├── docker-compose.yml              ← 主栈 9 个容器的定义（端口/卷/健康检查）
├── docker-compose.observability.yml ← 监控栈 4 个容器
├── nginx/conf.d/gateway.conf       ← 网站入口（TLS/反代/限流/缓存）
├── nginx/conf.d/security_headers.conf ← 安全响应头
└── observability/                  ← 监控告警的详细配置
    ├── prometheus/prometheus.yml + rules/
    ├── alertmanager/alertmanager.yml
    ├── alert-forwarder/config/webhooks.list  ← 告警发到哪（钉钉/企微/飞书）
    └── grafana/provisioning/
```

---

## 第 2 章 · `.env` 环境变量（文件层核心）

【在哪】`/root/projects/api-gateway/.env`（权限 600，只有 root 能读）
【是什么】启动时注入各容器的环境变量。改完要重启对应容器才生效。

### 2.1 逐项说明（按用途分组）

**数据库（PostgreSQL）**
| 变量 | 作用 | 改后生效 | 验证 |
|---|---|---|---|
| `POSTGRES_USER` | 数据库用户名（gateway） | `docker compose up -d postgres` | 见下 |
| `POSTGRES_PASSWORD` | 数据库密码 | 同上（改密码见 2.3 警告） | `docker exec gateway-postgres psql -U gateway -d new-api -c 'SELECT 1'` |
| `POSTGRES_DB` | 数据库名（new-api） | 同上 | — |
| `SQL_DSN` | new-api 连数据库的完整连接串 | `docker compose up -d new-api` | 启动日志无 DB 报错 |

**缓存（Redis）**
| 变量 | 作用 | 改后生效 | 验证 |
|---|---|---|---|
| `REDIS_PASSWORD` | redis 密码 | `docker compose up -d redis new-api` | `docker exec gateway-redis redis-cli -a <密码> ping` 返回 PONG |
| `REDIS_CONN_STRING` | new-api 连 redis 的串 | 同上 | 业务无缓存类报错 |

**安全密钥（⚠️ 高风险，改前必读 2.3）**
| 变量 | 作用 | 改动后果 |
|---|---|---|
| `SESSION_SECRET` | 用户登录会话签名密钥 | **改=所有用户被强制登出**，需重新登录 |
| `CRYPTO_SECRET` | 支付私钥/渠道密钥的加密密钥 | **改=所有已加密的密钥全部失效**，见第 5 章 |

**日志库（ClickHouse）**
| 变量 | 作用 | 验证 |
|---|---|---|
| `LOG_SQL_DSN` | new-api 写调用日志的连接串 | 智能调度看板有数据 |
| `CH_PASS` | ClickHouse 密码 | `docker exec gateway-clickhouse clickhouse-client --password <密码> -q 'SELECT 1'` |

**其他常用**
| 变量 | 作用 | 默认 |
|---|---|---|
| `TZ` | 容器时区 | Asia/Shanghai |
| `PORT` | new-api 监听端口 | 3000 |
| `SYNC_FREQUENCY` | options 表改动自动重载周期（秒） | 60 |
| `STREAMING_TIMEOUT` | 流式响应超时（秒） | 300 |
| `TRUSTED_PROXIES` | 信任的反代网段（决定真实客户端 IP 识别） | 127.0.0.1,172.16.0.0/12 |
| `SESSION_COOKIE_SECURE` | 安全 Cookie 开关 | true（配 true 必须同时配 `SESSION_COOKIE_TRUSTED_URL`，否则启动失败） |
| `SESSION_COOKIE_TRUSTED_URL` | 可信 HTTPS 来源白名单 | https://ai.yuxin.yun,https://203.0.113.10 |

### 2.2 怎么改 `.env`（标准流程）
```bash
# 1. 先备份（铁律：改配置前必备份）
ssh feifei 'cd /root/projects/api-gateway && cp .env .env.bak.$(date +%Y%m%d_%H%M%S)'

# 2. 编辑（用 sed 替换，或 vi 手动改）
ssh feifei 'cd /root/projects/api-gateway && sed -i "s/^TZ=.*/TZ=Asia\/Shanghai/" .env'

# 3. 重启受影响的容器
ssh feifei 'cd /root/projects/api-gateway && docker compose up -d new-api'

# 4. 验证
ssh feifei 'docker exec gateway-new-api date'
```

### 2.3 高危警告（改这两个之前必看）
- **改 `POSTGRES_PASSWORD`**：光改 .env 没用，数据库 volume 里的密码还是旧的，会连不上。正确做法：先 `ALTER USER` 改数据库里的密码，再同步改 .env 和 SQL_DSN，再重启。不确定就别动。
- **改 `CRYPTO_SECRET`**：会让所有加密的支付私钥/渠道密钥解不开（报 `failed to decrypt`），等于支付和上游渠道全瘫痪。**没有重加密全部密钥的把握，绝对不要改。**

---

## 第 3 章 · `docker-compose.yml`（容器编排）

【在哪】`/root/projects/api-gateway/docker-compose.yml`
【是什么】定义 9 个业务容器怎么跑。改完用 `docker compose up -d <服务名>` 生效（自动重建变化的容器）。

### 3.1 9 个容器各自干什么
| 容器 | 干什么 | 端口 | 改它的影响 |
|---|---|---|---|
| nginx | 网站入口 TLS/反代 | 80/443 对公网 | 改错=网站打不开 |
| new-api | 核心业务 | 3000（不对公网） | 改错=全站功能异常 |
| postgres | 业务数据库 | 5432（仅内网） | 改错=数据丢失风险 |
| redis | 缓存限流 | 6379（仅内网） | 改错=限流失效/性能降 |
| clickhouse | 调用日志库 | 9000/8123（仅内网） | 改错=对账/日志丢失 |
| wechat-server | 微信登录 | 127.0.0.1:3002 管理台 | 改错=微信登录失效 |
| 4个 exporter | 监控指标采集 | 仅内网 | 改错=监控断 |

### 3.2 常用操作
```bash
cd /root/projects/api-gateway
docker compose ps                    # 看状态（healthy=健康）
docker compose up -d <服务名>         # 改完配置重启某容器
docker compose logs -f new-api       # 跟踪业务日志
docker compose restart new-api       # 重启业务（约几秒中断）
```

### 3.3 验证
```bash
docker compose ps        # 全部 healthy 即正常
docker ps --format "{{.Names}} {{.Status}}"
```

### 3.4 已知坑
- 存在 7 个 `docker-compose.yml.bak.*`（release.sh 每次部署自动备份），不是垃圾，别删。
- 若发现 `docker ps` 里有不在 compose 里的容器（如 `clever_bartik`），用 `docker inspect <名字>` 查它是什么，通常是历史遗留，确认无用后 `docker rm -f <名字>`。

---

## 第 4 章 · nginx 网站入口配置

【在哪】`/root/projects/api-gateway/nginx/conf.d/gateway.conf`
【是什么】用户访问 https://ai.yuxin.yun 的第一道门。改完**热加载**（不中断服务）：
```bash
ssh feifei 'docker exec gateway-nginx nginx -t && docker exec gateway-nginx nginx -s reload'
```
`nginx -t` 显示 ok 才能 reload，否则配置有错。

### 4.1 关键配置项
| 配置 | 当前值 | 作用 |
|---|---|---|
| 80 端口 | 跳转 443 + Let's Encrypt 证书续期验证 | HTTP 强制转 HTTPS |
| 443 端口 TLS | Let's Encrypt 证书 | HTTPS 加密（证书自动续期，见 crontab 03:17） |
| Cloudflare realip | 22 个 CF 网段 | 识别真实客户端 IP（限流按真实 IP） |
| 限流 | api 600r/m；注册 5r/m（burst=3） | 防刷。注册限流最严 |
| 静态资源缓存 | js/css/图 7 天 | 加速页面加载 |
| SSE（流式） | buffering off + 300s 超时 | AI 流式输出不卡顿的关键 |

### 4.2 安全响应头（security_headers.conf）
HSTS、X-Frame-Options、CSP 等 10 条，已配好，一般不用动。

### 4.3 ⚠️ 已知问题：版本脱敏失效
`gateway.conf` 第 229 行有一句：
```
sub_filter '"version":"1.2.12-yuxin"' '"version":"unknown"';
```
本意是把 `/api/status` 里的版本号脱敏成 "unknown"，但**版本号写死了 1.2.12，现在已是 1.2.22，所以对不上、脱敏失效**（实测 `/api/status` 对外暴露真实版本号）。
**处理**：每次升级版本后，要么把这行的 `1.2.12-yuxin` 改成当前版本，要么直接删除这行。这属于"升级检查项"（见第 6 章）。

---

## 第 5 章 · 支付配置（最容易踩坑，重点看）

支付配置**不在 .env**，在**数据库 options 表**，通过**管理后台**或 **payment-setup 工具**配置。

### 5.1 微信支付需要哪些凭据、从哪拿
| 配置项 | 从哪获取 | 存哪 |
|---|---|---|
| WechatMerchantId | 微信商户平台「账户中心」，10 位数字 | options 表明文 |
| WechatAppId | 公众号/小程序后台，与商户号绑定 | options 表明文 |
| WechatApiV3Key | 商户平台「API 安全」→ APIv3 密钥，32 位 | options 表**加密** |
| WechatPrivateKey | 商户平台下载 apiclient_key.pem | options 表**加密** |
| WechatCertSerialNo | 商户平台「API 安全」→ 证书管理 | options 表明文 |
| WechatCertPublicKey | 商户证书 apiclient_cert.pem | options 表明文 |
| WechatNotifyUrl | 留空即可，默认 https://ai.yuxin.yun/api/wechat/notify | 商户后台须填同地址 |

### 5.2 支付宝需要哪些凭据
| 配置项 | 从哪获取 |
|---|---|
| AlipayAppId | 支付宝开放平台应用详情 |
| AlipayPrivateKey | 开放平台「应用→接口加签方式」，PKCS8 PEM |
| AlipayPublicKey | 同上「查看支付宝公钥」（验签用） |
| AlipayUseCertMode | false=公钥模式（当前用）；true=证书模式（需另配 3 个证书） |

### 5.3 三种写入方式（等价，选一）
- **方式 A（推荐，最简单）**：登录管理后台 → 系统设置 → 计费与支付 → 支付网关，逐项填入保存。
- **方式 B（运维批量）**：用 payment-setup 工具。⚠️ 注意：该工具**未预编译进容器**（容器里只有 /new-api 主程序，已实测），需先在服务器编译再挂入容器执行：
  ```bash
  cd /root/projects/api-gateway
  # 编译（服务器有 Go 环境）
  CGO_ENABLED=0 /usr/local/go/bin/go build -o /tmp/payment-setup ./cmd/payment-setup
  # 挂入容器执行
  docker cp /tmp/payment-setup gateway-new-api:/payment-setup
  docker cp payment.json gateway-new-api:/tmp/payment.json   # 先把凭据 json 传进去
  docker exec gateway-new-api /payment-setup --write /tmp/payment.json
  docker exec gateway-new-api /payment-setup --verify        # 验证客户端构造成功
  docker exec gateway-new-api rm -f /payment-setup /tmp/payment.json  # 用完即删
  rm -f payment.json /tmp/payment-setup
  ```
  嫌麻烦就直接用方式 A，效果完全一样。
- **方式 C（直接改库）**：改 options 表，60 秒内自动生效。

### 5.4 怎么验证支付配好了
```bash
# 看日志有没有解密报错
ssh feifei 'docker logs gateway-new-api 2>&1 | grep -i decrypt | tail'
# 用 payment-setup 验证
ssh feifei 'docker exec gateway-new-api /payment-setup --verify'
# 最终验证：前端用测试账号走一遍充值，看能否出二维码
```

### 5.5 加密机制（enc1 是什么）
- 支付私钥、APIv3 密钥在数据库里是**加密存储**的，格式 `enc1:<...>:<...>`
- 加密密钥来自 `.env` 的 `CRYPTO_SECRET`（AES-256-GCM）
- **改 CRYPTO_SECRET = 所有加密密钥失效**。轮换它必须先重新加密全部密钥。

---

## 第 6 章 · 发布新版本与回滚

### 6.1 发布（一条命令）
```bash
ssh feifei 'cd /root/projects/api-gateway && bash scripts/release.sh v1.2.23-yuxin'
```
这条命令自动做 6 件事：
1. 构建前端（bun）
2. 构建 Go 二进制（把版本号写进去）
3. 构建 Docker 镜像
4. **推送 AGPL 源码到公开披露仓库**（法律合规要求，推送失败会中止发布，不会出现"新代码上线但源码没公开"）
5. 更新 compose 镜像标签并重启
6. 健康检查 + **版本一致性校验**（防止"部署的版本"和"上报的版本"对不上）

### 6.2 发布后必须检查的 3 件事
```bash
# 1. 版本对不对
ssh feifei 'curl -sk https://ai.yuxin.yun/api/status | grep -o "1.2.23-yuxin"'
# 2. nginx 版本脱敏行要不要同步（见 4.3）
# 3. 网站能不能打开
curl -sk -o /dev/null -w "%{http_code}\n" https://ai.yuxin.yun/
```

### 6.3 回滚
```bash
# 看有哪些历史镜像
ssh feifei 'docker images "yuxin-api"'
# 把 compose 里的镜像标签改回上一个版本，再 up
ssh feifei 'cd /root/projects/api-gateway && sed -i "s|yuxin-api:v1.2.23.*|yuxin-api:v1.2.22-yuxin-wallet-pay|" docker-compose.yml && docker compose up -d new-api'
# 验证
ssh feifei 'curl -sk https://ai.yuxin.yun/api/status | grep -o "1.2.22"'
```

---

## 第 7 章 · 备份与恢复

### 7.1 自动备份（已配好，每天跑）
crontab 每天 02:00 自动备份到 `/root/projects/api-gateway/backups/auto/`：
- PostgreSQL 数据库（pg_dump）
- Redis（dump.rdb）
- ClickHouse 日志库
- 配置文件（.env + compose + nginx）
**保留 7 天**（注意：BACKUP.md 写 30 天是文档过时，实际代码是 7 天）。

### 7.2 手动备份
```bash
ssh feifei 'cd /root/projects/api-gateway && ./manage.sh backup'
```

### 7.3 恢复演练（每月自动验证备份可用）
crontab 每月 1 日 05:30 自动做恢复演练（把备份还原进临时容器验证数据完整）。看结果：
```bash
ssh feifei 'ls /root/projects/api-gateway/backups/auto/drill_*'
# 看到 PG_DRILL_PASS / CH_DRILL_PASS 即备份可用
```

### 7.4 真要恢复怎么办
```bash
# 1. 找到最近的备份
ssh feifei 'ls -la /root/projects/api-gateway/backups/auto/pg_*.sql.gz | tail -3'
# 2. 按 BACKUP.md 的灾难恢复流程操作（先恢复到临时容器验证，再切生产）
```

---

## 第 8 章 · 监控与告警

### 8.1 看板在哪
- Grafana（可视化看板）：http://127.0.0.1:3001（服务器本地端口，需 SSH 隧道或直接服务器上访问）
  ```bash
  # 从工作机开隧道
  ssh -L 3001:127.0.0.1:3001 feifei
  # 然后浏览器开 http://localhost:3001，用 grafana admin 密码登录
  ```
- 已有 3 块看板：API 网关监控、智能调度(auto)分布、运营看板

### 8.2 已有哪些告警（12 条自动监控）
服务类：new-api 宕机、postgres/redis/nginx 宕机、主机宕机
性能类：API P99 延迟>5s、API 错误率>10%、内存>90%、磁盘>85%、磁盘将满
业务类：**单小时成本>$5、单日成本>$50**（防盗刷/失控）

### 8.3 告警发到哪（需要配置）
告警目前只落本机。**要发到钉钉/企微/飞书，编辑**：
```bash
ssh feifei 'vi /root/projects/api-gateway/observability/alert-forwarder/config/webhooks.list'
# 每行填一个 webhook 地址，后缀标类型：
# https://oapi.dingtalk.com/robot/send?access_token=XXX#dingtalk
# https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=XXX#wecom
# 保存即生效（不用重启）
```

### 8.4 改告警规则
```bash
# 编辑规则
ssh feifei 'vi /root/projects/api-gateway/observability/prometheus/rules/yuxin.yml'
# 校验语法
ssh feifei 'docker exec gateway-prometheus promtool check rules /etc/prometheus/rules/yuxin.yml'
# 重启生效（注意：prometheus 不支持热加载，必须重启）
ssh feifei 'docker restart gateway-prometheus'
```

---

## 第 9 章 · 常见故障速查

| 症状 | 先看这个 | 大概率原因 |
|---|---|---|
| 网站打不开 | `docker compose ps` | 容器挂了 → `docker compose up -d` |
| 网站 502 | `docker logs gateway-nginx` + `docker ps` | new-api 挂了或在重启 |
| 支付报"配置错误" | `docker logs gateway-new-api \| grep -i decrypt` | 支付密钥加密失效（CRYPTO_SECRET 被改） |
| 用户调 API 报 503 | 管理后台看渠道状态 | 渠道被自动禁用（上游故障/测试超时） |
| 登录不上 | 后台看用户状态 | 密码错/账号被禁/限流（5r/m 注册限流） |
| 页面改了不生效 | 强刷（Ctrl+F5） | nginx 静态缓存 7 天，或 CDN 缓存 |
| 配置改了不生效 | 等 60 秒 | options 表 SQL 改动要 SYNC_FREQUENCY=60s 才重载 |

### 一键自检（客户也能跑）
```bash
ssh feifei 'cd /root/projects/api-gateway && bash scripts/self-check.sh'
```

---

## 附录 A · 后端配置入口总览图
```
文件层（改完重启/reload）
  .env                  → 数据库/redis/会话/加密密钥、时区、端口
  docker-compose.yml    → 9 容器拓扑
  nginx/conf.d/*.conf   → TLS/反代/限流/缓存/安全头（reload 热加载）
数据库层 options 表 57 键（后台改立即生效 / SQL 改 60s）
  站点品牌(7) 认证(3) 计费倍率(10) 支付(15) 智能调度(3) 安全运维(19)
代码层 model/option.go + setting/（改要重新发布）
```

## 附录 B · 当前已知问题（2026-08-19）
1. nginx 版本脱敏失效（sub_filter 写死 1.2.12，实际 1.2.22）→ 升级后要同步
2. .env 支付/SMTP/TURNSTILE 段是占位符，配置了不生效（要走后台/options 表）
3. BACKUP.md 保留期写 30 天，实际代码 7 天
4. 微信支付 APIv3 密钥待商户平台重置（见微信密钥阻塞记录）
