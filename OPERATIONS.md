# 运维手册 — 豫鑫 API 中转站

> **版本**: v1.2.1-yuxin
> **最后更新**: 2026-07-26

---

## 一、日常运维命令

所有运维操作通过 `manage.sh`：

```bash
./manage.sh start      # 启动
./manage.sh stop       # 停止
./manage.sh restart    # 重启
./manage.sh status     # 查看状态
./manage.sh logs       # 看日志（默认 new-api）
./manage.sh logs nginx # 看指定服务日志
./manage.sh backup     # 备份数据库
./manage.sh update     # 更新镜像
```

---

## 二、监控

### 2.1 Prometheus 指标

- 端点：`http://<host>/metrics`
- 采集的指标（前缀 `new_api_*`）：
  - 请求总数（按渠道/模型/状态码）
  - Token 用量（prompt/completion/total）
  - 成本（按渠道/模型）
  - 缓存命中/未命中
  - 渠道健康度

### 2.2 Grafana Dashboard

- 访问：`http://<host>:3000`（默认 admin/admin，**首次登录必须改密码**）
- 仪表盘：自动 provision（`observability/grafana/dashboards/`）

### 2.3 容器健康检查

```bash
./manage.sh status
# 或
docker compose ps
# 关注 health 列：healthy / unhealthy / starting
```

### 2.4 已知问题排查

#### gateway-clickhouse unhealthy

症状：`gateway-clickhouse` 容器持续 unhealthy。

排查：
```bash
docker compose logs clickhouse | tail -50
docker compose exec clickhouse clickhouse-client --query "SELECT 1"
```

常见原因：
- 磁盘空间不足（≥ 90% 占用）
- 数据损坏 → 重启容器或恢复备份
- 配置错误 → 检查 `docker-compose.yml` clickhouse 段

---

## 三、日志管理

### 3.1 查看日志

```bash
# 全部
docker compose logs -f

# 指定服务最近 200 行
docker compose logs --tail 200 new-api
```

### 3.2 日志轮转

Docker 默认 json-file 驱动，已配置：
```yaml
# docker-compose.yml
logging:
  driver: json-file
  options:
    max-size: "100m"
    max-file: "5"
```

应用层日志（容器内 `/app/logs/`）按天切割。

---

## 四、性能调优

### 4.1 数据库

```bash
# 查看慢查询
docker compose exec postgres psql -U gateway new-api -c \
  "SELECT * FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
```

### 4.2 Redis

```bash
docker compose exec redis redis-cli INFO memory
```

### 4.3 渠道并发

- 默认每个渠道无并发限制
- 建议在管理后台「渠道设置」中按上游限制配置
- Redis 用于缓存 + 限流

---

## 五、告警（已于 2026-08-02 部署）

Alertmanager 已上线（容器 `gateway-alertmanager`，仅 127.0.0.1:9093）。4 条核心规则已配置（`observability/prometheus/rules/yuxin.yml`）：NewAPIDown / APIHighLatency / APIErrorRateHigh / DiskUsageHigh。

**通知渠道待配置**：编辑 `observability/alertmanager/alertmanager.yml` 的 webhook url（当前为占位 `http://127.0.0.1:5001/`），改为你的钉钉/飞书/企业微信 webhook 或 SMTP，然后 `docker compose -f docker-compose.observability.yml up -d alertmanager`。

**备份**：已自动化，cron 每日 02:00 运行 `scripts/auto-backup.sh`（PG+Redis+配置，7 天保留，产物在 `backups/auto/`）。

**Grafana 访问**：仅内网（127.0.0.1:3001）。远程访问用 SSH 隧道：`ssh -L 3001:127.0.0.1:3001 hk5708`，浏览器开 `localhost:3001`，账号 admin，密码见 `.env` 的 `GF_SECURITY_ADMIN_PASSWORD`。

### 原建议接入方案（已被上方实际部署取代）


建议接入告警通道：
- Prometheus AlertManager → 邮件 / 钉钉 / 飞书
- 告警规则建议（PromQL）：
  - 渠道失败率 > 10% 持续 5 分钟
  - API P99 > 5s 持续 5 分钟
  - 磁盘占用 > 85%
  - 任意容器 unhealthy 持续 3 分钟

---

## 六、安全管理

### 6.1 密钥轮换

- 上游渠道 KEY：管理后台「渠道」页面，按月轮换
- 数据库密码：见 BACKUP.md 章节五
- JWT SESSION_SECRET：见 BACKUP.md 章节五

### 6.2 访问审计

- 管理后台所有写操作有审计日志（`/admin/logs`）
- API 调用日志保留 90 天（PostgreSQL）+ 1 年（ClickHouse）

### 6.3 安全更新

```bash
# 每月检查依赖漏洞
cd /root/projects/api-gateway
govulncheck ./... 2>&1 | tee evidence/govulncheck-$(date +%Y%m%d).txt
gitleaks detect --source . 2>&1 | tee evidence/gitleaks-$(date +%Y%m%d).txt
```

---

## 七、自检验收（2026-07-27 hotfix.2）

> 适用场景：每次升级后、交付前、客户要求验收时。
> 完整证据保存在 `evidence/final-verify-YYYYMMDD.txt`。

### 7.1 一键验收脚本

```bash
ssh feifei <<'REMOTE'
set +e
export PATH=/usr/local/go/bin:$HOME/.nvm/versions/node/v24.18.0/bin:$HOME/.bun/bin:$PATH
cd /root/projects/api-gateway

echo "=== [1] Container health ==="
docker ps --format 'table {{.Names}}\t{{.Status}}'

echo "=== [2] Public API smoke ==="
for u in /api/status /api/status_page /api/public/pricing /api/routing/config \
         /api/mcp/tools /api/canary/status /api/compliance/config; do
  printf '%-28s ' "$u"
  curl -s -o /dev/null -w 'HTTP %{http_code} %{time_total}s\n' --max-time 5 "http://127.0.0.1$u"
done

echo "=== [3] Auth gate ==="
curl -s -o /dev/null -w 'no-token  /api/user/self -> HTTP %{http_code}\n' \
  --max-time 5 http://127.0.0.1/api/user/self

echo "=== [4] Prometheus metrics ==="
curl -s 'http://127.0.0.1:9090/api/v1/label/__name__/values' \
  | python3 -c 'import json,sys; print(*[n for n in json.load(sys.stdin)["data"] if n.startswith("yuxin_")],sep="\n")'

echo "=== [5] Go quality gates ==="
go build ./... && echo "build: PASS" || echo "build: FAIL"
go vet ./common/... 2>&1 | grep -q custom-event && echo "vet custom-event: FAIL" || echo "vet custom-event: PASS"
go test -short -count=1 ./common/... ./service/{routing,mcp,guardrail,canary,metrics}/... 2>&1 | tail -7

echo "=== [6] Frontend gates ==="
cd web
bun run typecheck 2>&1 | tail -3
bun run build 2>&1 | tail -3
bun run lint 2>&1 | tail -3
REMOTE
```

### 7.2 完整工作流验收

```bash
# 注册测试账号
curl -s -X POST http://127.0.0.1/api/user/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"test_acceptance_001","password":"Test@Pass1234"}'

# 登录拿 token
TOKEN=$(curl -s -X POST http://127.0.0.1/api/user/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"test_acceptance_001","password":"Test@Pass1234"}' \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["access_token"])')

# 鉴权后访问
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1/api/user/self | python3 -m json.tool

# 合规检测-PII
curl -s -X POST http://127.0.0.1/api/compliance/check \
  -H 'Content-Type: application/json' \
  -d '{"text":"我的手机 13812345678，身份证 110101199001011234"}'

# 清理测试账号（交付前必做）
docker exec gateway-postgres psql -U postgres -d new-api -c \
  "DELETE FROM users WHERE username = 'test_acceptance_001';"
```

### 7.3 验收基线（hotfix.2 实测）

| 检查项 | 基线 | 不通过时排查 |
|---|---|---|
| 7 容器 healthy | 7/7 | `docker logs <name> --tail 50` |
| 公开 API 200 | 9/9 | `curl -v` 看 nginx 日志 |
| 鉴权 401 vs 200 | 严格分级 | 检查 JWT SESSION_SECRET |
| Prometheus yuxin_* 指标 | >=5 | `docker logs gateway-prometheus` |
| Go build | 0 错 | 看 `evidence/go-build.txt` |
| Go vet custom-event | 0 警告 | 没修就回滚到 hotfix.1 commit `99ebd126` |
| Frontend typecheck/build | exit 0 | 看 `evidence/web-typecheck.txt` |
| Frontend lint | < 400 errors | `bun run lint --fix` 自动修 |

### 7.4 已知非问题（避免误判）

1. **host:3000 curl 不通**：new-api 端口不映射到 host，外部走 nginx:80
2. **ClickHouse host:8123 不通**：仅 docker network 内部可达，容器本身 healthy
3. **`/api/models` 无 token 401**：正常，需要 Bearer

---

