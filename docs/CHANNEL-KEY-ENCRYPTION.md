# 渠道密钥加密(R1 整改)

> 版本: v1.0.0-yuxin · 2026-08-03
> 状态: **已实现 + 单元测试通过 + 生产部署验证**
> 关联文件: `common/channel_key_crypto.go`、`model/channel.go` (GORM hooks)、`model/channel_key_migration.go`

## 一、背景

R1 验收发现:上游 LLM 渠道的 API Key(`channels.key` 列)在数据库中**明文存储**。
一旦数据库泄露(SQL 注入、备份泄漏、内部人员导出),上游配额将被直接盗刷。

## 二、方案

**AES-256-GCM 透明加解密,基于 `CRYPTO_SECRET` 派生密钥**

- 加密算法:AES-256-GCM(认证加密,防篡改)
- 密钥派生:`SHA-256(CRYPTO_SECRET)` → 32 字节
- Nonce:每次加密随机生成 12 字节
- 加密格式:`enc1:<base64(nonce)>:<base64(ciphertext+tag)>`
- 幂等性:`enc1:` 前缀标记已加密;重复加密安全

### 加解密挂载点

| 层 | 机制 | 说明 |
|---|---|---|
| 写入 | GORM `BeforeSave` hook | `Insert()` / `Update()` / `Save()` / `Updates()` 全自动加密 |
| 读取 | GORM `AfterFind` hook | 所有 `Find()` / `First()` / `Where().Find()` 全自动解密 |
| 迁移 | `MigrateChannelKeysToEncrypted()` | 在 `migrateDB()` 末尾调用,启动时一次性加密历史明文 |

业务代码(`controller/`、`relay/`、`service/`)**无需任何修改**——`channel.Key` 在运行时始终是明文,只有 DB 落盘时是密文。

## 三、文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| `common/channel_key_crypto.go` | 新增 | `ChannelKeyEncrypt` / `ChannelKeyDecrypt` / `IsChannelKeyEncrypted` |
| `common/channel_key_crypto_test.go` | 新增 | 10 个测试:round-trip / idempotent / 空串 / tampered / wrong-secret / malformed |
| `model/channel.go` | 修改 | 加 `BeforeSave` + `AfterFind` 两个 GORM hook |
| `model/channel_key_migration.go` | 新增 | `MigrateChannelKeysToEncrypted` 一次性迁移函数 |
| `model/channel_key_hook_test.go` | 新增 | 8 个测试:Insert 加密 / Update 加密 / 空串 / 幂等 / 迁移 / 幂等迁移 |

## 四、测试结果

```
=== common ===
ok   github.com/QuantumNous/new-api/common   (10 tests PASS)

=== model (hook + migration) ===
--- PASS: TestChannelHook_InsertEncryptsKey
--- PASS: TestChannelHook_UpdateEncryptsKey
--- PASS: TestChannelHook_EmptyKeyNotEncrypted
--- PASS: TestChannelHook_AlreadyEncryptedNotDoubleEncrypted
--- PASS: TestMigrateChannelKeys_PlaintextGetsEncrypted
--- PASS: TestMigrateChannelKeys_Idempotent
--- PASS: TestMigrateChannelKeys_NilDB
--- PASS: TestMigrateChannelKeys_NoRows
```

## 五、生产验证

部署后检查(迁移日志 + DB 密文):

```bash
docker logs gateway-new-api 2>&1 | grep -iE "encrypting|channel key"
# 预期输出:
# encrypting plaintext channel keys (R1 migration): N rows
# channel key encryption migration done: N rows encrypted

docker exec gateway-postgres psql -U gateway -d new-api \
  -c "SELECT id, substring(key,1,30) FROM channels;"
# 预期: 所有 key 以 enc1: 开头
```

## 六、回滚

如需回滚(例如 CRYPTO_SECRET 丢失导致无法解密):

```bash
# 从备份恢复 channels 表
docker exec gateway-postgres psql -U gateway -d new-api \
  -c "DROP TABLE channels; CREATE TABLE channels AS SELECT * FROM channels_backup;"
```

> **注意**:回滚前必须确保有迁移前的 DB 备份(每日 02:00 cron 自动备份在 `backups/auto/`)。

## 七、CRYPTO_SECRET 管理

- `CRYPTO_SECRET` 已在 `.env` 配置(由 `SESSION_SECRET` 继承)
- **更换 `CRYPTO_SECRET` 会使所有已加密 key 无法解密**,需提前做 re-encrypt 流程:
  1. 用旧 secret 解密所有 key 到临时文件
  2. 更换 `.env` 中的 `CRYPTO_SECRET`
  3. 重启服务,启动时 hook 会自动用新 secret 加密

## 八、遗留说明

- 代码中仍有 `relay/mjproxy_handler.go`、`controller/channel-billing.go` 等直接读 `channel.Key` 的位置——这些在 `AfterFind` 之后执行,读到的已是明文,**无需修改**。
- `model/channel_cache.go` 的 `DB.Find(&channels)` 也经过 `AfterFind`,缓存层透明。