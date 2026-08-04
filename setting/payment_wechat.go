package setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

// 微信支付（Native 扫码）配置。
// 凭据由运营在 Web 后台（/setting → 支付设置）填写，不进代码、不进 .env 之外的位置。
var (
	// WechatEnabled 是否启用微信支付通道
	WechatEnabled bool
	// WechatMerchantId 微信支付商户号 mchid
	WechatMerchantId string
	// WechatAppId 与商户号绑定的 AppID（公众号/开放平台）
	WechatAppId string
	// WechatApiV3Key API V3 密钥（32 字节），用于回调通知 AES-256-GCM 解密
	WechatApiV3Key string
	// WechatPrivateKey 商户私钥 PEM 内容（apiclient_key.pem），用于请求签名
	WechatPrivateKey string
	// WechatCertSerialNo 商户证书序列号，用于请求头 WECHATPAY2-SHA256-RSA2048 的 serial_no
	WechatCertSerialNo string
	// WechatNotifyUrl 支付/退款回调地址，留空则自动拼 GetCallbackAddress()+/api/wechat/webhook
	WechatNotifyUrl string
	// WechatReturnUrl 支付完成跳转地址，留空默认回到钱包页
	WechatReturnUrl string
	// WechatUnitPrice 单充值单位价格（CNY），默认 1.0
	WechatUnitPrice float64 = 1.0
	// WechatMinTopUp 最小充值数量
	WechatMinTopUp int = 1
)

// GetWechatPayMethods 从 OptionMap 读取微信支付方式配置
func GetWechatPayMethods() []constant.WechatPayMethod {
	common.OptionMapRWMutex.RLock()
	jsonStr := common.OptionMap["WechatPayMethods"]
	common.OptionMapRWMutex.RUnlock()

	if jsonStr == "" {
		return copyDefaultWechatPayMethods()
	}
	var methods []constant.WechatPayMethod
	if err := common.UnmarshalJsonStr(jsonStr, &methods); err != nil {
		return copyDefaultWechatPayMethods()
	}
	return methods
}

// SetWechatPayMethods 序列化微信支付方式配置并写回 OptionMap
func SetWechatPayMethods(methods []constant.WechatPayMethod) error {
	jsonBytes, err := common.Marshal(methods)
	if err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	common.OptionMap["WechatPayMethods"] = string(jsonBytes)
	common.OptionMapRWMutex.Unlock()
	return nil
}

func copyDefaultWechatPayMethods() []constant.WechatPayMethod {
	cp := make([]constant.WechatPayMethod, len(constant.DefaultWechatPayMethods))
	copy(cp, constant.DefaultWechatPayMethods)
	return cp
}

// WechatPayMethods2JsonString 将默认微信支付方式序列化为 JSON（供 InitOptionMap 使用）
func WechatPayMethods2JsonString() string {
	jsonBytes, err := common.Marshal(constant.DefaultWechatPayMethods)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

// SetWechatPayMethodsFromJSON 从 JSON 字符串解析支付方式并写回 OptionMap
func SetWechatPayMethodsFromJSON(jsonStr string) error {
	var methods []constant.WechatPayMethod
	if err := common.UnmarshalJsonStr(jsonStr, &methods); err != nil {
		return err
	}
	return SetWechatPayMethods(methods)
}
