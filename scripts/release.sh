#!/bin/bash
# ============================================================
# 豫鑫 API 发布流水线
# 用法:
#   release.sh <镜像标签> [--skip-web]
#   例: release.sh v1.2.5-yuxin
# 流程:
#   1) 构建前端 (bun) + 后端 (go) + 镜像
#   2) 生成 AGPL 源码披露快照并推送公开仓库（推送失败则中止发布,
#      保持旧版本运行, 避免出现"新代码已上线但源码未披露"的合规缺口）
#   3) 更新 compose 镜像标签, 重启 new-api, 健康检查
# ============================================================
set -euo pipefail

DIR=/root/projects/api-gateway
TAG="${1:-}"
SKIP_WEB=0
[ "${2:-}" = "--skip-web" ] && SKIP_WEB=1

if [ -z "$TAG" ]; then
  echo "用法: $0 <镜像标签> [--skip-web]"
  echo "例:   $0 v1.2.5-yuxin"
  exit 1
fi
IMAGE="yuxin-api:${TAG}"
cd "$DIR"

# [版本纪律] 从 TAG 提取版本号写回 VERSION 文件（build 前定稿）。
# 根因: go build 经 -ldflags -X common.Version=$(cat VERSION) 注入版本,
# 若 VERSION 滞后则二进制版本与镜像 tag 漂移(2026-08-19 实测: 部署1.2.21/上报1.2.20)。
VER_FROM_TAG="${TAG#v}"          # v1.2.21-yuxin-guide-drawer -> 1.2.21-yuxin-guide-drawer
VER_SHORT="${VER_FROM_TAG%%-*}"  # -> 1.2.21 (与历史 VERSION 文件格式一致: 主版本-yuxin)
CUR_VERSION="$(cat VERSION 2>/dev/null || echo '')"
if [ -z "$CUR_VERSION" ] || [[ "$VER_FROM_TAG" != "$CUR_VERSION"* ]]; then
  # TAG 主版本与 VERSION 文件前缀不一致 -> 以 TAG 为准写回(保留 -yuxin 后缀)
  NEW_VERSION="${VER_SHORT}-yuxin"
  echo "📝 [版本纪律] VERSION: ${CUR_VERSION} -> ${NEW_VERSION} (跟随 TAG ${TAG})"
  echo "${NEW_VERSION}" > VERSION
else
  echo "📝 [版本纪律] VERSION=${CUR_VERSION} 与 TAG ${TAG} 一致, 无需改写"
fi

echo "🏗  [1/6] 构建前端..."
if [ "$SKIP_WEB" = "1" ]; then
  echo "   (--skip-web: 跳过, 使用现有 web/dist)"
else
  ( cd web && bun install --frozen-lockfile && \
    DISABLE_ESLINT_PLUGIN=true VITE_REACT_APP_VERSION="$(cat ../VERSION)" bun run build ) \
    > /tmp/release-web.log 2>&1 || { echo "❌ 前端构建失败: tail /tmp/release-web.log"; exit 1; }
  echo "   前端构建完成"
fi

echo "🏗  [2/6] 构建 Go 二进制..."
/usr/local/go/bin/go build \
  -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=$(cat VERSION)" \
  -o new-api-custom > /tmp/release-go.log 2>&1 \
  || { echo "❌ Go 构建失败: tail /tmp/release-go.log"; exit 1; }
echo "   new-api-custom 构建完成"

echo "🏗  [3/6] 构建镜像 $IMAGE..."
docker build -f Dockerfile.custom -t "$IMAGE" . > /tmp/release-docker.log 2>&1 \
  || { echo "❌ 镜像构建失败: tail /tmp/release-docker.log"; exit 1; }
echo "   镜像构建完成"

echo "📤 [4/6] AGPL 源码披露快照推送（合规门禁）..."
bash "$DIR/scripts/snapshot-source.sh" --push \
  || { echo "❌ 源码披露失败, 发布中止（旧版本继续运行, 合规无缺口）"; exit 1; }

echo "🚀 [5/6] 部署..."
cp docker-compose.yml "docker-compose.yml.bak.$(date +%Y%m%d%H%M%S)"
python3 - "$TAG" <<'EOF'
import re, sys
tag = sys.argv[1]
p = "docker-compose.yml"
s = open(p).read()
s2, n = re.subn(r"image: yuxin-api:[^\s]+", f"image: yuxin-api:{tag}", s, count=1)
assert n == 1, "compose 中未找到 yuxin-api 镜像行"
open(p, "w").write(s2)
print(f"compose 镜像标签: yuxin-api:{tag}")
EOF
docker compose up -d new-api

echo "🩺 [6/6] 健康检查..."
for i in $(seq 1 20); do
  sleep 5
  if curl -s -m 5 https://ai.yuxin.yun/api/status | grep -q '"success":true'; then
    # [版本纪律] 校验运行中二进制版本与 TAG 一致(防 build 顺序漂移复发)
    RUNTIME_VER="$(curl -s -m 5 https://ai.yuxin.yun/api/status | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"].get("version",""))' 2>/dev/null || echo '')"
    EXPECT_VER="$(cat VERSION)"
    if [ "$RUNTIME_VER" != "$EXPECT_VER" ]; then
      echo "⚠️  [版本纪律] 运行版本(${RUNTIME_VER})与 VERSION(${EXPECT_VER})不一致, 请检查 build 顺序"
    else
      echo "   版本一致: ${RUNTIME_VER}"
    fi
    echo "✅ 发布完成: $IMAGE"
    docker ps --filter name=gateway-new-api --format "   {{.Status}}"
    echo "   披露仓库已同步: https://github.com/jianpingli541/yuxin-api-source"
    exit 0
  fi
done
echo "❌ 健康检查超时（100s）, 请检查: docker logs gateway-new-api"
exit 1
