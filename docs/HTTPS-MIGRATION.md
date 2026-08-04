# HTTPS 证书切换手册(自签 → Let's Encrypt)

> 版本: v1.0.0-yuxin · 2026-08-03
> 适用: feifei 服务器(103.55.131.130)
> 当前状态: **自签证书**(nginx/ssl/selfsigned.*),浏览器会提示不安全
> 目标状态: Let's Encrypt 正式证书,浏览器无警告

## 一、当前架构(已生效)

- nginx 容器已监听 80 + 443(compose 已映射)
- 80 端口:仅放行 `/metrics`(内网)与 ACME 挑战路径,其余 301 → HTTPS
- 443 端口:TLS 1.2/1.3,完整安全头(HSTS/CSP/COOP/CORP/Permissions-Policy)
- 证书挂载:`./nginx/ssl:/etc/nginx/ssl:ro`
- 应用层:`SESSION_COOKIE_SECURE=true` + `SESSION_COOKIE_TRUSTED_URL=https://103.55.131.130` 已启用

## 二、切换前置条件

1. **域名解析**:将一个真实域名(如 `api.yuxin.net`)的 A 记录指向 `103.55.131.130`
   - 注意:`yuxin.com` 目前指向 146.235.218.82,需确认是否改用该 IP 或启用新子域
2. 80 端口从公网可达(ufw 已放行)

## 三、切换步骤

### 3.1 安装 certbot

```bash
apt-get update && apt-get install -y certbot
```

### 3.2 申请证书(webroot 模式,不停机)

```bash
mkdir -p /root/projects/api-gateway/nginx/ssl/acme
# 临时在 nginx 配置里确认 /.well-known/acme-challenge/ 指向 /var/www/certbot
certbot certonly --webroot -w /root/projects/api-gateway/nginx/ssl/acme \
  -d api.yuxin.net --agree-tos -m admin@yuxin.net --no-eff-email
```

### 3.3 更新 nginx 配置

编辑 `nginx/conf.d/gateway.conf`:

```nginx
# 替换这两行:
ssl_certificate     /etc/nginx/ssl/selfsigned.crt;
ssl_certificate_key /etc/nginx/ssl/selfsigned.key;
# 为:
ssl_certificate     /etc/nginx/ssl/acme/live/api.yuxin.net/fullchain.pem;
ssl_certificate_key /etc/nginx/ssl/acme/live/api.yuxin.net/privkey.pem;
```

### 3.4 更新 compose 挂载

`docker-compose.yml` nginx volumes 增加:

```yaml
- ./nginx/ssl/acme:/etc/nginx/ssl/acme:ro
```

### 3.5 更新应用配置

`.env`:

```
SESSION_COOKIE_TRUSTED_URL=https://api.yuxin.net
```

`middleware/cors.go` 白名单:

```go
"https://api.yuxin.net",
```

### 3.6 重启生效

```bash
cd /root/projects/api-gateway
docker compose up -d nginx
docker compose restart new-api   # 读取新的 TRUSTED_URL
```

### 3.7 自动续期

```bash
# certbot 自动续期定时器默认已启用,验证:
certbot renew --dry-run
```

## 四、回滚

若新证书配置出错,恢复自签:

```bash
cd /root/projects/api-gateway
cp /tmp/gateway.conf.bak nginx/conf.d/gateway.conf
docker compose restart nginx
```

## 五、验证清单

- [ ] `curl -sI https://api.yuxin.net/api/status` 无 `-k` 不报错
- [ ] 浏览器访问无证书警告
- [ ] 响应头含 `strict-transport-security`
- [ ] 登录流程 Cookie 带 `Secure; HttpOnly; SameSite=Strict`
- [ ] `http://api.yuxin.net/` 301 跳转 https
