package service

import (
	"context"
	"log"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	alipay "github.com/smartwalle/alipay/v3"
)

// StartAlipayReconciliation 支付宝后台对账任务：每 30s 主动查单。
// 原理：alipay SDK 有 client.TradeQuery，调用 attempt 已在 SDK 缓存内。
// 状态 TRADE_SUCCESS 主动入账。
func StartAlipayReconciliation(ctx context.Context) {
	if !setting.AlipayEnabled {
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
				reconcilePendingAlipayOrders()
			}
		}
	}()
}

func reconcilePendingAlipayOrders() {
	cutoff := time.Now().Add(-5 * time.Minute).Unix()
	pending, err := model.ListPendingTopUpsByProvider(model.PaymentProviderAlipay, cutoff, 50)
	if err != nil {
		log.Printf("[alipay-reconcile] list pending failed: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	client, err := GetAlipayClient(context.Background())
	if err != nil {
		return
	}
	for _, topUp := range pending {
		tradeNo := topUp.TradeNo
		rsp, err := client.TradeQuery(context.Background(), alipay.TradeQuery{OutTradeNo: tradeNo})
		if err != nil || rsp == nil {
			continue
		}
		if rsp.TradeStatus != alipay.TradeStatusSuccess && rsp.TradeStatus != alipay.TradeStatusFinished {
			continue
		}
		if err := model.RechargeAlipay(tradeNo, "alipay-reconcile"); err != nil {
			log.Printf("[alipay-reconcile] recharge failed trade_no=%s err=%v", tradeNo, err)
			continue
		}
		log.Printf("[alipay-reconcile] auto-reconciled pending order trade_no=%s", tradeNo)
	}
}
