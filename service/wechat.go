package service

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/setting"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

// wechatClientBundle 缓存微信支付 client 与回调处理器。
// 凭据变化（运营在后台改配置）时重建，避免用旧私钥签名。
type wechatClientBundle struct {
	credHash string
	client   *core.Client
	handler  *notify.Handler
	mchID    string
}

var (
	wechatBundle   *wechatClientBundle
	wechatBundleMu sync.Mutex
)

// wechatCredHash 用凭据内容派生哈希，判断是否需要重建 client
func wechatCredHash() string {
	h := sha256.Sum256([]byte(
		setting.WechatMerchantId + "|" +
			setting.WechatAppId + "|" +
			setting.WechatCertSerialNo + "|" +
			setting.WechatCertPublicKey + "|" +
			setting.WechatPrivateKey + "|" +
			setting.WechatApiV3Key,
	))
	return hex.EncodeToString(h[:])
}

// extractCertSerialNumber 从 apiclient_cert.pem 的 PEM 内容提取证书序列号。
// 序列号即 X.509 证书自身的 serialNumber 字段（微信商户平台展示的值），
// 以大写 hex 表示。注意与「证书指纹」（DER 的 SHA-1）是两个不同概念。
func extractCertSerialNumber(certPEM string) (string, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", fmt.Errorf("无效的商户证书 PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("解析商户证书失败: %v", err)
	}
	return strings.ToUpper(cert.SerialNumber.Text(16)), nil
}

// resolveWechatCertSerial 优先从 WechatCertPublicKey 解析序列号；
// 解析失败或字段为空时回退到手工填写的 WechatCertSerialNo。
func resolveWechatCertSerial() (string, error) {
	if setting.WechatCertPublicKey != "" {
		sn, err := extractCertSerialNumber(setting.WechatCertPublicKey)
		if err != nil {
			return "", fmt.Errorf("从 apiclient_cert.pem 提取证书序列号失败: %v", err)
		}
		return sn, nil
	}
	return setting.WechatCertSerialNo, nil
}

// GetWechatClient 返回微信支付 core.Client（带自动签名与平台证书定时下载）。
// 返回的 handler 用于回调通知验签 + AES-GCM 解密。
func GetWechatClient(ctx context.Context) (*core.Client, *notify.Handler, error) {
	wechatBundleMu.Lock()
	defer wechatBundleMu.Unlock()

	hash := wechatCredHash()
	if wechatBundle != nil && wechatBundle.credHash == hash {
		return wechatBundle.client, wechatBundle.handler, nil
	}

	if setting.WechatPrivateKey == "" {
		return nil, nil, fmt.Errorf("微信支付凭据未配置（缺少商户私钥 apiclient_key.pem）")
	}
	// 序列号二选一：证书本体优先，否则手工序列号
	if setting.WechatCertPublicKey == "" && setting.WechatCertSerialNo == "" {
		return nil, nil, fmt.Errorf("微信支付凭据未配置（缺少证书序列号或 apiclient_cert.pem）")
	}

	serialNo, err := resolveWechatCertSerial()
	if err != nil {
		return nil, nil, err
	}

	mchPrivateKey, err := utils.LoadPrivateKey(setting.WechatPrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("解析微信支付商户私钥失败: %v", err)
	}

	// WithWechatPayAutoAuthCipher：请求自动签名 + 注册平台证书下载器（定时刷新）
	client, err := core.NewClient(ctx,
		option.WithWechatPayAutoAuthCipher(
			setting.WechatMerchantId,
			serialNo,
			mchPrivateKey,
			setting.WechatApiV3Key,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("初始化微信支付客户端失败: %v", err)
	}

	// 回调验签器：使用自动下载的平台证书
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(setting.WechatMerchantId)
	verifier := verifiers.NewSHA256WithRSAVerifier(certificateVisitor)
	handler := notify.NewNotifyHandler(setting.WechatApiV3Key, verifier)

	wechatBundle = &wechatClientBundle{
		credHash: hash,
		client:   client,
		handler:  handler,
		mchID:    setting.WechatMerchantId,
	}
	return client, handler, nil
}
