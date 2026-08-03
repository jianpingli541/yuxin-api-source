# Lint 使用说明（golangci-lint）

配置文件：仓库根目录 `.golangci.yml`（golangci-lint **v2** 配置格式）。

## 安装

feifei 已安装 v2.12.2 于 `/usr/local/bin/golangci-lint`。其他机器安装：

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
```

注意：二进制安装脚本（raw.githubusercontent.com install.sh）在本机曾出现
GitHub release CDN 返回内容与官方 checksum 不符的情况，改用 `go install`
（有 sumdb 校验）更可靠。golangci-lint 是开发工具，不写入 go.mod。

## 运行

```bash
cd /root/projects/api-gateway
golangci-lint run --timeout 5m .
```

基线目标：**exit 0**（2026-08-03 基线已跑通）。warning 可列出但不 fail，
后续迭代逐步收紧。

## 启用规则（保守集）

| 规则 | 用途 |
|---|---|
| errcheck | 未检查的 error 返回值 |
| govet | Go 官方静态分析 |
| staticcheck | 综合静态分析 |
| unused | 未使用的符号 |
| ineffassign | 无效赋值 |
| misspell | 注释/字符串英文拼写 |
| gofmt（formatter） | 格式化检查 |

## 不启用 / 放宽

- **stylecheck**：风格类警告过多，本期不启用。
- **exhaustive**：业务枚举很多，全量要求不现实，本期不启用。
- **gocognit**（认知复杂度阈值放宽至 30）、**cyclop**（圈复杂度放宽至 25）：
  本期不启用，阈值已在 `.golangci.yml` 预留，收紧迭代时直接启用即可。

## 排除范围

- `web/`（前端资源）、`relaykit/` 目录
- `*_test.go` 测试文件
- `*.bak` / `*.bak.*` 备份文件

## 收紧路线（后续迭代建议）

1. 打开 gocognit(30) / cyclop(25)，处理超阈值函数清单；
2. 为 errcheck 逐包去掉豁免；
3. 视情况引入 stylecheck / revive 子集。
