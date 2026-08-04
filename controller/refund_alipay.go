package controller

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
	"github.com/smartwalle/alipay/v3"
)

// AlipayRefundRequest 管理员发起支付宝退款
type AlipayRefundRequest struct {
	TradeNo string `json:"trade_no"` // 原商户订单号
	Reason  string `json:"reason"`   // 退款原因（可选）
}

// AdminAlipayRefund 管理员对已成功的支付宝订单发起退款。
// 支付宝退款为同步 API：alipay.trade.refund 直接返回结果，无需退款回调。
// 流程：校验订单 → 调支付宝退款 API → 成功后立即扣减额度。
func AdminAlipayRefund(c *gin.Context) {
	var req AlipayRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	topUp := model.GetTopUpByTradeNo(req.TradeNo)
	if topUp == nil {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	if topUp.PaymentProvider != model.PaymentProviderAlipay {
		common.ApiErrorMsg(c, "非支付宝订单，无法通过此接口退款")
		return
	}
	if topUp.Status != common.TopUpStatusSuccess {
		common.ApiErrorMsg(c, fmt.Sprintf("订单状态为 %s，仅成功订单可退款", topUp.Status))
		return
	}

	ctx := c.Request.Context()
	client, err := service.GetAlipayClient(ctx)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝退款 SDK 初始化失败 trade_no=%s error=%q", req.TradeNo, err.Error()))
		common.ApiErrorMsg(c, "支付配置错误")
		return
	}

	// 退款金额（元）：默认全额
	refundAmount := fmt.Sprintf("%.2f", topUp.Money)
	if topUp.Money < 0.01 {
		common.ApiErrorMsg(c, "退款金额异常")
		return
	}
	outRequestNo := fmt.Sprintf("RF-%d-%s", time.Now().UnixMilli(), randstr.String(6))

	refundParam := alipay.TradeRefund{
		OutTradeNo:   topUp.TradeNo,
		RefundAmount: refundAmount,
		OutRequestNo: outRequestNo,
	}
	if req.Reason != "" {
		refundParam.RefundReason = req.Reason
	}

	rsp, err := client.TradeRefund(ctx, refundParam)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝退款 API 失败 trade_no=%s error=%q", topUp.TradeNo, err.Error()))
		common.ApiErrorMsg(c, "退款请求失败")
		return
	}
	if !rsp.IsSuccess() {
		logger.LogError(ctx, fmt.Sprintf("支付宝退款业务失败 trade_no=%s code=%s msg=%q sub_code=%s sub_msg=%q", topUp.TradeNo, rsp.Code, rsp.Msg, rsp.SubCode, rsp.SubMsg))
		common.ApiErrorMsg(c, fmt.Sprintf("退款失败: %s", rsp.SubMsg))
		return
	}
	// FundChange == "Y" 表示本次退款发生了资金变化（全额退款时部分渠道返回 N 需人工核对）
	logger.LogInfo(ctx, fmt.Sprintf("支付宝退款成功 trade_no=%s refund_no=%s refund_fee=%s fund_change=%s admin=%s", topUp.TradeNo, outRequestNo, rsp.RefundFee, rsp.FundChange, c.ClientIP()))

	// 同步扣减额度
	if err := model.RefundAlipayTopUp(topUp.TradeNo, c.ClientIP()); err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝退款扣减额度失败 trade_no=%s error=%q（资金已退，需人工核对额度）", topUp.TradeNo, err.Error()))
		common.ApiErrorMsg(c, "资金已退款，但扣减额度失败，请联系管理员核对")
		return
	}
	logger.LogInfo(ctx, fmt.Sprintf("支付宝退款完成并已扣减额度 trade_no=%s", topUp.TradeNo))
	common.ApiSuccess(c, gin.H{
		"refund_no":  outRequestNo,
		"trade_no":   topUp.TradeNo,
		"refund_fee": rsp.RefundFee,
	})
}
