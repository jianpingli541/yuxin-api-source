# 第三方依赖与许可证清单

> 生成日期: 2026-08-03
> 后端扫描: `go list -deps` 实际构建依赖(129 个模块)
> 前端扫描: `npx license-checker --production`(778 个 npm 包)

## 一、许可证汇总

### 后端(129 个构建依赖)

| 许可证 | 数量 | 备注 |
|---|---|---|
| MIT | 67 | 主流宽松许可,商用无问题 |
| Apache-2.0 | 29 | 宽松许可,含专利授权 |
| 自定义 | 27 | 各上游项目自定义条款(多为 BSD/MIT 变体),逐一核对见 CSV |
| MPL-2.0 | 1 | `github.com/hashicorp/go-retryablehttp`(文件级 copyleft) |
| Unlicense | 1 | `github.com/mewkiz/flac`(公共领域) |
| UNKNOWN-no-license-file | 1 | `github.com/Calcium-Ion/go-epay`(需上游确认,见风险) |
| 项目内部子模块 | 1 | `github.com/QuantumNous/new-api/relaykit`(AGPL-3.0,随主项目) |

### 前端(778 个生产依赖)

| 许可证 | 数量 | 备注 |
|---|---|---|
| MIT | 670 | 主流宽松 |
| ISC | 48 | 宽松 |
| BSD-3-Clause | 26 | 宽松 |
| Apache-2.0 | 11 | 宽松 + 专利 |
| BSD-2-Clause | 4 | 宽松 |
| Unlicense | 3 | 公共领域 |
| MPL-2.0 | 2 | 文件级 copyleft |
| 0BSD | 2 | 宽松 |
| OFL-1.1 | 2 | 字体许可 |
| (BSD-3 AND Apache-2.0) | 1 | 双许可 |
| (MPL-2.0 OR Apache-2.0) | 1 | 双许可 |
| (AFL-2.1 OR BSD-3-Clause) | 1 | 双许可 |
| Custom | 3 | giscus/shiki/spline(各自条款) |
| UNKNOWN | 1 | 项目根包自己,忽略 |

## 二、商用兼容性评估

**结论:依赖许可证总体与 AGPL-3.0 商用分发兼容,但有 2 项需法务确认。**

| 项 | 风险 | 说明 | 建议 |
|---|---|---|---|
| `github.com/Calcium-Ion/go-epay`(易支付 SDK) | 中 | 仓库无 LICENSE 文件,README 未声明许可;GitHub 页面未显示 license | 与上游确认许可证,或替换为 MIT/Apache 许可的等价库 |
| `MPL-2.0` 依赖(`hashicorp/go-retryablehttp`、前端 `@tanstack/*` 部分) | 低 | MPL-2.0 仅约束被修改文件;若未修改其源文件,无传染风险 | 保持不修改 MPL 文件即可 |
| 其余(MIT/Apache/BSD/ISC/Unlicense/0BSD) | 无 | 标准宽松许可 | 无 |

## 三、AGPL-3.0 与依赖的关系

- **本项目基线** `QuantumNous/new-api`(上游)为 **AGPL-3.0**。
- AGPL 只约束"修改本项目代码"的部分,不约束"依赖本项目代码的第三方库"。
- 上述所有依赖均为独立许可证,与 AGPL 无冲突。
- **但** AGPL 的网络分发条款(SaaS 即分发)要求:如果以网络服务方式运营本项目,**必须向所有用户提供修改后的完整源代码**。这是 AGPL 对商用分发的硬性约束(详见 PRD §7)。

## 四、依赖安全状态(2026-08-03)

| 项 | 结果 |
|---|---|
| govulncheck | 已升级到 Go 1.25.12 + x/text v0.39 + x/image v0.43,**代码调用层面 0 个漏洞** |
| npm audit(生产依赖) | low:2 / moderate:2 / high:4 / critical:0(详见 R6 整改) |

## 五、文件位置

- 后端清单: `docs/DEPENDENCIES-backend.csv`
- 前端清单: `docs/DEPENDENCIES-frontend.csv`
- 本文档: `docs/DEPENDENCIES.md`
