package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

// sensitiveOptionKeys 列出必须加密存储的 option 键(支付私钥/对称密钥)。
// 与 channels.key 的加密方案一致:AES-256-GCM,密钥由 CRYPTO_SECRET 派生。
var sensitiveOptionKeys = []string{
	"AlipayPrivateKey",
	"WechatPrivateKey",
	"WechatApiV3Key",
}

// IsSensitiveOptionKey 判断 option 是否属于需加密的敏感集合。
func IsSensitiveOptionKey(key string) bool {
	for _, k := range sensitiveOptionKeys {
		if k == key {
			return true
		}
	}
	return false
}

// DecryptOptionValue 解密带 enc1: 前缀的值;明文原样返回(向后兼容)。
// 解密失败时记录错误并原样返回,避免配置链路因脏数据整体不可用。
func DecryptOptionValue(value string) string {
	if !common.IsChannelKeyEncrypted(value) {
		return value
	}
	dec, err := common.ChannelKeyDecrypt(value)
	if err != nil {
		common.SysError("failed to decrypt option value: " + err.Error())
		return value
	}
	return dec
}

// EncryptSensitiveOptionValue 对敏感 option 的明文值加密;
// 非敏感、空值或已加密值原样返回(幂等)。
func EncryptSensitiveOptionValue(key, value string) string {
	if !IsSensitiveOptionKey(key) || value == "" || common.IsChannelKeyEncrypted(value) {
		return value
	}
	enc, err := common.ChannelKeyEncrypt(value)
	if err != nil {
		common.SysError("failed to encrypt option " + key + ": " + err.Error())
		return value
	}
	return enc
}

// MigrateSensitiveOptionsToEncrypted 一次性迁移:将数据库中明文存储的
// 支付私钥/对称密钥用 AES-256-GCM 加密(CRYPTO_SECRET 派生)。
//
// 幂等:已带 enc1: 前缀的记录被跳过,重复执行安全。
// 读取路径(loadOptionsFromDatabase)统一走 DecryptOptionValue,
// 写入路径(UpdateOption/UpdateOptionsBulk)统一走 EncryptSensitiveOptionValue,
// 因此迁移后管理面板行为不变(内存 OptionMap 始终为明文)。
func MigrateSensitiveOptionsToEncrypted() error {
	if DB == nil {
		return nil
	}
	var rows []struct {
		Key   string
		Value string
	}
	// 只在 SQL 层过滤未加密记录,避免全表加载
	prefix := common.ChannelKeyCipherVersion + ":%"
	err := DB.Raw("SELECT key, value FROM options WHERE value IS NOT NULL AND value != '' AND value NOT LIKE ?", prefix).
		Scan(&rows).Error
	if err != nil {
		return err
	}
	encryptedCount := 0
	for _, row := range rows {
		if !IsSensitiveOptionKey(row.Key) {
			continue
		}
		enc, err := common.ChannelKeyEncrypt(row.Value)
		if err != nil {
			return err
		}
		if err := DB.Exec("UPDATE options SET value = ? WHERE key = ?", enc, row.Key).Error; err != nil {
			return err
		}
		encryptedCount++
	}
	if encryptedCount > 0 {
		common.SysLog(fmt.Sprintf("sensitive option encryption migration done: %d rows encrypted", encryptedCount))
	}
	return nil
}
