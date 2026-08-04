package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/QuantumNous/new-api/setting"

	"github.com/smartwalle/alipay/v3"
)

// alipayClientBundle 缓存支付宝 client。
// 凭据变化（运营在后台改配置）时重建，避免用旧私钥签名。
type alipayClientBundle struct {
	credHash string
	client   *alipay.Client
}

var (
	alipayBundle   *alipayClientBundle
	alipayBundleMu sync.Mutex
)

// alipayCredHash 用凭据内容派生哈希，判断是否需要重建 client
func alipayCredHash() string {
	h := sha256.Sum256([]byte(
		setting.AlipayAppId + "|" +
			setting.AlipayPrivateKey + "|" +
			setting.AlipayPublicKey + "|" +
			fmt.Sprintf("%v", setting.AlipaySandbox),
	))
	return hex.EncodeToString(h[:])
}

// GetAlipayClient 返回支付宝 client（已加载应用私钥与支付宝公钥）。
// 验签逻辑由 SDK 内部完成（异步通知走 DecodeNotification）。
func GetAlipayClient(ctx context.Context) (*alipay.Client, error) {
	alipayBundleMu.Lock()
	defer alipayBundleMu.Unlock()

	hash := alipayCredHash()
	if alipayBundle != nil && alipayBundle.credHash == hash {
		return alipayBundle.client, nil
	}

	if setting.AlipayPrivateKey == "" || setting.AlipayPublicKey == "" {
		return nil, fmt.Errorf("支付宝凭据未配置（缺少应用私钥或支付宝公钥）")
	}

	client, err := alipay.New(setting.AlipayAppId, setting.AlipayPrivateKey, !setting.AlipaySandbox)
	if err != nil {
		return nil, fmt.Errorf("初始化支付宝客户端失败: %v", err)
	}

	if err := client.LoadAliPayPublicKey(setting.AlipayPublicKey); err != nil {
		return nil, fmt.Errorf("加载支付宝公钥失败: %v", err)
	}

	alipayBundle = &alipayClientBundle{
		credHash: hash,
		client:   client,
	}
	return client, nil
}
