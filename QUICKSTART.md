# 豫鑫 API — 快速上手指南

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
./manage.sh logs       # new-api日志
./manage.sh backup     # 备份数据库+配置
./manage.sh shell new-api  # 进容器
```

## 管理后台

- 地址: http://103.55.131.130
- 登录后可管理: 渠道/用户/令牌/定价/系统设置

## 添加上游渠道

1. 登录管理后台
2. 渠道管理 → 添加渠道
3. 填写: 名称 / 类型(OpenAI/Claude等) / API Key / Base URL / 支持模型
4. 保存后自动注册到 abilities 路由表

## 创建用户令牌

1. 登录管理后台
2. 令牌管理 → 添加令牌
3. ⚠️ 必须通过界面或API创建，不能直接插数据库

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

## 重要文件位置

| 文件 | 用途 |
|------|------|
| .env | 所有密码和配置 |
| docker-compose.yml | 5个容器编排 |
| manage.sh | 运维一键脚本 |
| PROJECT_DOC.md | 完整项目文档 |
| nginx/conf.d/gateway.conf | Nginx反代 |
| controller/status_page.go | 状态页API |
| service/webhook/ | Webhook通知 |
| service/guardrail/ | 内容过滤 |
| service/cache/ | 响应缓存 |

## 详细文档

完整项目文档见同目录 `PROJECT_DOC.md`
原项目文档: https://docs.newapi.pro
