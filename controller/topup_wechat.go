package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

// WechatPayRequest 前端发起微信充值时提交的请求体
type WechatPayRequest struct {
	Amount         int64 `json:"amount"`           // 充值数量（与现有 epay/waffo 一致，按展示类型传值）
	PayMethodIndex *int  `json:"pay_method_index"` // 支付方式索引（Native 必传 0）
}

func getWechatPayMoney(amount float64, group string) float64 {
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
	return amount * setting.WechatUnitPrice * topupGroupRatio * discount
}

// RequestWechatAmount 报价接口：返回本次充值实际需要支付的 CNY 金额（保留两位小数）
func RequestWechatAmount(c *gin.Context) {
	var req WechatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	wechatMin := int64(setting.WechatMinTopUp)
	if req.Amount < wechatMin {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", wechatMin)})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getWechatPayMoney(float64(req.Amount), group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

// RequestWechatPay 创建微信 Native 支付订单，返回 code_url（二维码内容）
func RequestWechatPay(c *gin.Context) {
	if !isWechatTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "微信支付未启用"})
		return
	}

	var req WechatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	wechatMin := int64(setting.WechatMinTopUp)
	if req.Amount < wechatMin {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", wechatMin)})
		return
	}

	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}

	if req.PayMethodIndex == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "请选择支付方式"})
		return
	}
	methods := setting.GetWechatPayMethods()
	idx := *req.PayMethodIndex
	if idx < 0 || idx >= len(methods) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "不支持的支付方式"})
		return
	}
	if methods[idx].PayMethodType != "NATIVE" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前仅支持 NATIVE 扫码支付"})
		return
	}

	group, _ := model.GetUserGroup(id, true)
	payMoney := getWechatPayMoney(float64(req.Amount), group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	outTradeNo := fmt.Sprintf("WECHAT-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(6))

	// Token 模式下归一化 Amount，避免 RechargeWechat 双重放大
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = int64(float64(req.Amount) / common.QuotaPerUnit)
		if amount < 1 {
			amount = 1
		}
	}

	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           payMoney,
		TradeNo:         outTradeNo,
		PaymentMethod:   model.PaymentMethodWechat,
		PaymentProvider: model.PaymentProviderWechat,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, outTradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	ctx := c.Request.Context()
	client, _, err := service.GetWechatClient(ctx)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 SDK 初始化失败 user_id=%d trade_no=%s error=%q", id, outTradeNo, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付配置错误"})
		return
	}

	callbackAddr := service.GetCallbackAddress()
	notifyUrl := callbackAddr + "/api/wechat/notify"
	if setting.WechatNotifyUrl != "" {
		notifyUrl = setting.WechatNotifyUrl
	}

	// 微信金额单位为"分"
	totalFen := int64(payMoney * 100)
	if totalFen < 1 {
		totalFen = 1
	}

	desc := fmt.Sprintf("账号 %d 充值", user.Id)
	expireAt := time.Now().Add(30 * time.Minute)

	prepayReq := native.PrepayRequest{
		Appid:       &setting.WechatAppId,
		Mchid:       &setting.WechatMerchantId,
		Description: &desc,
		OutTradeNo:  &outTradeNo,
		TimeExpire:  &expireAt,
		NotifyUrl:   &notifyUrl,
		Amount: &native.Amount{
			Total:    int64Ptr(totalFen),
			Currency: stringPtr("CNY"),
		},
	}

	nativeSvc := &native.NativeApiService{Client: client}
	resp, _, err := nativeSvc.Prepay(ctx, prepayReq)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 Prepay 失败 user_id=%d trade_no=%s error=%q", id, outTradeNo, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	if resp == nil || resp.CodeUrl == nil || *resp.CodeUrl == "" {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付 Prepay 业务失败 user_id=%d trade_no=%s resp=%+v", id, outTradeNo, resp))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f", id, outTradeNo, req.Amount, payMoney))

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"code_url":  *resp.CodeUrl,
			"order_id":  outTradeNo,
			"expire_at": expireAt.Unix(),
		},
	})
}

// WechatNotify 处理微信支付回调通知（验签 + AES-GCM 解密 + 订单入账）
func WechatNotify(c *gin.Context) {
	if !isWechatWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	_, handler, err := service.GetWechatClient(c.Request.Context())
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 webhook 初始化失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var tx payments.Transaction
	notifyReq, err := handler.ParseNotifyRequest(c.Request.Context(), c.Request, &tx)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付 webhook 验签/解密失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	outTradeNo := ptrString(tx.OutTradeNo)
	tradeState := ptrString(tx.TradeState)
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付 webhook 验签成功 out_trade_no=%s trade_state=%s client_ip=%s summary=%q", outTradeNo, tradeState, c.ClientIP(), notifyReq.Summary))

	if tradeState == "SUCCESS" {
		if outTradeNo == "" {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付 webhook 缺少 out_trade_no client_ip=%s", c.ClientIP()))
			c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": "missing out_trade_no"})
			return
		}
		LockOrder(outTradeNo)
		defer UnlockOrder(outTradeNo)
		if err := model.RechargeWechat(outTradeNo, c.ClientIP()); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 充值处理失败 out_trade_no=%s client_ip=%s error=%q", outTradeNo, c.ClientIP(), err.Error()))
			c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": err.Error()})
			return
		}
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付 充值成功 out_trade_no=%s client_ip=%s", outTradeNo, c.ClientIP()))
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
		return
	}

	// 非成功终态：标记为 failed，避免永远 pending
	if isTerminalTradeState(tradeState) && outTradeNo != "" {
		if err := model.UpdatePendingTopUpStatus(outTradeNo, model.PaymentProviderWechat, common.TopUpStatusFailed); err != nil &&
			!errors.Is(err, model.ErrTopUpNotFound) &&
			!errors.Is(err, model.ErrTopUpStatusInvalid) {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 标记失败订单状态失败 out_trade_no=%s error=%q", outTradeNo, err.Error()))
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "ok"})
}

// isTerminalTradeState 微信支付订单终态（成功/已退款/已关闭/已撤销/支付失败）
func isTerminalTradeState(state string) bool {
	switch strings.ToUpper(state) {
	case "SUCCESS", "REFUND", "CLOSED", "REVOKED", "PAYERROR":
		return true
	}
	return false
}

// ptrString 安全解引用 *string
func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// stringPtr SDK 模型字段为指针类型，统一构造助手
func stringPtr(s string) *string { return &s }
