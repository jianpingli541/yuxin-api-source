package setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

// 支付宝（当面付 precreate 扫码）配置。
// 凭据由运营在 Web 后台填写，不进代码、不进 .env 之外的位置。
var (
	// AlipayEnabled 是否启用支付宝通道
	AlipayEnabled bool
	// AlipayAppId 支付宝开放平台应用 ID
	AlipayAppId string
	// AlipayPrivateKey 应用私钥 PEM 内容（用于请求签名 RSA2）
	AlipayPrivateKey string
	// AlipayPublicKey 支付宝公钥 PEM 内容（用于异步通知验签 RSA2）
	AlipayPublicKey string
	// AlipaySandbox 是否沙箱环境
	AlipaySandbox bool
	// AlipayNotifyUrl 异步通知地址，留空则自动拼 GetCallbackAddress()+/api/alipay/notify
	AlipayNotifyUrl string
	// AlipayReturnUrl 同步返回地址（当面付一般不用，留作未来扩展）
	AlipayReturnUrl string
	// AlipayUnitPrice 单充值单位价格（CNY），默认 1.0
	AlipayUnitPrice float64 = 1.0
	// AlipayMinTopUp 最小充值数量
	AlipayMinTopUp int = 1
)

// GetAlipayPayMethods 从 OptionMap 读取支付宝支付方式配置
func GetAlipayPayMethods() []constant.AlipayPayMethod {
	common.OptionMapRWMutex.RLock()
	jsonStr := common.OptionMap["AlipayPayMethods"]
	common.OptionMapRWMutex.RUnlock()

	if jsonStr == "" {
		return copyDefaultAlipayPayMethods()
	}
	var methods []constant.AlipayPayMethod
	if err := common.UnmarshalJsonStr(jsonStr, &methods); err != nil {
		return copyDefaultAlipayPayMethods()
	}
	return methods
}

// SetAlipayPayMethods 序列化支付宝支付方式配置并写回 OptionMap
func SetAlipayPayMethods(methods []constant.AlipayPayMethod) error {
	jsonBytes, err := common.Marshal(methods)
	if err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	common.OptionMap["AlipayPayMethods"] = string(jsonBytes)
	common.OptionMapRWMutex.Unlock()
	return nil
}

// SetAlipayPayMethodsFromJSON 从 JSON 字符串解析支付方式并写回 OptionMap
func SetAlipayPayMethodsFromJSON(jsonStr string) error {
	var methods []constant.AlipayPayMethod
	if err := common.UnmarshalJsonStr(jsonStr, &methods); err != nil {
		return err
	}
	return SetAlipayPayMethods(methods)
}

func copyDefaultAlipayPayMethods() []constant.AlipayPayMethod {
	cp := make([]constant.AlipayPayMethod, len(constant.DefaultAlipayPayMethods))
	copy(cp, constant.DefaultAlipayPayMethods)
	return cp
}

// AlipayPayMethods2JsonString 将默认支付方式序列化为 JSON（供 InitOptionMap 使用）
func AlipayPayMethods2JsonString() string {
	jsonBytes, err := common.Marshal(constant.DefaultAlipayPayMethods)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}
