package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// PublicPricingModel OpenRouter 兼容的公开定价模型
type PublicPricingModel struct {
	ID            string             `json:"id"`
	Provider      string             `json:"provider"`
	Object        string             `json:"object"`
	ContextLength int                `json:"context_length,omitempty"`
	Pricing       PublicPricingPrice `json:"pricing"`
	Capabilities  []string           `json:"capabilities,omitempty"`
}

type PublicPricingPrice struct {
	Prompt          string `json:"prompt"`           // $/1K tokens
	Completion      string `json:"completion"`       // $/1K tokens
	Image           string `json:"image,omitempty"`
	Audio           string `json:"audio,omitempty"`
	AudioCompletion string `json:"audio_completion,omitempty"`
	CacheRead       string `json:"cache_read,omitempty"`
	CacheWrite      string `json:"cache_write,omitempty"`
}

// PublicPricing 公开定价 API —— 无需认证
// GET /api/public/pricing
// 兼容 OpenRouter /api/v1/models 的定价格式，可被 AI Agent 自动拉取
func PublicPricing(c *gin.Context) {
	pricing := model.GetPricing()
	quotaPerUnit := common.QuotaPerUnit
	if quotaPerUnit == 0 {
		quotaPerUnit = 500000
	}

	models := make([]PublicPricingModel, 0, len(pricing))
	for _, p := range pricing {
		if p.QuotaType == 1 && p.ModelPrice > 0 {
			// 按次计费
			models = append(models, PublicPricingModel{
				ID:       p.ModelName,
				Provider: p.OwnerBy,
				Object:   "model",
				Pricing: PublicPricingPrice{
					Prompt:     formatPriceUSD(p.ModelPrice, quotaPerUnit),
					Completion: formatPriceUSD(p.ModelPrice, quotaPerUnit),
				},
				Capabilities: endpointTypesToCapabilities(p.SupportedEndpointTypes),
			})
			continue
		}

		// 按 Token 计费
		pm := PublicPricingModel{
			ID:       p.ModelName,
			Provider: p.OwnerBy,
			Object:   "model",
			Pricing: PublicPricingPrice{
				Prompt:     ratioToPriceUSD(p.ModelRatio, quotaPerUnit),
				Completion: ratioToPriceUSD(p.ModelRatio*p.CompletionRatio, quotaPerUnit),
			},
			Capabilities: endpointTypesToCapabilities(p.SupportedEndpointTypes),
		}

		if p.CacheRatio != nil {
			pm.Pricing.CacheRead = ratioToPriceUSD(p.ModelRatio*(*p.CacheRatio), quotaPerUnit)
		}
		if p.CreateCacheRatio != nil {
			pm.Pricing.CacheWrite = ratioToPriceUSD(p.ModelRatio*(*p.CreateCacheRatio), quotaPerUnit)
		}
		if p.ImageRatio != nil {
			pm.Pricing.Image = formatPriceUSD(p.ModelRatio*(*p.ImageRatio), quotaPerUnit)
		}
		if p.AudioRatio != nil {
			pm.Pricing.Audio = ratioToPriceUSD(p.ModelRatio*(*p.AudioRatio), quotaPerUnit)
		}
		if p.AudioCompletionRatio != nil {
			pm.Pricing.AudioCompletion = ratioToPriceUSD(p.ModelRatio*(*p.AudioCompletionRatio), quotaPerUnit)
		}

		models = append(models, pm)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"models":          models,
			"pricing_version": "yuxin-public-v1",
		},
	})
}

func ratioToPriceUSD(ratio float64, quotaPerUnit float64) string {
	if ratio <= 0 {
		return "0"
	}
	pricePer1K := ratio * 0.002 / (500000.0 / quotaPerUnit)
	return formatFloat(pricePer1K)
}

func formatPriceUSD(price float64, quotaPerUnit float64) string {
	if price <= 0 {
		return "0"
	}
	pricePer1K := price * 0.002 / (500000.0 / quotaPerUnit)
	return formatFloat(pricePer1K)
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.8f", f)
}

func endpointTypesToCapabilities(types []constant.EndpointType) []string {
	caps := make([]string, 0, len(types))
	seen := map[string]bool{}
	for _, t := range types {
		var cap string
		switch string(t) {
		case "openai", "anthropic", "gemini", "openai-response", "openai-response-compact":
			cap = "chat"
		case "embeddings":
			cap = "embeddings"
		case "image-generation":
			cap = "image_generation"
		case "jina-rerank":
			cap = "rerank"
		case "openai-video":
			cap = "video"
		default:
			cap = string(t)
		}
		if !seen[cap] {
			seen[cap] = true
			caps = append(caps, cap)
		}
	}
	return caps
}
