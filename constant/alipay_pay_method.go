package constant

// AlipayPayMethod 定义支付宝支付支持的支付方式（产品形态）。
// 当面付 precreate 为主：用户 PC 端扫码支付，与微信支付 Native 并列。
type AlipayPayMethod struct {
	Name          string `json:"name"`          // 前端展示名
	Icon          string `json:"icon"`          // 前端图标路径
	PayMethodType string `json:"payMethodType"` // 支付宝产品类型: PRECREATE / APP / H5 / WAP / PAGE
	PayMethodName string `json:"payMethodName"` // 展示用副标题，可空
}

// DefaultAlipayPayMethods 默认启用的支付宝支付方式。
// 当面付扫码为主，与微信支付 Native 互补（用户用哪个钱包都能扫）。
var DefaultAlipayPayMethods = []AlipayPayMethod{
	{Name: "支付宝", Icon: "/pay-alipay.png", PayMethodType: "PRECREATE", PayMethodName: "扫码支付"},
}
