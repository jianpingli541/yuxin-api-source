package common

import (
	"strings"
	"testing"
)

func TestChannelKeyEncryptDecrypt_RoundTrip(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = "test-secret-for-channel-key-encryption"

	cases := []string{
		"sk-abc123",
		"sk-b9b09e225da0427aa5f0e1c5c3c2d4a6",
		"sk-abc\nsk-def\nsk-ghi",
		strings.Repeat("a", 1000),
		"key with unicode 中文密钥 🔑",
	}
	for _, plain := range cases {
		enc, err := ChannelKeyEncrypt(plain)
		if err != nil {
			t.Fatalf("encrypt %q: %v", plain[:min(20, len(plain))], err)
		}
		if !IsChannelKeyEncrypted(enc) {
			t.Fatalf("ciphertext missing prefix: %s", enc[:30])
		}
		if strings.Contains(enc, plain[:min(8, len(plain))]) {
			t.Fatal("ciphertext leaks plaintext prefix")
		}
		got, err := ChannelKeyDecrypt(enc)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if got != plain {
			t.Fatalf("round-trip mismatch: got %q want %q", got[:min(20, len(got))], plain[:min(20, len(plain))])
		}
	}
}

func TestChannelKeyEncrypt_Idempotent(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = "test-secret"

	enc1, err := ChannelKeyEncrypt("sk-abc")
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := ChannelKeyEncrypt(enc1)
	if err != nil {
		t.Fatal(err)
	}
	if enc1 != enc2 {
		t.Fatal("double encrypt should be idempotent")
	}
}

func TestChannelKeyEncrypt_EmptyInput(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = "test-secret"
	enc, err := ChannelKeyEncrypt("")
	if err != nil || enc != "" {
		t.Fatalf("empty input should return empty, got %q err=%v", enc, err)
	}
}

func TestChannelKeyEncrypt_UniqueNonce(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = "test-secret"
	enc1, _ := ChannelKeyEncrypt("sk-abc")
	enc2, _ := ChannelKeyEncrypt("sk-abc")
	if enc1 == enc2 {
		t.Fatal("same plaintext should produce different ciphertexts (random nonce)")
	}
	// both decrypt to same value
	d1, _ := ChannelKeyDecrypt(enc1)
	d2, _ := ChannelKeyDecrypt(enc2)
	if d1 != d2 || d1 != "sk-abc" {
		t.Fatalf("decrypt mismatch: %q %q", d1, d2)
	}
}

func TestChannelKeyDecrypt_PlainPassthrough(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = "test-secret"
	got, err := ChannelKeyDecrypt("sk-plaintext-legacy")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-plaintext-legacy" {
		t.Fatalf("plaintext should pass through, got %q", got)
	}
}

func TestChannelKeyDecrypt_Tampered(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = "test-secret"
	enc, _ := ChannelKeyEncrypt("sk-abc")
	// tamper with the ciphertext part
	parts := strings.Split(enc, ":")
	b := []byte(parts[2])
	b[len(b)-2] ^= 0xff
	tampered := parts[0] + ":" + parts[1] + ":" + string(b)
	if _, err := ChannelKeyDecrypt(tampered); err == nil {
		t.Fatal("tampered ciphertext should fail to decrypt")
	}
}

func TestChannelKeyDecrypt_WrongSecret(t *testing.T) {
	CryptoSecret = "secret-one"
	enc, err := ChannelKeyEncrypt("sk-abc")
	if err != nil {
		t.Fatal(err)
	}
	CryptoSecret = "secret-two"
	if _, err := ChannelKeyDecrypt(enc); err == nil {
		t.Fatal("wrong secret should fail to decrypt")
	}
}

func TestChannelKeyDecrypt_Malformed(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = "test-secret"
	cases := []string{
		"enc1:",
		"enc1:only-one-part",
		"enc1:!!!notbase64:abc",
		"enc1:abc:!!!",
	}
	for _, c := range cases {
		if _, err := ChannelKeyDecrypt(c); err == nil {
			t.Errorf("malformed %q should error", c)
		}
	}
}

func TestChannelKeyEncrypt_NoSecret(t *testing.T) {
	prev := CryptoSecret
	defer func() { CryptoSecret = prev }()
	CryptoSecret = ""
	if _, err := ChannelKeyEncrypt("sk-abc"); err == nil {
		t.Fatal("missing CRYPTO_SECRET should error")
	}
}

func TestIsChannelKeyEncrypted(t *testing.T) {
	if !IsChannelKeyEncrypted("enc1:abc:def") {
		t.Fatal("should detect enc1 prefix")
	}
	if IsChannelKeyEncrypted("sk-abc") {
		t.Fatal("plain sk- should not match")
	}
	if IsChannelKeyEncrypted("enc2:abc:def") {
		t.Fatal("future version should not match")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}