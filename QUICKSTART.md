# 豫鑫 API 中转站 — 快速上手指南

> **版本**: v1.2.1-yuxin | **更新**: 2026-08-04

## 首次接管

```bash
# 1. SSH 登录服务器
ssh feifei

# 2. 进入项目目录
cd /root/projects/api-gateway

# 3. 检查服务状态
./manage.sh status

# 4. 查看日志确认正常
./manage.sh logs
```

## 日常运维

```bash
./manage.sh start      # 启动全部服务
./manage.sh stop       # 停止
./manage.sh restart    # 重启
./manage.sh status     # 看容器+资源
./manage.sh logs       # new-api 日志
./manage.sh backup     # 备份数据库+配置
```

## 关键地址

| 服务 | 地址 |
|------|------|
| 网站首页 | http://203.0.113.10 |
| 服务状态页 | http://203.0.113.10/status-page |
| 模型定价页 | http://203.0.113.10/pricing-page |
| Prometheus | 仅内网 127.0.0.1:9090，SSH 隧道访问 |
| Grafana | 仅内网，SSH 隧道 `ssh -L 3001:127.0.0.1:3001 hk5708` 后开 localhost:3001 (admin/见 .env GF_SECURITY_ADMIN_PASSWORD) |

## 管理后台

- 地址: http://203.0.113.10
- 管理员: `lijianping`
- 登录后可管理: 渠道 / 用户 / 令牌 / 定价 / 系统设置

## 添加上游渠道

1. 登录管理后台
2. 渠道管理 → 添加渠道
3. 填写: 名称 / 类型(OpenAI/Claude等) / API Key / Base URL / 支持模型
4. 保存后自动注册到 abilities 路由表

## 创建用户令牌

1. 登录管理后台
2. 令牌管理 → 添加令牌
3. ⚠️ **必须通过界面或 API 创建，不能直接插数据库**

## 修改代码后部署

```bash
cd /root/projects/api-gateway
export NVM_DIR="$HOME/.nvm" && . "$NVM_DIR/nvm.sh"
export PATH=$PATH:/usr/local/go/bin

# 前端
cd web && npm run build && cd ..

# 后端
CGO_ENABLED=0 go build -ldflags="-s -w" -o new-api-custom .

# 镜像
docker build -f Dockerfile.custom -t yuxin-api:latest .

# 部署
docker compose up -d new-api
```

## 重置管理员密码

```bash
# 1. 生成 bcrypt 哈希
python3 -c "import bcrypt;print(bcrypt.hashpw(b'新密码',bcrypt.gensalt()).decode())"

# 2. 更新数据库（注意 $ 转义）
docker exec gateway-postgres psql -U gateway -d new-api \
  -c "UPDATE users SET password = E'\$2b\$12\$xxx完整哈希xxx' WHERE username = 'lijianping';"

# 3. 清除缓存 + 重启
docker exec gateway-redis redis-cli -a <密码> FLUSHALL
docker compose restart new-api
```

## 重要文件

| 文件 | 用途 |
|------|------|
| `.env` | 所有密码和配置 |
| `docker-compose.yml` | 主容器编排 |
| `docker-compose.observability.yml` | 监控服务 |
| `nginx/conf.d/gateway.conf` | Nginx 反代+限速 |
| `PROJECT_DOC.md` | 完整项目文档 |
| `manage.sh` | 运维一键脚本 |

## 常见问题

**Q: 改了密码登录不上？**  
A: 需要 `redis-cli FLUSHALL` 清缓存 + `docker compose restart new-api`

**Q: API 调用返回错误？**  
A: 检查渠道是否配置了有效的上游 API Key

**Q: ClickHouse 显示 unhealthy？**  
A: 不影响功能，是健康检查脚本兼容性问题

**Q: Nginx 返回 429/503？**  
A: 触发了限速，检查 `nginx/conf.d/gateway.conf` 中的 `rate` 和 `burst` 值
