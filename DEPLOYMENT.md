# 部署手册 — 豫鑫 API 中转站

> **版本**: v1.0.0-yuxin
> **适用环境**: Ubuntu 22.04+ / Debian 12+
> **最后更新**: 2026-07-26

---

## 一、服务器要求

| 项目 | 最低 | 推荐 |
|---|---|---|
| CPU | 4 核 | 8 核+ |
| 内存 | 8 GB | 16 GB+ |
| 磁盘 | 50 GB | 100 GB+ SSD |
| 带宽 | 10 Mbps | 100 Mbps+ |
| 操作系统 | Ubuntu 22.04 LTS | Ubuntu 24.04 LTS |
| Docker | 24.0+ | 最新稳定版 |
| Docker Compose | v2.20+ | 最新稳定版 |

---

## 二、首次部署（Step-by-Step）

### 2.1 安装 Docker

```bash
# 官方脚本一键安装
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker
docker --version  # 验证 ≥ 24.0
```

### 2.2 克隆代码

```bash
# 私有仓库，需要 SSH key 或 PAT
git clone git@github.com:jianpingli541/uxin-api-gateway.git /root/projects/api-gateway
cd /root/projects/api-gateway
```

### 2.3 配置环境变量

```bash
cp .env.example .env
vim .env
```

**必填项**（详见 `.env.example` 注释）：
- `POSTGRES_PASSWORD` — 数据库密码（**必须强密码**）
- `REDIS_PASSWORD` — Redis 密码
- `SESSION_SECRET` — 会话密钥（**必须 32 字符随机串**）
- `CRYPTO_SECRET` — 加密密钥（**必须 32 字符随机串**）
- 上游渠道 KEY（按需配置）

生成强随机串：
```bash
openssl rand -hex 32
```

### 2.4 构建并启动

```bash
# 一键启动（推荐）
./manage.sh start

# 或手动
docker compose up -d
```

### 2.5 验证部署

```bash
# 1. 容器健康
./manage.sh status

# 2. API 健康
curl http://localhost/api/status | jq .

# 3. 跑验收脚本（见 acceptance/run-acceptance.sh）
bash acceptance/run-acceptance.sh
```

### 2.6 配置 Nginx 反向代理（可选，如已有上层 nginx）

```bash
# 复用项目内的 nginx 配置
cp nginx/conf.d/gateway.conf /etc/nginx/conf.d/
nginx -t && systemctl reload nginx
```

---

## 三、升级部署

### 3.1 滚动升级（推荐）

```bash
cd /root/projects/api-gateway
git pull uxin main

# 拉新镜像 + 滚动重启
docker compose pull new-api
docker compose up -d new-api

# 验证
./manage.sh status
```

### 3.2 数据库迁移（如有 schema 变更）

```bash
# 备份当前数据库（必做！）
./manage.sh backup

# 跑迁移（容器内）
docker compose exec new-api ./new-api migrate
```

---

## 四、回滚

### 4.1 应用回滚

```bash
# 回到上一个 commit
git log --oneline -5
git checkout <prev_commit_sha>

docker compose up -d --build new-api
```

### 4.2 数据库回滚

```bash
# 从备份恢复
ls -lt /root/projects/api-gateway/backups/ | head -5
docker compose exec -T postgres psql -U gateway new-api < backups/db_<timestamp>.sql
```

---

## 五、域名 + HTTPS（可选）

```bash
# 用 Caddy 或 nginx + certbot
# 以 certbot 为例
apt install certbot python3-certbot-nginx
certbot --nginx -d api.yourdomain.com
```

---

