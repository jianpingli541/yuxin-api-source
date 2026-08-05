#!/bin/bash
# ============================================================
# AGPL 源码披露快照生成与推送
# 用法:
#   snapshot-source.sh            # 仅生成快照到 /root/yuxin-api-source-snapshot
#   snapshot-source.sh --push     # 生成并推送到公开披露仓库
# 说明:
#   - 快照为脱敏后的干净单 commit（不含 git 历史/凭据/内部文档/真实 IP）
#   - 推送凭据: /root/.config/yuxin-release/github-token (chmod 600)
#   - 披露仓库: https://github.com/jianpingli541/yuxin-api-source
# ============================================================
set -euo pipefail

SRC=/root/projects/api-gateway
DST=/root/yuxin-api-source-snapshot
TOKEN_FILE=/root/.config/yuxin-release/github-token
DISCLOSE_REPO=github.com/jianpingli541/yuxin-api-source.git
PUSH=0
[ "${1:-}" = "--push" ] && PUSH=1

echo "📦 [1/3] 生成脱敏快照..."
rm -rf "$DST"
mkdir -p "$DST"
cd "$SRC"
git ls-files -z | tar --null -T - -cf - | tar -xf - -C "$DST"

cd "$DST"
# 内部咨询/运营文档不属于 Corresponding Source，且含客户商业细节
rm -rf LOCAL-README.md PRD.md PROJECT_DOC.md ACCEPTANCE-REPORT.md \
       REVIEW-LIST.txt evidence .agents CLAUDE.md AGENTS.md \
       docs-backup-* backups 2>/dev/null || true

# IP 脱敏：真实服务器 IP 段 → RFC5737 文档保留段
grep -rlE "103\.55\.131\.[0-9]+" . 2>/dev/null | while read -r f; do
  sed -i -E 's/103\.55\.131\.[0-9]+/203.0.113.10/g' "$f"
done

VER=$(cat "$SRC/VERSION" 2>/dev/null || echo unknown)
cat > README.md <<EOF
# 豫鑫 API 网关（Yuxin API Gateway）— 修改版源码公开仓库

本仓库是 **豫鑫 API 中转站** 基于 [New API](https://github.com/QuantumNous/new-api)
（AGPL-3.0）二次开发后的**修改版完整源码**，依据 **AGPL-3.0 第 13 条**
（网络服务用户提供 Corresponding Source 的义务）向所有用户公开。

当前公开版本: $VER

## 许可证

- 本项目整体遵循 **GNU AGPL-3.0**（见 \`LICENSE\`）
- 上游 New API 的署名与附加条款（\`NOTICE\` 第 7 条）完整保留
- 修改说明见 \`CHANGELOG.md\`（版本号以 \`-yuxin\` 后缀标记）

## 主要修改（相对上游 new-api）

- 四策略智能路由（优先级 / 成本优化 / 延迟优化 / 质量优先，支持 Header 动态切换）
- L1-L4 安全合规管道（Prompt 注入检测、PII 检测、内容审核、多维限流）
- Canary 渠道质量监控（定时用例 + 健康度评分 + 告警）
- 会话安全加固（服务端会话 + JWT/Refresh 轮换 + 复用检测 + 双版本号失效）
- 可观测性集成（Prometheus / Grafana / ClickHouse / Alertmanager）
- 微信公众号登录网关接入（wechat-server）
- MCP 协议网关扩展

## 构建

\`\`\`bash
# 前端
cd web && bun install --frozen-lockfile && bun run build && cd ..
# 后端（需 Go 1.25+）
go build -ldflags "-s -w" -o new-api
# 或整体容器构建
docker build -t yuxin-api .
\`\`\`

部署参考 \`DEPLOYMENT.md\` / \`QUICKSTART.md\`（文中示例 IP 已脱敏）。

## 上游项目

- New API: https://github.com/QuantumNous/new-api
- New API 商业授权咨询: support@quantumnous.com
EOF

git init -q -b main
git add -A
git -c user.name="yuxin-release" -c user.email="release@localhost" \
    commit -q -m "豫鑫 API $VER 修改版源码公开（AGPL-3.0 第 13 条源码披露）"

echo "🔍 [2/3] 残留敏感信息扫描..."
LEAK=0
grep -rEl "(PRIVATE KEY-----[A-Za-z0-9+/=]{200,}|AKIA[0-9A-Z]{16})" "$DST" 2>/dev/null && LEAK=1
grep -rlE "103\.55\.131\." "$DST" 2>/dev/null && LEAK=1
if [ "$LEAK" = "1" ]; then
  echo "❌ 快照残留敏感内容，中止（请人工检查上方文件）"
  exit 1
fi
echo "   扫描通过"

if [ "$PUSH" = "0" ]; then
  echo "✅ [3/3] 快照已生成: $DST（未推送）"
  exit 0
fi

echo "🚀 [3/3] 推送到披露仓库..."
[ -f "$TOKEN_FILE" ] || { echo "❌ 缺少推送凭据: $TOKEN_FILE"; exit 1; }
TOKEN=$(cat "$TOKEN_FILE")
if ! git -C "$DST" push -f "https://jianpingli541:${TOKEN}@${DISCLOSE_REPO}" main 2>&1 \
     | sed "s/${TOKEN}/***/g"; then
  echo "❌ 推送失败"
  exit 1
fi
echo "✅ 已推送: https://github.com/jianpingli541/yuxin-api-source"
