package service

import (
	"context"
	"log"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	nativeSvc "github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

// StartWechatReconciliation 启动后台对账任务：每 30 秒轮询所有 5 分钟内
// pending 状态的微信订单，通过 SDK QueryOrderByOutTradeNo 主动查单。
// 若微信侧已 SUCCESS，则幂等入账（RechargeWechat 内部对已成功订单直接返回）。
// 这绕开了回调通道的脆弱性（平台公钥迁移、网络抖动、回调超时等），
// 保证真实付款一定入账。
func StartWechatReconciliation(ctx context.Context) {
	if !setting.WechatEnabled {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reconcilePendingWechatOrders()
			}
		}
	}()
}

func reconcilePendingWechatOrders() {
	cutoff := time.Now().Add(-1 * time.Minute).Unix()
	pending, err := model.ListPendingTopUpsByProvider(model.PaymentProviderWechat, cutoff, 50)
	if err != nil {
		log.Printf("[wechat-reconcile] list pending failed: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	client, _, err := GetWechatClient(context.Background())
	if err != nil {
		return
	}
	svc := nativeSvc.NativeApiService{Client: client}
	for _, topUp := range pending {
		tradeNo := topUp.TradeNo
		rsp, _, err := svc.QueryOrderByOutTradeNo(context.Background(), nativeSvc.QueryOrderByOutTradeNoRequest{
			Mchid:      strPtr(setting.WechatMerchantId),
			OutTradeNo: strPtr(tradeNo),
		})
		if err != nil || rsp == nil || rsp.TradeState == nil {
			continue
		}
		if *rsp.TradeState != "SUCCESS" {
			continue
		}
		if err := model.RechargeWechat(tradeNo, "wechat-reconcile"); err != nil {
			log.Printf("[wechat-reconcile] recharge failed trade_no=%s err=%v", tradeNo, err)
			continue
		}
		log.Printf("[wechat-reconcile] auto-reconciled pending order trade_no=%s", tradeNo)
	}
}

func strPtr(s string) *string { return &s }
