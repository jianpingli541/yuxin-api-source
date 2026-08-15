package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestEncryptDecryptOptionValueRoundTrip(t *testing.T) {
	common.CryptoSecret = "unit-test-secret"
	plain := "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSj\n-----END PRIVATE KEY-----"

	enc := EncryptSensitiveOptionValue("AlipayPrivateKey", plain)
	if !common.IsChannelKeyEncrypted(enc) {
		t.Fatal("expected enc1: prefixed output")
	}
	if enc == plain {
		t.Fatal("ciphertext equals plaintext")
	}
	if got := DecryptOptionValue(enc); got != plain {
		t.Fatal("round trip mismatch")
	}
}

func TestEncryptSensitiveOptionValueSkipsNonSensitive(t *testing.T) {
	common.CryptoSecret = "unit-test-secret"
	if got := EncryptSensitiveOptionValue("AlipayAppId", "2021001100000000"); got != "2021001100000000" {
		t.Fatal("non-sensitive key must stay plaintext")
	}
	if got := EncryptSensitiveOptionValue("AlipayPrivateKey", ""); got != "" {
		t.Fatal("empty value must stay empty")
	}
	enc := EncryptSensitiveOptionValue("WechatPrivateKey", "pk-data")
	if got := EncryptSensitiveOptionValue("WechatPrivateKey", enc); got != enc {
		t.Fatal("encryption must be idempotent for enc1: input")
	}
}

func TestDecryptOptionValuePlaintextPassthrough(t *testing.T) {
	if got := DecryptOptionValue("plain-value"); got != "plain-value" {
		t.Fatal("plaintext must pass through unchanged")
	}
	if got := DecryptOptionValue(""); got != "" {
		t.Fatal("empty must pass through unchanged")
	}
}
