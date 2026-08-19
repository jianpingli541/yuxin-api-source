package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
)

// WechatRefundRequest 管理员发起微信退款
type WechatRefundRequest struct {
	TradeNo string `json:"trade_no"` // 原商户订单号
	Reason  string `json:"reason"`   // 退款原因（可选）
}

// AdminWechatRefund 管理员对已成功的微信支付订单发起退款。
// 流程：校验订单 → 调微信退款 API → 回调 /api/wechat/refund-notify 扣减额度。
// 退款金额默认全额（与原订单 money 一致）。
func AdminWechatRefund(c *gin.Context) {
	var req WechatRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	topUp := model.GetTopUpByTradeNo(req.TradeNo)
	if topUp == nil {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	if topUp.PaymentProvider != model.PaymentProviderWechat {
		common.ApiErrorMsg(c, "非微信支付订单，无法通过此接口退款")
		return
	}
	if topUp.Status != common.TopUpStatusSuccess {
		common.ApiErrorMsg(c, fmt.Sprintf("订单状态为 %s，仅成功订单可退款", topUp.Status))
		return
	}

	ctx := c.Request.Context()
	client, _, err := service.GetWechatClient(ctx)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("微信退款 SDK 初始化失败 trade_no=%s error=%q", req.TradeNo, err.Error()))
		common.ApiErrorMsg(c, "支付配置错误")
		return
	}

	// 退款金额（分）：默认全额
	totalFen := int64(topUp.Money * 100)
	if totalFen < 1 {
		common.ApiErrorMsg(c, "退款金额异常")
		return
	}
	outRefundNo := fmt.Sprintf("RF-%d-%s", time.Now().UnixMilli(), randstr.String(6))

	callbackAddr := service.GetCallbackAddress()
	refundNotifyUrl := callbackAddr + "/api/wechat/refund-notify"
	if setting.WechatNotifyUrl != "" {
		// 复用同一回调前缀：支付 notify_url 同目录下 refund-notify
		refundNotifyUrl = callbackAddr + "/api/wechat/refund-notify"
	}

	var reasonPtr *string
	if req.Reason != "" {
		reasonPtr = &req.Reason
	}

	refundReq := refunddomestic.CreateRequest{
		OutTradeNo:  &topUp.TradeNo,
		OutRefundNo: &outRefundNo,
		Reason:      reasonPtr,
		NotifyUrl:   &refundNotifyUrl,
		Amount: &refunddomestic.AmountReq{
			Refund:   int64Ptr(totalFen),
			Total:    int64Ptr(totalFen),
			Currency: stringPtr("CNY"),
		},
	}

	refundSvc := &refunddomestic.RefundsApiService{Client: client}
	resp, _, err := refundSvc.Create(ctx, refundReq)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("微信退款 API 失败 trade_no=%s refund_no=%s error=%q", topUp.TradeNo, outRefundNo, err.Error()))
		common.ApiErrorMsg(c, "退款请求失败")
		return
	}

	state := ""
	if resp != nil && resp.Status != nil {
		state = string(*resp.Status)
	}
	logger.LogInfo(ctx, fmt.Sprintf("微信退款请求已提交 trade_no=%s refund_no=%s state=%s admin=%s", topUp.TradeNo, outRefundNo, state, c.ClientIP()))
	common.ApiSuccess(c, gin.H{
		"refund_no":    outRefundNo,
		"trade_no":     topUp.TradeNo,
		"refund_state": state,
	})
}

// WechatRefundNotify 处理微信退款回调：成功则扣减用户额度并标记订单
func WechatRefundNotify(c *gin.Context) {
	if !isWechatWebhookEnabled() {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	_, handler, err := service.GetWechatClient(c.Request.Context())
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信退款 webhook 初始化失败 error=%q", err.Error()))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var refundResult refunddomestic.Refund
	notifyReq, err := handler.ParseNotifyRequest(c.Request.Context(), c.Request, &refundResult)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信退款 webhook 验签/解密失败 error=%q", err.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	outTradeNo := ptrString(refundResult.OutTradeNo)
	refundStatus := ""
	if refundResult.Status != nil {
		refundStatus = string(*refundResult.Status)
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信退款 webhook 验签成功 out_trade_no=%s refund_status=%s client_ip=%s summary=%q", outTradeNo, refundStatus, c.ClientIP(), notifyReq.Summary))

	if refundStatus != "SUCCESS" {
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "ignored non-success"})
		return
	}

	if err := model.RefundWechatTopUp(outTradeNo, c.ClientIP()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信退款扣减额度失败 out_trade_no=%s error=%q", outTradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信退款完成 out_trade_no=%s client_ip=%s", outTradeNo, c.ClientIP()))
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "成功"})
}

func int64Ptr(i int64) *int64 { return &i }
