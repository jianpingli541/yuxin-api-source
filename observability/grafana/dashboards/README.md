# Grafana 运营看板接入说明

面向豫鑫 API 网关的运营可观测性看板。数据源 = 已有的 Prometheus（总量指标）+ ClickHouse `new_api_logs.logs`（调用明细）。

## 文件清单

```
observability/grafana/
├── dashboards/
│   ├── dashboard-api-gateway.json      # 原有基础监控（Prom 指标）
│   └── dashboard-operations.json       # 新增: 运营看板（成本/延迟/错误率/渠道）
└── provisioning/
    ├── dashboards/dashboard-provider.yml
    └── datasources/
        ├── datasource.yml              # 原有: Prometheus (uid=yuxin-prometheus)
        └── clickhouse.yml              # 新增: ClickHouse 日志库 (uid=yuxin-clickhouse)
```

## 覆盖的四个面板域

| 域 | 面板 | 数据源 | SQL/指标 |
|---|---|---|---|
| 每日成本 | 每日成本(按模型)趋势、成本 TOP 模型 | ClickHouse | `type=2`(LogTypeConsume), `sum(quota)*0.002` |
| 延迟 P95/P99 | 延迟分位(按模型)、每日 P95 趋势 | ClickHouse | `quantile(0.5/0.95/0.99)(use_time)` |
| 渠道错误率矩阵 | 渠道×成功率/错误率、每日错误数 | ClickHouse | `type=2` 成功 / `type=5` 错误 |
| 渠道调用量 | 渠道调用量&成本(按 channel_id) | ClickHouse | `count() GROUP BY channel_id` |
| 总量补充 | 总成本/成功率/缓存命中率/请求速率 | Prometheus | `yuxin_api_*` |

## 必须先装 ClickHouse 数据源插件

运营看板的 ClickHouse 面板依赖插件 **`grafana-clickhouse-datasource`**（Grafana 官方 ClickHouse 插件）。
**当前容器未安装**，不装则这些面板报 "Data source not found / plugin not installed"。

在 `docker-compose.observability.yml` 的 `grafana.environment` 下加一行：

```yaml
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=${GF_SECURITY_ADMIN_PASSWORD}
      - GF_USERS_ALLOW_SIGN_UP=false
      - GF_AUTH_ANONYMOUS_ENABLED=false
      - GF_INSTALL_PLUGINS=grafana-clickhouse-datasource   # 新增
```

或进入容器手动装（不回写 compose，不持久）：

```bash
docker exec gateway-grafana grafana-cli plugins install grafana-clickhouse-datasource
docker restart gateway-grafana
```

## 挂载与生效（由 PM 统一执行重启）

1. 确认本目录文件已就位（compose 已挂载）：
   - `provisioning/` → 容器 `/etc/grafana/provisioning`（数据源+看板目录自动加载）
   - `dashboards/` → 容器 `/var/lib/grafana/dashboards`（provider 每 30s 扫描）
2. 改 `docker-compose.observability.yml`（加 `GF_INSTALL_PLUGINS`，见上）。
3. 由 PM 执行（**本次未重启**，以下为接入命令）：

```bash
cd /root/projects/api-gateway
docker compose -f docker-compose.observability.yml up -d grafana
# 首次装插件会联网下载，稍等插件初始化
```

4. 验证：
   - 打开 `http://127.0.0.1:3001`（admin / `${GF_SECURITY_ADMIN_PASSWORD}`）
   - Administration → Data sources：应出现 `Prometheus`（默认）与 `ClickHouse-Logs`
   - Dashboards：`豫鑫 API 网关 · 运营看板`（uid `yuxin-api-operations`）

## 数据源连接说明

- **ClickHouse-Logs**：`host=gateway-clickhouse, port=8123`（HTTP 接口，与 grafana 同 `api-gateway_gateway-net` 网络），
  `defaultDatabase=new_api_logs`，用户 `default`。已实测 grafana→`http://gateway-clickhouse:8123/ping` 返回 `Ok.`。
- **Prometheus**：原有，URL `http://prometheus:9090`，`isDefault=true`。

## 语义约定（写查询时已核对源码）

- `type=2` = LogTypeConsume（真实消费/成功调用，含 model_name/channel_id/tokens/quota/use_time）
- `type=5` = LogTypeError（错误调用）
- `type=3` = 管理操作、`type=7` = 登录（看板不计入）
- `quota` 是内部账单单位：`QuotaPerUnit = 500*1000`（common/constants.go），**1 quota = $0.002 / 1K tokens**，故 USD 换算 `sum(quota)*0.002`
- `use_time` 为**整数秒**（粒度较粗；要更细延迟请在网关侧补 histogram 指标）
- `channel_id` 是**数字**，日志表无渠道名字段；如需显示渠道名，建议在 ClickHouse 端建 channel 维表后 JOIN（本看板先按 channel_id 出，够用）

## 数据现状（勘察基线）

- ClickHouse `new_api_logs.logs` 当前 86 行：`type=2` 仅 1 条（模型测试 deepseek-v4-pro）、`type=3` 63 条、`type=7` 22 条、**`type=5` 0 条**（故错误率面板暂为空属正常）
- Prometheus `yuxin_api_*` 指标**无** model/channel/latency 标签，仅 `type` 维度 → 明细只能走 ClickHouse

## 已做验证（勘察 + SQL 实测）

- Dashboard JSON 通过 `python3 json.load` 校验（11 面板）
- 全部 7 条 ClickHouse SQL 在 `gateway-clickhouse` 容器内实测通过（`$__timeFilter` 宏用具体时间替身测试），样例：
  `SELECT toDate(toDateTime(created_at)) AS day, model_name, sum(quota)*0.002 AS cost_usd FROM new_api_logs.logs WHERE type=2 ... GROUP BY day, model_name` → `2026-08-02  deepseek-v4-pro  0.064`

## 待办

- [ ] PM 统一重启 grafana 使插件 + provisioning 生效
- [ ] 装插件后人工核对 ClickHouse 面板有数据
- [ ] （可选）建 channel 维表实现渠道名展示
- [ ] （可选）网关侧补 http_request_duration histogram，替代整数秒 use_time 的延迟粒度
