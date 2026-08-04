package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建一个内存 SQLite 数据库用于测试 GORM hooks。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&Channel{}, &Ability{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	prev := DB
	DB = db
	t.Cleanup(func() { DB = prev })
	return db
}

func withCryptoSecret(t *testing.T, secret string) {
	t.Helper()
	prev := common.CryptoSecret
	common.CryptoSecret = secret
	t.Cleanup(func() { common.CryptoSecret = prev })
}

func TestChannelHook_InsertEncryptsKey(t *testing.T) {
	setupTestDB(t)
	withCryptoSecret(t, "test-secret-r1")

	ch := &Channel{
		Id:   1,
		Name: "test",
		Key:  "sk-plaintext-abc123",
		Type: 1,
	}
	if err := ch.Insert(); err != nil {
		t.Fatal(err)
	}

	// 验证 DB 里是密文(不走 AfterFind,直接 raw 查)
	var dbKey string
	DB.Raw("SELECT key FROM channels WHERE id = 1").Scan(&dbKey)
	if dbKey == "sk-plaintext-abc123" {
		t.Fatal("key stored as plaintext")
	}
	if !common.IsChannelKeyEncrypted(dbKey) {
		t.Fatalf("key not encrypted format: %s", dbKey[:minInt(30, len(dbKey))])
	}

	// 通过 GORM 读取应自动解密
	got, err := GetChannelById(1, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "sk-plaintext-abc123" {
		t.Fatalf("AfterFind should decrypt: got %s", got.Key)
	}
}

func TestChannelHook_UpdateEncryptsKey(t *testing.T) {
	setupTestDB(t)
	withCryptoSecret(t, "test-secret-r1")

	ch := &Channel{Id: 1, Name: "t", Key: "sk-original", Type: 1}
	ch.Insert()

	ch.Key = "sk-rotated"
	if err := ch.Update(); err != nil {
		t.Fatal(err)
	}

	// DB 里密文应改变
	var dbKey string
	DB.Raw("SELECT key FROM channels WHERE id = 1").Scan(&dbKey)
	if !common.IsChannelKeyEncrypted(dbKey) {
		t.Fatal("updated key not encrypted")
	}

	got, _ := GetChannelById(1, true)
	if got.Key != "sk-rotated" {
		t.Fatalf("decrypted key mismatch: %s", got.Key)
	}
}

func TestChannelHook_EmptyKeyNotEncrypted(t *testing.T) {
	setupTestDB(t)
	withCryptoSecret(t, "test-secret-r1")

	ch := &Channel{Id: 1, Name: "t", Key: "", Type: 1}
	ch.Insert()

	var dbKey string
	DB.Raw("SELECT key FROM channels WHERE id = 1").Scan(&dbKey)
	if dbKey != "" {
		t.Fatalf("empty key should stay empty, got %s", dbKey)
	}
}

func TestChannelHook_AlreadyEncryptedNotDoubleEncrypted(t *testing.T) {
	setupTestDB(t)
	withCryptoSecret(t, "test-secret-r1")

	enc, _ := common.ChannelKeyEncrypt("sk-already")
	ch := &Channel{Id: 1, Name: "t", Key: enc, Type: 1}
	ch.Insert()

	var dbKey string
	DB.Raw("SELECT key FROM channels WHERE id = 1").Scan(&dbKey)
	if dbKey != enc {
		t.Fatal("already-encrypted key should not be re-encrypted")
	}
}

func TestMigrateChannelKeys_PlaintextGetsEncrypted(t *testing.T) {
	setupTestDB(t)
	withCryptoSecret(t, "test-secret-r1")

	// 直接 raw insert 3 条明文(绕过 hook)
	DB.Exec("INSERT INTO channels (id, name, key, type) VALUES (1, 'a', 'sk-plain-1', 1)")
	DB.Exec("INSERT INTO channels (id, name, key, type) VALUES (2, 'b', 'sk-plain-2', 1)")
	DB.Exec("INSERT INTO channels (id, name, key, type) VALUES (3, 'c', '', 1)")

	if err := MigrateChannelKeysToEncrypted(); err != nil {
		t.Fatal(err)
	}

	var rows []struct {
		Id  int
		Key string
	}
	DB.Raw("SELECT id, key FROM channels ORDER BY id").Scan(&rows)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if !common.IsChannelKeyEncrypted(rows[0].Key) {
		t.Fatalf("row 1 not encrypted: %s", rows[0].Key)
	}
	if !common.IsChannelKeyEncrypted(rows[1].Key) {
		t.Fatalf("row 2 not encrypted: %s", rows[1].Key)
	}
	if rows[2].Key != "" {
		t.Fatalf("empty key should stay empty: %s", rows[2].Key)
	}

	// 解密验证
	d1, _ := common.ChannelKeyDecrypt(rows[0].Key)
	if d1 != "sk-plain-1" {
		t.Fatalf("decrypt mismatch: %s", d1)
	}
}

func TestMigrateChannelKeys_Idempotent(t *testing.T) {
	setupTestDB(t)
	withCryptoSecret(t, "test-secret-r1")

	enc, _ := common.ChannelKeyEncrypt("sk-already")
	DB.Exec("INSERT INTO channels (id, name, key, type) VALUES (1, 'a', ?, 1)", enc)
	DB.Exec("INSERT INTO channels (id, name, key, type) VALUES (2, 'b', 'sk-plain', 1)")

	// 第一次迁移只处理明文记录(id=2)
	if err := MigrateChannelKeysToEncrypted(); err != nil {
		t.Fatal(err)
	}

	// 第二次迁移应该零工作量
	if err := MigrateChannelKeysToEncrypted(); err != nil {
		t.Fatal(err)
	}

	var dbKey1, dbKey2 string
	DB.Raw("SELECT key FROM channels WHERE id = 1").Scan(&dbKey1)
	DB.Raw("SELECT key FROM channels WHERE id = 2").Scan(&dbKey2)
	if dbKey1 != enc {
		t.Fatal("id=1 key changed during idempotent migration")
	}
	d2, _ := common.ChannelKeyDecrypt(dbKey2)
	if d2 != "sk-plain" {
		t.Fatalf("id=2 decrypt mismatch: %s", d2)
	}
}

func TestMigrateChannelKeys_NilDB(t *testing.T) {
	prev := DB
	DB = nil
	defer func() { DB = prev }()
	if err := MigrateChannelKeysToEncrypted(); err != nil {
		t.Fatalf("nil DB should return nil, got %v", err)
	}
}

func TestMigrateChannelKeys_NoRows(t *testing.T) {
	setupTestDB(t)
	withCryptoSecret(t, "test-secret-r1")
	if err := MigrateChannelKeysToEncrypted(); err != nil {
		t.Fatalf("empty table should return nil, got %v", err)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}