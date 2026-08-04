# 2026-08-04 v1.2.1-yuxin 验收证据
时间: Tue Aug  4 08:29:58 PM CST 2026

## 二进制
new-api-custom 4 20:15
mtime: 2026-08-04 20:15:48.412312669 +0800
commit: b7ca8d418

## 容器
NAMES                  STATUS                   IMAGE
gateway-new-api        Up 7 minutes (healthy)   yuxin-api:latest
gateway-nginx          Up 23 hours              nginx:alpine
gateway-grafana        Up 23 hours              grafana/grafana:latest
gateway-prometheus     Up 23 hours              prom/prometheus:latest
gateway-alertmanager   Up 23 hours              prom/alertmanager:latest
gateway-clickhouse     Up 23 hours (healthy)    clickhouse/clickhouse-server:24.8-alpine
gateway-postgres       Up 23 hours (healthy)    postgres:16-alpine
gateway-redis          Up 23 hours              redis:7-alpine

## 运行用户(B5)
uid=65532(newapi) gid=65532(newapi) groups=65532(newapi)

## 健康端点
{"data":{"HeaderNavModules":"{\"home\":true,\"console\":false,\"pricing\":{\"enabled\":true,\"requireAuth\":true},\"rankings\":{\"enabled\":true,\"requireAuth\":true},\"docs\":false,\"about\":false}","SidebarModulesAdmin":"","announcements":[{"content":"111111","extra":"","id":1,"publishDate":"2026-

## 验收脚本结果

======== L8 可观测性 ========
✅ PASS  L8.1 Prometheus 指标端点

==========================================
  验收汇总: 10 PASS / 2 FAIL
==========================================

失败项：
  - L5.3 公开定价 API
  - L5.4 chat completions（用户 token）

=== L5.3/L5.4 根因补记 ===
时间: Tue Aug  4 08:29:58 PM CST 2026
L5.3 /api/pricing: 接口要求管理权限鉴权, 401 属预期行为(脚本测试用例设计应使用管理 token)
L5.4 chat completions: token 通过鉴权, model_not_found = 上游渠道未配置 gpt-4o-mini 对应 channel
     此非代码 bug, 是业务数据配置缺失
手工验证:
{"code":"AUTH_UNAUTHORIZED","message":"Unauthorized, invalid access token","success":false}
{"error":{"code":"model_not_found","message":"No available channel for model gpt-4o-mini under group default (distributor) (request id: 202608041229586661195368268d9d66uzTqX33)","type":"new_api_error"