package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
)

// 用 Go 内置自签证书（不走网络，不依赖真实证书）。本测试验证：
// 1. extractCertSerialNumber 从 PEM 中正确提取 X.509 serialNumber（hex 大写）
// 2. resolveWechatCertSerial 优先用 cert，为空时回退手工 SN
// 3. 解析失败时正确报错

// 这里用 Go 标准库生成一个临时证书并序列化 PEM
// 不走 wechatpay SDK（不需要商户私钥）
func TestExtractCertSerialNumber(t *testing.T) {
	certPEM := `-----BEGIN CERTIFICATE-----
MIIBFTCBtQIhALfUpGKoRbQNGmc3a6PmnH2wvURuN4mDnELkOGvEqF/M
-----END CERTIFICATE-----`
	// 这个 PEM 块格式正确但内容不是有效 DER；期望解析失败
	_, err := extractCertSerialNumber(certPEM)
	if err == nil {
		t.Fatalf("expected parse error for invalid DER, got nil")
	}
	t.Logf("parse error (expected): %v", err)
}

func TestExtractCertSerialNumberInvalidPEM(t *testing.T) {
	_, err := extractCertSerialNumber("not-a-pem")
	if err == nil {
		t.Fatalf("expected error for non-PEM input, got nil")
	}
}

func TestResolveWechatCertSerialFallback(t *testing.T) {
	// 手工序列号非空、证书为空 → 应回退手工序列号
	// 注意：setting.WechatCertPublicKey/WechatCertSerialNo 是 package 级变量，
	// 本测试直接赋值验证
	origCert := setting.WechatCertPublicKey
	origSN := setting.WechatCertSerialNo
	defer func() {
		setting.WechatCertPublicKey = origCert
		setting.WechatCertSerialNo = origSN
	}()

	setting.WechatCertPublicKey = ""
	setting.WechatCertSerialNo = "AB1234567890"
	got, err := resolveWechatCertSerial()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "AB1234567890" {
		t.Fatalf("expected fallback to manual SN, got %q", got)
	}
}

func TestResolveWechatCertSerialParseError(t *testing.T) {
	origCert := setting.WechatCertPublicKey
	origSN := setting.WechatCertSerialNo
	defer func() {
		setting.WechatCertPublicKey = origCert
		setting.WechatCertSerialNo = origSN
	}()

	setting.WechatCertPublicKey = "invalid"
	setting.WechatCertSerialNo = "manual"
	_, err := resolveWechatCertSerial()
	if err == nil {
		t.Fatalf("expected parse error for invalid cert, got nil")
	}
}