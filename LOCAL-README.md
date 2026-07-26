# uxin-api-gateway（本机工作副本）

> **⚠️ 这不是项目本体，是开发机上的工作副本。**

## 项目本体位置

| 位置 | 路径 / URL | 角色 |
|---|---|---|
| **GitHub 仓库（权威源）** | https://github.com/jianpingli541/uxin-api-gateway | 私有，所有 push/pull 的中心 |
| **生产服务器** | `feifei` (103.55.131.130) `/root/projects/api-gateway/` | 客户访问的运行实例 |
| **开发机工作副本（本目录）** | `/root/projects/active/uxin-api-gateway/` | 开发、调试、文档产出 |
| **草稿归档** | `/root/projects/active/uxin-api-gateway.drafts/` | 2026-07-26 之前在 /root 散落的草稿，已归档 |

## Remotes

- `origin` → 上游 new-api (https://github.com/Calcium-Ion/new-api.git)
- `uxin`  → 我们自己的私有仓库 (https://github.com/jianpingli541/uxin-api-gateway)

## 同步规则（避免历史重演）

1. **任何代码改动** → 在本机改 → `git push uxin main` → ssh feifei `git pull`
2. **不要在 feifei 上直接改代码**（生产服务器只 pull，不开发）
3. **生产部署** → 在 feifei 上 `git pull && docker compose build && docker compose up -d`

## 历史教训（2026-07-26 动作 A 跑出来的）

- ❌ 散落在 /root 根目录写 .go → 误判"项目没源码"
- ❌ 浅克隆 (shallow clone) → push 缺父对象，被 GitHub 拒绝
- ❌ 编译产物 new-api-custom (113MB) 被 commit → 超过 GitHub 100MB 限制
- ❌ 先 commit 才加 .gitignore → 大文件进了历史，只能 filter-repo 重写

详见 `experience.db` 崩点 #1-#4。

## 下一步

- [动作 B] 补 PRD + 跑质量门禁 + 收 evidence
- [动作 C] 客户验收 + 复盘入库
