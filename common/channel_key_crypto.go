package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

// ChannelKeyCipherVersion 当前加密格式版本。
// 格式:enc1:<base64(nonce)>:<base64(ciphertext+tag)>
//   - nonce: 12 字节 AES-GCM 标准 nonce
//   - ciphertext: AES-GCM 输出(含 tag)
//   - key 派生: SHA-256(CRYPTO_SECRET) -> 32 字节
const ChannelKeyCipherVersion = "enc1"

// ChannelKeyEncrypt 使用 AES-256-GCM 加密上游渠道密钥。
// 返回格式:enc1:<base64(nonce)>:<base64(ciphertext+tag)>
// 空串或已加密输入原样返回(幂等)。
func ChannelKeyEncrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	// 已加密的输入直接返回,避免重复加密
	if IsChannelKeyEncrypted(plaintext) {
		return plaintext, nil
	}
	key, err := deriveChannelKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	enc := ChannelKeyCipherVersion + ":" +
		base64.StdEncoding.EncodeToString(nonce) + ":" +
		base64.StdEncoding.EncodeToString(ciphertext)
	return enc, nil
}

// ChannelKeyDecrypt 解密上游渠道密钥。
// 明文输入原样返回(向后兼容),非法格式返回错误。
func ChannelKeyDecrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !IsChannelKeyEncrypted(ciphertext) {
		return ciphertext, nil // 已是明文
	}
	parts := strings.SplitN(ciphertext, ":", 3)
	if len(parts) != 3 || parts[0] != ChannelKeyCipherVersion {
		return "", errors.New("invalid channel key cipher format")
	}
	nonce, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	ciphertextBlob, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", err
	}
	key, err := deriveChannelKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertextBlob, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// IsChannelKeyEncrypted 判断字符串是否已是加密格式。
func IsChannelKeyEncrypted(s string) bool {
	return strings.HasPrefix(s, ChannelKeyCipherVersion+":")
}

func deriveChannelKey() ([]byte, error) {
	if CryptoSecret == "" {
		return nil, errors.New("CRYPTO_SECRET not configured")
	}
	sum := sha256.Sum256([]byte(CryptoSecret))
	return sum[:], nil
}
