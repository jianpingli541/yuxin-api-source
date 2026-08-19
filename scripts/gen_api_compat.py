#!/usr/bin/env python3
"""生成 docs/api-compat.md 的管理 API 路由表骨架。

用法：
    python3 scripts/gen_api_compat.py [router_file]

输出：Markdown 表格到 stdout（列：端点 / HTTP 方法 / 鉴权要求 / 速率限制 / 业务类别）。

说明：
- 仅基于路由注册行，组级 .Use() 中间件按定义顺序继承；注册行外的动态挂载不追踪。
- 业务类别：admin=管理员、user=登录用户、public=无鉴权（待核）、system=系统/监控。
"""
import re
import sys

SRC = sys.argv[1] if len(sys.argv) > 1 else "router/api-router.go"

ROUTE_RE = re.compile(
    r'\b([A-Za-z_]\w*)\.(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS|Any)\(\s*"([^"]*)"'
)
GROUP_RE = re.compile(r'\b(\w+)\s*:?=\s*([A-Za-z_]\w*)\.Group\(\s*"([^"]*)"')
USE_RE = re.compile(r'\b([A-Za-z_]\w*)\.Use\(')
MW_RE = re.compile(r'middleware\.(\w+)')


def main():
    prefixes = {"router": ""}
    mws = {"router": set()}
    rows = []
    with open(SRC, encoding="utf-8") as f:
        for line in f:
            g = GROUP_RE.search(line)
            if g:
                var, parent, seg = g.groups()
                prefixes[var] = (prefixes.get(parent, "") + seg).rstrip("/")
                mws[var] = set(mws.get(parent, set())) | set(MW_RE.findall(line))
                continue
            u = USE_RE.search(line)
            if u:
                var = u.group(1)
                mws.setdefault(var, set())
                mws[var] |= set(MW_RE.findall(line))
                continue
            r = ROUTE_RE.search(line)
            if r:
                var, method, path = r.groups()
                full = prefixes.get(var, "") + path
                if not full.startswith("/"):
                    full = "/" + full
                eff = mws.get(var, set()) | set(MW_RE.findall(line))
                rows.append((full, method, eff))

    out = [
        "| 端点 | HTTP 方法 | 鉴权要求 | 速率限制 | 业务类别 |",
        "|---|---|---|---|---|",
    ]
    stat = {"admin": 0, "user": 0, "public": 0, "system": 0}
    for full, method, eff in rows:
        auth_mw = [m for m in sorted(eff) if m.endswith("Auth")]
        rl_mw = [m for m in sorted(eff) if "RateLimit" in m]
        if {"AdminAuth", "RootAuth"} & eff:
            cat = "admin"
        elif "UserAuth" in eff:
            cat = "user"
        elif "metrics" in full.lower():
            cat = "system"
        else:
            cat = "public"
        stat[cat] += 1
        auth = "、".join(auth_mw) if auth_mw else "无鉴权（待核）"
        rl = "、".join(rl_mw) if rl_mw else "—"
        out.append(f"| `{full}` | {method} | {auth} | {rl} | {cat} |")
    out.append("")
    out.append(
        f"> 合计 **{len(rows)}** 条路由：admin {stat['admin']}、user {stat['user']}、"
        f"public {stat['public']}（含待核）、system {stat['system']}。"
    )
    print("\n".join(out))


if __name__ == "__main__":
    main()
