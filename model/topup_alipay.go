package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	// PaymentMethodAlipay 支付宝支付方式标识
	PaymentMethodAlipay = "alipay"
	// PaymentProviderAlipay 支付宝服务商标识
	PaymentProviderAlipay = "alipay"
)

// RechargeAlipay 处理支付宝支付成功回调的入账。
// 幂等：已成功直接返回；订单不存在/状态错误返回错误。
func RechargeAlipay(tradeNo string, callerIp string) (err error) {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}
		if topUp.PaymentProvider != PaymentProviderAlipay {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			return nil // 幂等
		}
		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		dAmount := decimal.NewFromInt(topUp.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota + ?", quotaToAdd)).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		common.SysError("alipay topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if quotaToAdd > 0 {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("支付宝充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodAlipay)
	}
	return nil
}

// RefundAlipayTopUp 处理支付宝退款成功后的额度扣减。
// 支付宝退款为同步 API（alipay.trade.refund 直接返回结果），调用方在退款成功后调用本函数。
// 幂等：已退款直接返回。
func RefundAlipayTopUp(tradeNo string, callerIp string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToDeduct int
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return errors.New("充值订单不存在")
		}
		if topUp.PaymentProvider != PaymentProviderAlipay {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusRefunded {
			return nil // 幂等
		}
		if topUp.Status != common.TopUpStatusSuccess {
			return errors.New("仅成功订单可退款")
		}

		dAmount := decimal.NewFromInt(topUp.Amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quotaToDeduct = int(dAmount.Mul(dQuotaPerUnit).IntPart())
		if quotaToDeduct <= 0 {
			return errors.New("无效的退款额度")
		}

		// 检查用户余额是否足够扣回
		var user User
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).First(&user).Error; err != nil {
			return errors.New("用户不存在")
		}
		if int64(user.Quota) < int64(quotaToDeduct) {
			return fmt.Errorf("用户余额不足，无法退款：当前 %d，需扣减 %d", user.Quota, quotaToDeduct)
		}

		topUp.Status = common.TopUpStatusRefunded
		topUp.CompleteTime = common.GetTimestamp()
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", topUp.UserId).Update("quota", gorm.Expr("quota - ?", quotaToDeduct)).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		common.SysError("alipay refund failed: " + err.Error())
		return errors.New("退款失败，请稍后重试")
	}
	if quotaToDeduct > 0 {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("支付宝退款，扣减额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToDeduct), topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodAlipay)
	}
	return nil
}
