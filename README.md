# 豫鑫 API 网关（Yuxin API Gateway）— 修改版源码公开仓库

本仓库是 **豫鑫 API 中转站** 基于 [New API](https://github.com/QuantumNous/new-api)
（AGPL-3.0）二次开发后的**修改版完整源码**，依据 **AGPL-3.0 第 13 条**
（网络服务用户提供 Corresponding Source 的义务）向所有用户公开。

当前公开版本: v1.2.1-yuxin

## 许可证

- 本项目整体遵循 **GNU AGPL-3.0**（见 `LICENSE`）
- 上游 New API 的署名与附加条款（`NOTICE` 第 7 条）完整保留
- 修改说明见 `CHANGELOG.md`（版本号以 `-yuxin` 后缀标记）

## 主要修改（相对上游 new-api）

- 四策略智能路由（优先级 / 成本优化 / 延迟优化 / 质量优先，支持 Header 动态切换）
- L1-L4 安全合规管道（Prompt 注入检测、PII 检测、内容审核、多维限流）
- Canary 渠道质量监控（定时用例 + 健康度评分 + 告警）
- 会话安全加固（服务端会话 + JWT/Refresh 轮换 + 复用检测 + 双版本号失效）
- 可观测性集成（Prometheus / Grafana / ClickHouse / Alertmanager）
- 微信公众号登录网关接入（wechat-server）
- MCP 协议网关扩展

## 构建

```bash
# 前端
cd web && bun install --frozen-lockfile && bun run build && cd ..
# 后端（需 Go 1.25+）
go build -ldflags "-s -w" -o new-api
# 或整体容器构建
docker build -t yuxin-api .
```

部署参考 `DEPLOYMENT.md` / `QUICKSTART.md`（文中示例 IP 已脱敏）。

## 上游项目

- New API: https://github.com/QuantumNous/new-api
- New API 商业授权咨询: support@quantumnous.com
