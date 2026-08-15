package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/smartwalle/alipay/v3"
	"github.com/thanhpk/randstr"
)

// AlipayPayRequest 前端发起支付宝充值时提交的请求体
type AlipayPayRequest struct {
	Amount         int64 `json:"amount"`           // 充值数量（与现有 waffo/stripe/wechat 一致）
	PayMethodIndex *int  `json:"pay_method_index"` // 支付方式索引（precreate 必传 0）
}

func getAlipayPayMoney(amount float64, group string) float64 {
	originalAmount := amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = amount / common.QuotaPerUnit
	}
	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(originalAmount)]; ok {
		if ds > 0 {
			discount = ds
		}
	}
	return amount * setting.AlipayUnitPrice * topupGroupRatio * discount
}

// RequestAlipayAmount 报价接口：返回本次充值实际需要支付的 CNY 金额（保留两位小数）
func RequestAlipayAmount(c *gin.Context) {
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	alipayMin := int64(setting.AlipayMinTopUp)
	if req.Amount < alipayMin {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", alipayMin)})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getAlipayPayMoney(float64(req.Amount), group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

// RequestAlipayPay 兼容入口：原 /api/user/self/alipay/pay 路由保留，
// 实际逻辑下沉到 requestAlipayPayCore 供 RequestEpay 聚合通道复用。
func RequestAlipayPay(c *gin.Context) {
	if !isAlipayTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝未启用"})
		return
	}
	var req AlipayPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	requestAlipayPayCore(c, req.Amount, req.PayMethodIndex)
}

// requestAlipayPayCore 创建支付宝当面付 precreate 订单的原子能力。
// 聚合通道 RequestEpay 与独立入口 RequestAlipayPay 共用此函数。
func requestAlipayPayCore(c *gin.Context, amount int64, payMethodIndex *int) {
	alipayMin := int64(setting.AlipayMinTopUp)
	if amount < alipayMin {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", alipayMin)})
		return
	}

	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}

	if payMethodIndex == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "请选择支付方式"})
		return
	}
	methods := setting.GetAlipayPayMethods()
	idx := *payMethodIndex
	if idx < 0 || idx >= len(methods) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "不支持的支付方式"})
		return
	}
	if methods[idx].PayMethodType != "PRECREATE" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前仅支持 PRECREATE 当面付扫码"})
		return
	}

	group, _ := model.GetUserGroup(id, true)
	payMoney := getAlipayPayMoney(float64(amount), group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	// 商户订单号：ALIPAY-{userId}-{ts_ms}-{rand}
	outTradeNo := fmt.Sprintf("ALIPAY-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(6))

	// Token 模式下归一化 Amount，避免入账双重放大
	// amount 已由参数传入, Token 模式下做归一化
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = int64(float64(amount) / common.QuotaPerUnit)
		if amount < 1 {
			amount = 1
		}
	}

	// 本地订单：先 pending，等异步通知入账
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           payMoney,
		TradeNo:         outTradeNo,
		PaymentMethod:   model.PaymentMethodAlipay,
		PaymentProvider: model.PaymentProviderAlipay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, outTradeNo, amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	ctx := c.Request.Context()
	client, err := service.GetAlipayClient(ctx)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝 SDK 初始化失败 user_id=%d trade_no=%s error=%q", id, outTradeNo, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付配置错误"})
		return
	}

	// 回调地址
	callbackAddr := service.GetCallbackAddress()
	notifyUrl := callbackAddr + "/api/alipay/notify"
	if setting.AlipayNotifyUrl != "" {
		notifyUrl = setting.AlipayNotifyUrl
	}

	totalAmount := fmt.Sprintf("%.2f", payMoney)
	subject := fmt.Sprintf("账号 %d 充值", user.Id)

	preCreate := alipay.TradePreCreate{
		Trade: alipay.Trade{
			NotifyURL:   notifyUrl,
			Subject:     subject,
			OutTradeNo:  outTradeNo,
			TotalAmount: totalAmount,
			ProductCode: "FACE_TO_FACE_PAYMENT",
		},
	}

	rsp, err := client.TradePreCreate(ctx, preCreate)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝 Precreate 失败 user_id=%d trade_no=%s error=%q", id, outTradeNo, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	if !rsp.IsSuccess() {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝 Precreate 业务失败 user_id=%d trade_no=%s code=%s msg=%q sub_code=%s sub_msg=%q", id, outTradeNo, rsp.Code, rsp.Msg, rsp.SubCode, rsp.SubMsg))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	if rsp.QRCode == "" {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝 Precreate 未返回二维码 user_id=%d trade_no=%s rsp=%+v", id, outTradeNo, rsp))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("支付宝 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f", id, outTradeNo, amount, payMoney))

	expireAt := time.Now().Add(15 * time.Minute).Unix() // 支付宝默认 15 分钟订单有效期
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"qr_code":   rsp.QRCode,
			"order_id":  outTradeNo,
			"expire_at": expireAt,
		},
	})
}

// AlipayNotify 处理支付宝异步通知（验签 + 订单入账）
func AlipayNotify(c *gin.Context) {
	if !isAlipayWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	ctx := c.Request.Context()
	client, err := service.GetAlipayClient(ctx)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝 webhook 初始化失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	// SDK 自动验签（用支付宝公钥 RSA2 验签 + 校验 app_id）
	if err := c.Request.ParseForm(); err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝 webhook 表单解析失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	notification, err := client.DecodeNotification(ctx, c.Request.Form)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝 webhook 验签失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	outTradeNo := notification.OutTradeNo
	tradeStatus := notification.TradeStatus
	logger.LogInfo(ctx, fmt.Sprintf("支付宝 webhook 验签成功 out_trade_no=%s trade_status=%s total_amount=%s client_ip=%s", outTradeNo, tradeStatus, notification.TotalAmount, c.ClientIP()))

	switch tradeStatus {
	case alipay.TradeStatusSuccess:
		// 成功
		if outTradeNo == "" {
			logger.LogWarn(ctx, "支付宝 webhook 缺少 out_trade_no")
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		LockOrder(outTradeNo)
		defer UnlockOrder(outTradeNo)
		if err := model.RechargeAlipay(outTradeNo, c.ClientIP()); err != nil {
			logger.LogError(ctx, fmt.Sprintf("支付宝 充值处理失败 out_trade_no=%s client_ip=%s error=%q", outTradeNo, c.ClientIP(), err.Error()))
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		logger.LogInfo(ctx, fmt.Sprintf("支付宝 充值成功 out_trade_no=%s client_ip=%s", outTradeNo, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("success"))

	case alipay.TradeStatusClosed:
		// 未付款超时关闭 / 全额退款关闭
		if outTradeNo != "" {
			_ = model.UpdatePendingTopUpStatus(outTradeNo, model.PaymentProviderAlipay, common.TopUpStatusFailed)
		}
		_, _ = c.Writer.Write([]byte("success"))

	default:
		// WAIT_BUYER_PAY 等中间状态：直接 ack success，避免重试
		logger.LogInfo(ctx, fmt.Sprintf("支付宝 webhook 中间状态 out_trade_no=%s status=%s", outTradeNo, tradeStatus))
		_, _ = c.Writer.Write([]byte("success"))
	}
}

// AlipayNativeQuery 前端轮询入口：主动查询支付宝订单状态并在已支付时入账。
// 作用：与微信路径完全对称——回调可能因平台公钥/网络抖动失败，
// 本接口用 SDK 主动查单作为入账兜底。幂等：RechargeAlipay 内部对已成功直接返回。
func AlipayNativeQuery(c *gin.Context) {
	if !isAlipayTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝原生通道未启用"})
		return
	}
	tradeNo := c.Query("trade_no")
	if tradeNo == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "缺少 trade_no"})
		return
	}
	userId := c.GetInt("id")
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.UserId != userId {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "订单不存在"})
		return
	}
	if topUp.Status == common.TopUpStatusSuccess {
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"status": "paid"}})
		return
	}
	client, err := service.GetAlipayClient(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"status": "pending"}})
		return
	}
	rsp, err := client.TradeQuery(c.Request.Context(), alipay.TradeQuery{OutTradeNo: tradeNo})
	if err != nil || rsp == nil {
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"status": "pending"}})
		return
	}
	if rsp.TradeStatus != alipay.TradeStatusSuccess && rsp.TradeStatus != alipay.TradeStatusFinished {
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"status": "pending"}})
		return
	}
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)
	if err := model.RechargeAlipay(tradeNo, c.ClientIP()); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "入账失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"status": "paid"}})
}
