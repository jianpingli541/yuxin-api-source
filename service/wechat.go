package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
			setting.WechatPrivateKey + "|" +
			setting.WechatApiV3Key,
	))
	return hex.EncodeToString(h[:])
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

	if setting.WechatPrivateKey == "" || setting.WechatCertSerialNo == "" {
		return nil, nil, fmt.Errorf("微信支付凭据未配置（缺少私钥或证书序列号）")
	}

	mchPrivateKey, err := utils.LoadPrivateKey(setting.WechatPrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("解析微信支付商户私钥失败: %v", err)
	}

	// WithWechatPayAutoAuthCipher：请求自动签名 + 注册平台证书下载器（定时刷新）
	client, err := core.NewClient(ctx,
		option.WithWechatPayAutoAuthCipher(
			setting.WechatMerchantId,
			setting.WechatCertSerialNo,
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
