package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestGenerateHMACWithKey_Deterministic(t *testing.T) {
	key := []byte("test-key-12345")
	got1 := GenerateHMACWithKey(key, "hello")
	got2 := GenerateHMACWithKey(key, "hello")
	if got1 != got2 {
		t.Fatalf("HMAC not deterministic: %s vs %s", got1, got2)
	}
	// expected: hex-encoded HMAC-SHA256
	h := hmac.New(sha256.New, key)
	h.Write([]byte("hello"))
	expected := hex.EncodeToString(h.Sum(nil))
	if got1 != expected {
		t.Fatalf("HMAC mismatch: got %s want %s", got1, expected)
	}
	if len(got1) != 64 {
		t.Fatalf("HMAC hex length: got %d want 64", len(got1))
	}
}

func TestGenerateHMACWithKey_DifferentKeysDifferentMac(t *testing.T) {
	m1 := GenerateHMACWithKey([]byte("key-a"), "data")
	m2 := GenerateHMACWithKey([]byte("key-b"), "data")
	if m1 == m2 {
		t.Fatalf("HMAC should differ with different keys")
	}
}

func TestGenerateHMAC_UsesGlobalSecret(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = "global-secret"
	got := GenerateHMAC("hello")
	expected := GenerateHMACWithKey([]byte("global-secret"), "hello")
	if got != expected {
		t.Fatalf("GenerateHMAC should use CryptoSecret")
	}
}

func TestPassword2Hash_ValidBcrypt(t *testing.T) {
	hash, err := Password2Hash("MySecureP@ss123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Fatalf("not a bcrypt hash: %s", hash[:5])
	}
	if !ValidatePasswordAndHash("MySecureP@ss123", hash) {
		t.Fatal("hash does not validate")
	}
	if ValidatePasswordAndHash("wrong-pass", hash) {
		t.Fatal("wrong password validated")
	}
}

func TestPassword2Hash_DifferentHashesForSamePassword(t *testing.T) {
	h1, _ := Password2Hash("abc")
	h2, _ := Password2Hash("abc")
	if h1 == h2 {
		t.Fatal("bcrypt salt should make each hash unique")
	}
	if !ValidatePasswordAndHash("abc", h1) || !ValidatePasswordAndHash("abc", h2) {
		t.Fatal("both salts should validate")
	}
}

func TestValidatePasswordAndHash_EmptyHash(t *testing.T) {
	if ValidatePasswordAndHash("anything", "") {
		t.Fatal("empty hash should not validate")
	}
}

func TestValidatePasswordAndHash_EmptyPassword(t *testing.T) {
	hash, _ := Password2Hash("real")
	if ValidatePasswordAndHash("", hash) {
		t.Fatal("empty password should not validate against non-empty hash")
	}
}