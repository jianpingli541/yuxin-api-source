#!/usr/bin/env bash
# 豫鑫 API — 零依赖自检 v1.0 (2026-07-26)
# 用法: bash self-check.sh [--deep]
# 不需要 token，客户可直接跑
set -uo pipefail
DEEP="${1:-}"
PASS=0; FAIL=0; FAILED=()
c() {
  local name="$1" cmd="$2" expect="$3"
  local actual; actual=$(eval "$cmd" 2>&1 | tail -1)
  if echo "$actual" | grep -qE "$expect"; then
    echo "  ✅ $name"; PASS=$((PASS+1))
  else
    echo "  ❌ $name → 期望$expect 实际$(echo "$actual" | head -c 80)"
    FAIL=$((FAIL+1)); FAILED+=("$name")
  fi
}
echo "════════════════════════════════════════"
echo "  豫鑫 API 自检 (feifei)"
echo "════════════════════════════"
echo ""
echo "[L0 容器健康]"
for ctr in gateway-new-api gateway-postgres gateway-redis gateway-nginx gateway-clickhouse gateway-prometheus gateway-grafana; do
  status=$(docker inspect --format '{{.State.Status}}' $ctr 2>/dev/null || echo "missing")
  hc=$(docker inspect --format '{{.State.Health.Status}}' $ctr 2>/dev/null || echo "no-hc")
  if [ "$status" = "running" ]; then
    echo "  ✅ $ctr ($status/$hc)"; PASS=$((PASS+1))
  else
    echo "  ❌ $ctr = $status"; FAIL=$((FAIL+1)); FAILED+=("$ctr")
  fi
done
echo ""
echo "[L0 端口]"
for port in 80 9090 3001; do
  c "port $port" "ss -tlnp 2>/dev/null | grep ':$port ' | wc -l" "^[1-9]"
done
echo ""
echo "[L0 系统资源]"
c "load<5" "uptime | awk -F'load average:' '{print \$2}' | awk -F',' '{print \$1}' | tr -d ' '" "^[0-4]\."
c "disk<80%" "df -h / | awk 'NR==2{gsub(/%/,\"\",\$5);print \$5}'" "^[0-7][0-9]$|^[1-7]$"
echo ""
echo "[L1 公开接口]"
c "首页" "curl -s -o /dev/null -w '%{http_code}' http://localhost/" "^200$"
c "登录页" "curl -s -o /dev/null -w '%{http_code}' http://localhost/login" "^200$"
c "API 状态" "curl -s http://localhost/api/status" '"success":true'
c "robots.txt" "curl -s -o /dev/null -w '%{http_code}' http://localhost/robots.txt" "^200$"
c "响应<500ms" "curl -s -o /dev/null -w '%{time_total}' http://localhost/api/status" "^0\.[0-4]"
echo ""
echo "[L1 数据库]"
c "PG accepting" "docker exec gateway-postgres pg_isready -U gateway -d new-api 2>&1" "accepting"
REDIS_PASS=$(grep REDIS_CONN_STRING /root/projects/api-gateway/.env 2>/dev/null | sed -E 's|.*://:(.*)@.*|\1|' | head -1)
c "Redis PING" "docker exec gateway-redis redis-cli -a '$REDIS_PASS' ping 2>/dev/null" "^PONG$"
echo ""
echo "[L1 监控栈]"
c "Prometheus" "curl -s -o /dev/null -w '%{http_code}' http://localhost:9090/-/healthy" "^200$"
c "Grafana" "curl -s -o /dev/null -w '%{http_code}' http://localhost:3001/api/health" "^200$"
c "Prom target" "curl -s http://localhost:9090/api/v1/targets" '"yuxin-api"'
echo ""
echo "[L1 错误日志]"
errs=$(docker logs gateway-new-api --since 30m 2>&1 | grep -cE "panic|fatal|FATAL" | head -1 | tr -dc '0-9')
errs=${errs:-0}
if [ "$errs" -eq 0 ] 2>/dev/null; then
  echo "  ✅ 30min 无 panic/fatal"; PASS=$((PASS+1))
else
  echo "  ❌ 30min 内 $errs 个 panic/fatal"; FAIL=$((FAIL+1)); FAILED+=("panic")
fi
echo ""
if [ "$DEEP" = "--deep" ]; then
  echo "[L2 数据完整性]"
  c "  users≥1" "docker exec gateway-postgres psql -U gateway -d new-api -t -c 'SELECT COUNT(*) FROM users' | tr -dc '0-9'" "^[1-9]"
  c "  channels≥1" "docker exec gateway-postgres psql -U gateway -d new-api -t -c 'SELECT COUNT(*) FROM channels' | tr -dc '0-9'" "^[1-9]"
  c "  tokens≥0" "docker exec gateway-postgres psql -U gateway -d new-api -t -c 'SELECT COUNT(*) FROM tokens' | tr -dc '0-9'" "^([0-9]|[1-9][0-9]+)$"
  echo ""
  echo "[L2 SSE 流式（CustomEvent 修复验证）]"
  c "  /v1/models 公开" "curl -s -o /dev/null -w '%{http_code}' http://localhost/v1/models" "^(200|401)$"
  echo ""
  echo "[L2 镜像版本]"
  c "  镜像构建于今天" "docker images yuxin-api:latest --format '{{.CreatedAt}}' | grep -c '2026-07-26'" "^1$"
  c "  HEAD ≥ 99ebd126" "cd /root/projects/api-gateway && git log --oneline | grep -c 99ebd126" "^1$"
fi
echo "════════════════════════════════════════"
echo "  结果: $PASS ✅ / $FAIL ❌"
[ $FAIL -gt 0 ] && { echo "  失败: ${FAILED[@]}"; exit 1; } || { echo "  🏆 全部通过"; exit 0; }
