# 运维手册 — 豫鑫 API 中转站

> **版本**: v1.0.0-yuxin
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

## 五、告警（可选，建议接入）

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

