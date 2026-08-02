package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/canary"
	"github.com/QuantumNous/new-api/service/guardrail"

	"github.com/gin-gonic/gin"
)

// GetComplianceConfig 获取安全合规配置
func GetComplianceConfig(c *gin.Context) {
	cfg := guardrail.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cfg,
	})
}

// UpdateComplianceConfig 更新安全合规配置
func UpdateComplianceConfig(c *gin.Context) {
	var cfg guardrail.ComplianceConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}
	guardrail.UpdateConfig(cfg)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "compliance config updated"})
}

// RunComplianceCheck 执行安全检查（测试用）
func RunComplianceCheck(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}

	// 构造测试请求
	results := guardrail.CheckAllText(req.Text)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

// GetMarketplaceModels 获取模型广场数据（公开API）
func GetMarketplaceModels(c *gin.Context) {
	pricing := model.GetPricing()

	type MarketModel struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Provider     string   `json:"provider"`
		Description  string   `json:"description"`
		Icon         string   `json:"icon"`
		Tags         string   `json:"tags"`
		InputPrice   float64  `json:"input_price"`  // $/1M tokens
		OutputPrice  float64  `json:"output_price"` // $/1M tokens
		Capabilities []string `json:"capabilities"`
		QuotaType    int      `json:"quota_type"`
		VendorID     int      `json:"vendor_id"`
	}

	models := make([]MarketModel, 0, len(pricing))
	for _, p := range pricing {
		mm := MarketModel{
			ID:          p.ModelName,
			Name:        p.ModelName,
			Provider:    p.OwnerBy,
			Description: p.Description,
			Icon:        p.Icon,
			Tags:        p.Tags,
			InputPrice:  p.ModelRatio * 2,
			OutputPrice: p.ModelRatio * p.CompletionRatio * 2,
			QuotaType:   p.QuotaType,
			VendorID:    p.VendorID,
		}
		models = append(models, mm)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"models":  models,
			"total":   len(models),
			"vendors": model.GetVendors(),
		},
	})
}

// GetCanaryStatus 获取Canary质量监控状态
func GetCanaryStatus(c *gin.Context) {
	engine := canary.GetEngine()
	healthMap := engine.GetAllChannelHealth()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":  engine.IsEnabled(),
			"tests":    engine.GetTests(),
			"channels": healthMap,
		},
	})
}

// RunCanaryTest 手动触发Canary测试
func RunCanaryTest(c *gin.Context) {
	var req struct {
		ChannelID int    `json:"channel_id"`
		ModelName string `json:"model_name"`
		BaseURL   string `json:"base_url"`
		APIKey    string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}

	engine := canary.GetEngine()
	results := engine.RunAllTestsForChannel(req.ChannelID, "", req.ModelName, req.BaseURL, req.APIKey)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

// EnableCanary 启用/禁用Canary监控
func EnableCanary(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}

	engine := canary.GetEngine()
	engine.SetEnabled(req.Enabled)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "canary monitoring updated",
	})
}
