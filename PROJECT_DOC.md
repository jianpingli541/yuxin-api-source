# 豫鑫 API 中转站 — 项目交接文档
# 最后更新: 2026-07-25
# 公司: 惠州市豫鑫网络科技有限公司

## 一、项目概述

### 项目名称
豫鑫 API · AI 模型网关 (YUXIN API Gateway)

### 项目定位
商业级 AI API 中转站，聚合 OpenAI / Claude / Gemini 等主流大模型。

### 技术基座
基于 new-api (QuantumNous/new-api) v1.0.0-rc.21 二次开发。
许可证: AGPL-3.0 | 当前版本: v1.0.0-yuxin

### 服务器
| 项目 | 值 |
|------|-----|
| SSH | ssh feifei 或 ssh root@103.55.131.130 -p 11572 |
| OS | Ubuntu 24.04.4 LTS, 40核/62GB/915GB |
| 项目目录 | /root/projects/api-gateway/ |
| 公网地址 | http://103.55.131.130 |

## 二、架构

```
Nginx(:80) → new-api(:3000) → PostgreSQL(:5432) / Redis(:6379) / ClickHouse(:9000)
```

5个Docker容器: gateway-nginx / gateway-new-api / gateway-postgres / gateway-redis / gateway-clickhouse

## 三、新增/修改文件 (★标记)

新增文件:
- controller/status_page.go          ★ 公开状态页 API
- service/cache/response_cache.go     ★ API响应缓存
- service/webhook/event_notify.go     ★ Webhook事件通知
- service/guardrail/safety_filter.go  ★ 内容安全过滤
- web/src/routes/docs/index.tsx       ★ API文档页
- Dockerfile.custom                   ★ 自定义镜像
- manage.sh                           ★ 运维脚本
- nginx/conf.d/gateway.conf           ★ Nginx反代配置
- web/public/logo.svg                 ★ Logo矢量图
- web/public/logo.png                 ★ Logo图片
- web/public/favicon.ico              ★ 浏览器图标

修改文件:
- service/channel.go                  ★ 注入webhook通知
- web/index.html                      ★ 品牌化meta标签
- web/src/assets/logo.tsx             ★ Logo组件
- web/src/components/layout/components/footer.tsx  ★ 版权信息
- web/src/styles/theme-presets.css    ★ yuxin-tech主题
- web/src/i18n/locales/zh.json        ★ 品牌文案
- web/src/i18n/locales/en.json        ★ 品牌文案
- web/src/lib/constants.ts            ★ 默认系统名
- router/api-router.go                ★ 新增路由

## 四、核心设计

### Ability路由矩阵
abilities表: group + model + channel_id → enabled + priority + weight
路由逻辑: 按用户分组+模型名匹配 → 优先级最高渠道 → 同优先级随机加权

### 预扣额度计费
请求前预扣(防超额) → 请求后实扣(多退少补)

### 倍率计费
最终费用 = prompt_tokens × model_ratio × group_ratio

## 五、运维命令

```bash
ssh feifei && cd /root/projects/api-gateway

./manage.sh start        # 启动
./manage.sh stop         # 停止
./manage.sh restart      # 重启
./manage.sh status       # 状态
./manage.sh logs         # 日志
./manage.sh backup       # 备份
./manage.sh update       # 更新版本
```

重新编译部署:
```bash
cd /root/projects/api-gateway/web && npm run build
cd ..
export PATH=$PATH:/usr/local/go/bin
CGO_ENABLED=0 go build -ldflags="-s -w" -o new-api-custom .
docker build -f Dockerfile.custom -t yuxin-api:latest .
docker compose up -d new-api
```

## 六、编译环境
- Go 1.22.5 (/usr/local/go/)
- Node.js 24.18.0 (via nvm)
- Docker 29.6.2 / Compose v5.3.1
- Go框架: Gin + GORM + go-redis/v8

## 七、待完成 (TODO)

P0 紧急:
- 接入真实上游API Key (后台→渠道管理)
- SMTP邮件配置+开启邮箱验证
- 支付集成(支付宝/微信/Stripe)

P1 重要:
- 状态页前端UI(/status可视化)
- Webhook后台配置UI
- Guardrail后台配置UI
- 域名绑定+SSL

P2 增强:
- 响应缓存中间件集成到relay
- 团队/子账号(Casbin RBAC)
- 多区域部署

## 八、已知问题
1. ClickHouse健康检查显示unhealthy (不影响日志写入)
2. 直接插数据库的令牌不生效(必须通过API创建)
3. OpenAI不支持香港IP(用Azure或代理)
4. 版本号偶尔显示v0.0.0(确保编译带ldflags)

## 九、注意事项
1. AGPL-3.0许可证:对外服务需开源修改，建议获取商业授权
2. 定期 ./manage.sh backup
3. .env含所有密码，勿提交Git
4. ClickHouse默认占~300MB内存
5. 日志保留30天(LOG_SQL_CLICKHOUSE_TTL_DAYS=30)

参考文档: https://docs.newapi.pro
