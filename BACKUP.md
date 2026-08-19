# 备份恢复手册 — 豫鑫 API 中转站

> **版本**: v1.0.0-yuxin
> **最后更新**: 2026-07-26

---

## 一、备份范围

| 组件 | 内容 | 位置 |
|---|---|---|
| PostgreSQL | 业务数据（用户/订单/渠道/日志）| Docker volume `gateway_pg_data` |
| Redis | 缓存 + 会话（**可丢，重建即可**）| Docker volume `gateway_redis_data` |
| ClickHouse | 长期日志/统计 | Docker volume `gateway_clickhouse_data` |
| 配置 | `.env` + `nginx/conf.d/` | 主机文件 |
| 代码 | 完整 git 仓库 | GitHub `uxin-api-gateway` |

**关键备份**：PostgreSQL + `.env` + nginx 配置。其他可重建。

---

## 二、自动备份（推荐）

### 2.1 用 cron 每天 03:00 备份

```bash
crontab -e
# 加入：
0 3 * * * cd /root/projects/api-gateway && ./manage.sh backup >> /var/log/yuxin-backup.log 2>&1
```

### 2.2 备份脚本逻辑（已在 manage.sh 实现）

- 命名：`backups/db_YYYYMMDD_HHMMSS.sql`
- 保留：默认 30 天
- 清理：`find backups/ -mtime +30 -delete`

---

## 三、手动备份

### 3.1 数据库

```bash
cd /root/projects/api-gateway
./manage.sh backup
# 或
docker compose exec -T postgres pg_dump -U gateway new-api > backups/db_$(date +%Y%m%d).sql
```

### 3.2 配置

```bash
tar czf backups/config_$(date +%Y%m%d).tar.gz .env nginx/conf.d/
```

### 3.3 完整快照（含 ClickHouse，较慢）

```bash
docker run --rm \
  -v gateway_clickhouse_data:/data:ro \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/clickhouse_$(date +%Y%m%d).tar.gz /data
```

---

## 四、恢复

### 4.1 数据库恢复

```bash
# 1. 停服务（避免新写入）
./manage.sh stop

# 2. 恢复 SQL
docker compose up -d postgres  # 只启 postgres
docker compose exec -T postgres psql -U gateway new-api < backups/db_<timestamp>.sql

# 3. 启服务
./manage.sh start
```

### 4.2 配置恢复

```bash
tar xzf backups/config_<timestamp>.tar.gz
docker compose restart nginx new-api
```

### 4.3 灾难恢复（全换服务器）

1. 按 DEPLOYMENT.md 在新服务器装好
2. git clone uxin-api-gateway
3. 恢复 `.env`（从备份 tar）
4. 启动 postgres → 恢复 SQL → 启动其他服务
5. 跑 acceptance/run-acceptance.sh 验证

---

## 五、密钥管理（含轮换）

### 5.1 关键密钥清单

| 密钥 | 用途 | 轮换周期 |
|---|---|---|
| `SESSION_SECRET` | 用户会话签名 | 6 个月 |
| `CRYPTO_SECRET` | 渠道 KEY 加密 | 6 个月 |
| `POSTGRES_PASSWORD` | 数据库 | 1 年 |
| `REDIS_PASSWORD` | Redis | 1 年 |
| 上游渠道 KEY | 业务调用 | 按上游策略 |

### 5.2 轮换 SESSION_SECRET / CRYPTO_SECRET

> ⚠️ **轮换 SESSION_SECRET 会让所有用户登出**

```bash
# 1. 备份
cp .env .env.backup.$(date +%Y%m%d)

# 2. 生成新密钥
NEW_SECRET=$(openssl rand -hex 32)
sed -i "s/SESSION_SECRET=.*/SESSION_SECRET=$NEW_SECRET/" .env

# 3. 重启
docker compose restart new-api

# 4. 通知用户会话已重置
```

**CRYPTO_SECRET 轮换必须重新加密所有渠道 KEY**，建议联系开发支持。

---

## 六、备份验证（每月一次）

```bash
# 拉一份备份，恢复到测试容器，跑验收
docker run --rm -d --name pg-test \
  -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=new-api \
  -v $(pwd)/backups/db_latest.sql:/docker-entrypoint-initdb.d/restore.sql:ro \
  postgres:16

# 等 30 秒后验证
docker exec pg-test psql -U postgres new-api -c "SELECT COUNT(*) FROM users;"
docker rm -f pg-test
```

---

