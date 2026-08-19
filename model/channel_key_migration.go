package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

// MigrateChannelKeysToEncrypted 一次性迁移:将数据库中明文存储的
// channels.key 用 AES-256-GCM 加密(CRYPTO_SECRET 派生)。
//
// 幂等:已带 enc1: 前缀的记录会被跳过,重复执行安全。
// 迁移失败会返回错误并阻止启动,避免半加密状态下密钥泄露面扩大。
func MigrateChannelKeysToEncrypted() error {
	if DB == nil {
		return nil
	}
	var rows []struct {
		Id  int
		Key string
	}
	// 只取未加密的 key;前缀判断在 SQL 层完成,避免全表加载到内存
	prefix := common.ChannelKeyCipherVersion + ":%"
	err := DB.Raw("SELECT id, key FROM channels WHERE key IS NOT NULL AND key != '' AND key NOT LIKE ?", prefix).
		Scan(&rows).Error
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	common.SysLog(fmt.Sprintf("encrypting plaintext channel keys (R1 migration): %d rows", len(rows)))
	encryptedCount := 0
	for _, row := range rows {
		enc, err := common.ChannelKeyEncrypt(row.Key)
		if err != nil {
			return err
		}
		if err := DB.Exec("UPDATE channels SET key = ? WHERE id = ?", enc, row.Id).Error; err != nil {
			return err
		}
		encryptedCount++
	}
	common.SysLog(fmt.Sprintf("channel key encryption migration done: %d rows encrypted", encryptedCount))
	return nil
}
