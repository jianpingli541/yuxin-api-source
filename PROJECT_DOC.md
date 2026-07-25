# 豫鑫 API 中转站 — 项目交接文档
# 最后更新: 2026-07-25 (第一阶段集成完成)
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
Prometheus(:9090) ← /metrics ← new-api
Grafana(:3001) → Prometheus
```

7个Docker容器: gateway-nginx / gateway-new-api / gateway-postgres / gateway-redis / gateway-clickhouse / gateway-prometheus / gateway-grafana

## 三、新增模块清单（第一阶段）

### 智能路由引擎 (service/routing/)
- types.go — 路由策略类型定义
- engine.go — 路由引擎核心（4种策略）
- config.go — 路由配置管理
- integration.go — 请求层集成
支持策略: priority_weight / cost_optimized / latency_optimized / quality_first

### MCP 协议网关 (service/mcp/)
- registry.go — 工具注册表
- tools.go — 内置工具（web_search / get_current_time / calculator）
- middleware.go — 请求拦截和工具注入中间件

### 可观测性 (service/metrics/)
- metrics.go — Prometheus 指标收集
指标: 请求计数/Token消耗/成本统计/渠道延迟/缓存命中率

### 公开 API
- controller/public_pricing.go — OpenRouter兼容定价API
- controller/status_page.go — 公开状态页
- controller/routing_admin.go — 路由策略管理

## 四、API 端点清单

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | /api/public/pricing | 无 | 公开定价（OpenRouter兼容） |
| GET | /api/status_page | 无 | 公开服务状态 |
| GET | /api/routing/config | 无 | 路由配置查看 |
| POST | /api/routing/config | 管理员 | 路由策略更新 |
| GET | /api/mcp/tools | 无 | MCP工具列表 |
| POST | /api/mcp/execute | 用户 | MCP工具执行 |
| GET | /api/mcp/config | 无 | MCP配置查看 |
| POST | /api/mcp/config | 管理员 | MCP配置更新 |
| GET | /metrics | 无 | Prometheus指标 |

## 五、运维命令

```bash
ssh feifei && cd /root/projects/api-gateway

./manage.sh start        # 启动
./manage.sh stop         # 停止
./manage.sh restart      # 重启
./manage.sh status       # 状态
./manage.sh logs         # 日志
./manage.sh backup       # 备份

# 可观测性服务
docker compose -f docker-compose.observability.yml up -d    # 启动监控
docker compose -f docker-compose.observability.yml down      # 停止监控

# 重新编译部署
cd web && npm run build && cd ..
export PATH=$PATH:/usr/local/go/bin
CGO_ENABLED=0 go build -ldflags="-s -w" -o new-api-custom .
docker build -f Dockerfile.custom -t yuxin-api:latest .
docker compose up -d new-api
```

## 六、可观测性面板

| 服务 | 地址 | 凭据 |
|------|------|------|
| Prometheus | http://103.55.131.130:9090 | 无 |
| Grafana | http://103.55.131.130:3001 | admin / yuxin2024 |

## 七、编译环境
- Go 1.22.5 (/usr/local/go/)
- Node.js 24.18.0 (via nvm)
- Docker 29.6.2 / Compose v5.3.1

## 八、待完成 (TODO)

### P0 紧急（第一阶段已完成 ✅）
- ✅ 公开定价 API
- ✅ 智能路由引擎
- ✅ Prometheus + Grafana 可观测性
- ✅ MCP 协议网关

### P1 重要（第二阶段）
- [ ] 安全合规升级（Prompt注入检测/PII脱敏/内容审核）
- [ ] 模型广场前端UI
- [ ] 在线 Playground
- [ ] Canary 质量监控
- [ ] 域名绑定+SSL

### P2 增强（第三阶段）
- [ ] AI 驱动智能运维
- [ ] 开放 API + 开发者生态
- [ ] 多租户企业版

## 九、已知问题
1. ClickHouse健康检查显示unhealthy (不影响日志写入)
2. 直接插数据库的令牌不生效(必须通过API创建)
3. OpenAI不支持香港IP(用Azure或代理)
4. 版本号偶尔显示v0.0.0(确保编译带ldflags)
