#!/usr/bin/env bash
# =============================================================================
# 豫鑫 API 中转站 — 验收脚本 v1.0.0-yuxin
# =============================================================================
# 用法：
#   1. 填入下面的 YUXIN_USER_TOKEN / YUXIN_ADMIN_TOKEN
#   2. bash acceptance/run-acceptance.sh
#
# 全部 PASS = 验收通过；任一 FAIL = 验收失败
# =============================================================================

set -uo pipefail

# ------------------- 验收前置变量（客户填）-------------------
export YUXIN_BASE_URL="${YUXIN_BASE_URL:-http://103.55.131.130}"
export YUXIN_USER_TOKEN="${YUXIN_USER_TOKEN:-<填入普通用户 token>}"
export YUXIN_ADMIN_TOKEN="${YUXIN_ADMIN_TOKEN:-<填入管理员 token>}"
export YUXIN_TEST_MODEL="${YUXIN_TEST_MODEL:-gpt-4o-mini}"

# ------------------- 工具函数 -------------------
PASS=0
FAIL=0
FAILED_ITEMS=()

check() {
    local name="$1"
    local cmd="$2"
    local expect="$3"
    local actual
    actual=$(eval "$cmd" 2>&1)
    if echo "$actual" | grep -qE "$expect"; then
        echo "✅ PASS  $name"
        PASS=$((PASS+1))
    else
        echo "❌ FAIL  $name"
        echo "        cmd:    $cmd"
        echo "        expect: $expect"
        echo "        actual: $(echo "$actual" | head -1)"
        FAIL=$((FAIL+1))
        FAILED_ITEMS+=("$name")
    fi
}

# ------------------- L1 公开页面 -------------------
echo ""
echo "======== L1 公开页面 ========"
check "L1.1 首页加载" \
    "curl -sS -o /dev/null -w '%{http_code}' $YUXIN_BASE_URL/" \
    "^200$"
check "L1.2 状态页加载" \
    "curl -sS -o /dev/null -w '%{http_code}' $YUXIN_BASE_URL/status" \
    "^200$"
check "L1.3 定价页加载" \
    "curl -sS -o /dev/null -w '%{http_code}' $YUXIN_BASE_URL/pricing" \
    "^200$"

# ------------------- L2 认证流程 -------------------
echo ""
echo "======== L2 认证流程 ========"
check "L2.1 登录页" \
    "curl -sS -o /dev/null -w '%{http_code}' $YUXIN_BASE_URL/login" \
    "^200$"
check "L2.2 注册页" \
    "curl -sS -o /dev/null -w '%{http_code}' $YUXIN_BASE_URL/register" \
    "^200$"
check "L2.3 错误登录被拒" \
    "curl -sS -X POST $YUXIN_BASE_URL/api/user/login -H 'Content-Type: application/json' -d '{\"username\":\"nouser12345\",\"password\":\"wrongpass\"}'" \
    '"success":\s*false'

# ------------------- L5 API 链路 -------------------
echo ""
echo "======== L5 API 链路 ========"
check "L5.1 系统状态 API" \
    "curl -sS $YUXIN_BASE_URL/api/status" \
    '"success":\s*true'
check "L5.3 公开定价 API" \
    "curl -sS $YUXIN_BASE_URL/api/pricing" \
    '"data"'
check "L5.4 chat completions（用户 token）" \
    "curl -sS $YUXIN_BASE_URL/v1/chat/completions -H 'Authorization: Bearer $YUXIN_USER_TOKEN' -H 'Content-Type: application/json' -d '{\"model\":\"'\$YUXIN_TEST_MODEL'\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"max_tokens\":10}'" \
    '"choices"'
check "L5.6a 无 token 被拒" \
    "curl -sS -o /dev/null -w '%{http_code}' $YUXIN_BASE_URL/v1/chat/completions -H 'Content-Type: application/json' -d '{\"model\":\"'\$YUXIN_TEST_MODEL'\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}]}'" \
    "^401$"
check "L5.6b 错 token 被拒" \
    "curl -sS -o /dev/null -w '%{http_code}' $YUXIN_BASE_URL/v1/chat/completions -H 'Authorization: Bearer sk-invalid-token' -H 'Content-Type: application/json' -d '{\"model\":\"'\$YUXIN_TEST_MODEL'\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}]}'" \
    "^401$"

# ------------------- L8 可观测性 -------------------
echo ""
echo "======== L8 可观测性 ========"
check "L8.1 Prometheus 指标端点" \
    "curl -sS $YUXIN_BASE_URL/metrics | head -50" \
    "(new_api_|yuxin_|process_|go_)"

# ------------------- 汇总 -------------------
echo ""
echo "=========================================="
echo "  验收汇总: $PASS PASS / $FAIL FAIL"
echo "=========================================="
if [ $FAIL -gt 0 ]; then
    echo ""
    echo "失败项："
    for item in "${FAILED_ITEMS[@]}"; do
        echo "  - $item"
    done
    exit 1
fi
exit 0
