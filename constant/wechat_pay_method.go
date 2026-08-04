package constant

// WechatPayMethod 定义微信支付支持的支付方式（产品形态）。
// 当前主链路只启用 Native（PC 扫码）；JSAPI/H5 预留以便后续扩展。
type WechatPayMethod struct {
	Name          string `json:"name"`          // 前端展示名
	Icon          string `json:"icon"`          // 前端图标路径
	PayMethodType string `json:"payMethodType"` // 微信支付产品类型: NATIVE / JSAPI / H5
	PayMethodName string `json:"payMethodName"` // 展示用副标题，可空
}

// DefaultWechatPayMethods 默认启用的微信支付方式。
// Native 为主：用户在 PC 端扫码完成支付，与现有 Stripe/Waffo 并列。
var DefaultWechatPayMethods = []WechatPayMethod{
	{Name: "微信支付", Icon: "/pay-wechat.png", PayMethodType: "NATIVE", PayMethodName: "扫码支付"},
}
