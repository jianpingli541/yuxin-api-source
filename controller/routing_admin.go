package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/routing"

	"github.com/gin-gonic/gin"
)

// GetRoutingConfig 获取路由配置
func GetRoutingConfig(c *gin.Context) {
	config := routing.GetDefaultConfig()

	// 获取所有渠道的质量评分
	channels, _ := model.GetAllChannels(0, 10000, false, false)
	qualityMap := make(map[int]float64)
	for _, ch := range channels {
		qualityMap[ch.Id] = float64(ch.ResponseTime)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"strategy":       config.Strategy,
			"fallback_chain": config.FallbackChain,
			"quality_scores": qualityMap,
			"available_strategies": []string{
				string(routing.StrategyPriorityWeight),
				string(routing.StrategyCostOptimized),
				string(routing.StrategyLatencyOptimized),
				string(routing.StrategyQualityFirst),
			},
		},
	})
}

// UpdateRoutingConfig 更新路由配置
func UpdateRoutingConfig(c *gin.Context) {
	var req struct {
		Strategy      string `json:"strategy"`
		FallbackChain []int  `json:"fallback_chain"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}

	config := routing.GetDefaultConfig()
	if req.Strategy != "" {
		config.Strategy = routing.RoutingStrategy(req.Strategy)
	}
	if req.FallbackChain != nil {
		config.FallbackChain = req.FallbackChain
	}

	routing.SetDefaultConfig(config)

	c.JSON(http.StatusOK, gin.H{"success": true})
}
