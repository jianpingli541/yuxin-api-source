# payment-setup 凭据注入工具

用于在 **无法通过浏览器后台**（root 账号 2FA 加持、或运维不便登入）时，
把微信支付 / 支付宝凭据写入 new-api 数据库。走 `model.UpdateOption`，
与 GUI 保存链路完全等价；GUI 面板后续仍可正常调整任一字段。

> 凭据边界：本工具不内置任何默认凭据，AI 助手不接触凭据内容。
> 凭据由运维人员自行填写 `payment.json` 并 scp 到服务器执行。

## 字段对照（payment.json 模板）

```json
{
  "wechat": {
    "WechatEnabled": true,
    "WechatMerchantId": "微信支付商户号",
    "WechatAppId": "公众号/小程序/移动应用 AppId",
    "WechatApiV3Key": "APIv3 密钥（微信商户平台设置）",
    "WechatPrivateKey": "<PEM 私钥全文，含 BEGIN/END 行>",
    "WechatCertSerialNo": "商户证书序列号",
    "WechatNotifyUrl": "",
    "WechatReturnUrl": "",
    "WechatUnitPrice": 1,
    "WechatMinTopUp": 1
  },
  "alipay": {
    "AlipayEnabled": true,
    "AlipayAppId": "支付宝应用 AppId",
    "AlipayPrivateKey": "<PEM 私钥全文>",
    "AlipayPublicKey": "<支付宝公钥 PEM>",
    "AlipaySandbox": false,
    "AlipayNotifyUrl": "",
    "AlipayReturnUrl": "",
    "AlipayUnitPrice": 1,
    "AlipayMinTopUp": 1
  }
}
```

字段说明：

| 字段 | 来源 | 备注 |
|---|---|---|
| `WechatMerchantId` | 微信商户平台「账户中心」 | 10 位数字 |
| `WechatAppId` | 公众号/小程序管理后台 | 与商户号绑定 |
| `WechatApiV3Key` | 商户平台「API 安全」→ APIv3 密钥 | 32 位字符串 |
| `WechatPrivateKey` | 商户平台下载 API 证书 | PEM 格式；如压成单行可用字面 `\n` 分隔，工具自动还原 |
| `WechatCertSerialNo` | 商户平台「API 安全」→ 证书管理 | 工具用它选商户证书 |
| `AlipayAppId` | 开放平台应用详情 | |
| `AlipayPrivateKey` | 开放平台「应用 → 接口加签方式」 | PKCS8 PEM |
| `AlipayPublicKey` | 开放平台「应用 → 接口加签方式 → 查看支付宝公钥」 | 工具 `LoadAliPayPublicKey` 用它验签 |
| `WechatNotifyUrl`/`AlipayNotifyUrl` | 可留空 | 留空时用 `ServerAddress+/api/{wechat,alipay}/notify` |

## 商户后台回调地址配置

若 `NotifyUrl` 留空，默认回调走 `https://ai.yuxin.yun/api/wechat/notify`
与 `/api/alipay/notify`。务必在对应商户后台填同样的地址：

- **微信**：商户平台「产品中心 → 开发配置 → 支付结果通知」
  填 `https://ai.yuxin.yun/api/wechat/notify`
- **支付宝**：开放平台「应用 → 接口加签方式 / 异步通知地址」
  填 `https://ai.yuxin.yun/api/alipay/notify`

`ReturnUrl` 是用户支付完成后跳转的前端页面，可填 `https://ai.yuxin.yun/wallet`
或留空（不跳转）。

## 使用步骤

1. 填写本机 `payment.json`（参考上方模板，填入真实凭据）
2. 传到服务器：

   ```bash
   scp payment.json feifei:/tmp/payment.json
   ```

3. 在服务器上以容器环境执行（共享 `SQL_DSN`）：

   ```bash
   docker cp /tmp/payment.json gateway-new-api:/tmp/payment.json
   docker exec gateway-new-api /payment-setup --write /tmp/payment.json
   ```

4. 看到 `微信 client 构造成功` 和 `支付宝 client 构造成功` 即生效；
   回调 URL 可达性会同时打印。

5. 立刻删除凭据文件：

   ```bash
   rm -f /tmp/payment.json
   docker exec gateway-new-api rm -f /tmp/payment.json
   ```

6. 网关进程会在 options 同步周期内自动 reload；如需立刻生效：

   ```bash
   docker restart gateway-new-api
   ```

## 子命令

- `--check`：只读自检，打印当前 DB 凭据掩码 + SDK 客户端构造状态
- `--write <path>`：从 JSON 写入 19 个字段，写完自动 verify
- `--verify`：客户端构造 + 回调 URL HEAD 可达性探测
- `--remove`：清空全部 Wechat/Alipay options（应急回滚）

## 安全提示

- 本工具只做凭据落库；不做支付下单测试
- 凭据文件用后即删，勿入 git / 备份 / 聊天记录
- DB `options` 表的支付字段一旦写入，任何能登入后台的账号都能看到
  （建议保持 root 账号 2FA 开启，限制后台访问源 IP）
