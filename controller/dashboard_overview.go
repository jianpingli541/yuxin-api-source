package controller

import (
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/canary"
	"github.com/QuantumNous/new-api/service/guardrail"
	"github.com/QuantumNous/new-api/service/mcp"
	"github.com/QuantumNous/new-api/service/metrics"
	"github.com/QuantumNous/new-api/service/routing"

	"github.com/gin-gonic/gin"
)

// DashboardData 管理后台聚合数据
type DashboardData struct {
	System    SystemInfo             `json:"system"`
	Channels  []ChannelSummary       `json:"channels"`
	Metrics   map[string]interface{} `json:"metrics"`
	Routing   RoutingSummary         `json:"routing"`
	Security  SecuritySummary        `json:"security"`
	MCP       MCPSummary             `json:"mcp"`
	Canary    CanarySummary          `json:"canary"`
	TopModels []ModelUsage           `json:"top_models"`
}

type SystemInfo struct {
	Version     string  `json:"version"`
	Uptime      float64 `json:"uptime_seconds"`
	TotalModels int     `json:"total_models"`
}

type ChannelSummary struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Type         int     `json:"type"`
	Status       int     `json:"status"`
	ResponseTime int     `json:"response_time_ms"`
	UsedQuota    int64   `json:"used_quota"`
	Balance      float64 `json:"balance"`
}

type RoutingSummary struct {
	Strategy            string   `json:"strategy"`
	AvailableStrategies []string `json:"available_strategies"`
}

type SecuritySummary struct {
	Enabled           bool     `json:"enabled"`
	PIIDetection      bool     `json:"pii_detection"`
	ContentModeration bool     `json:"content_moderation"`
	BlockedCategories []string `json:"blocked_categories"`
}

type MCPSummary struct {
	Enabled   bool `json:"enabled"`
	ToolCount int  `json:"tool_count"`
}

type CanarySummary struct {
	Enabled       bool                          `json:"enabled"`
	TestCount     int                           `json:"test_count"`
	ChannelHealth map[int]*canary.ChannelHealth `json:"channel_health"`
}

type ModelUsage struct {
	ModelName        string `json:"model_name"`
	QuotaConsumed    int64  `json:"quota_consumed"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
}

// GetDashboard 获取管理后台聚合数据
// GET /api/dashboard/overview
func GetDashboard(c *gin.Context) {
	// 系统信息
	sysInfo := SystemInfo{
		Version: "v1.0.0-yuxin",
	}

	// 指标
	metricsData := metrics.GetMetricsSummary()
	if uptime, ok := metricsData["uptime_seconds"].(float64); ok {
		sysInfo.Uptime = uptime
	}

	// 模型数
	pricing := model.GetPricing()
	sysInfo.TotalModels = len(pricing)

	// 渠道概览
	allChannels, _ := model.GetAllChannels(0, 100, true, true)
	channels := make([]ChannelSummary, 0, len(allChannels))
	for _, ch := range allChannels {
		channels = append(channels, ChannelSummary{
			ID:           ch.Id,
			Name:         ch.Name,
			Type:         ch.Type,
			Status:       ch.Status,
			ResponseTime: ch.ResponseTime,
			UsedQuota:    ch.UsedQuota,
			Balance:      ch.Balance,
		})
	}

	// 路由配置
	routingCfg := routing.GetDefaultConfig()
	routingSummary := RoutingSummary{
		Strategy: string(routingCfg.Strategy),
		AvailableStrategies: []string{
			string(routing.StrategyPriorityWeight),
			string(routing.StrategyCostOptimized),
			string(routing.StrategyLatencyOptimized),
			string(routing.StrategyQualityFirst),
		},
	}

	// 安全合规
	complianceCfg := guardrail.GetConfig()
	securitySummary := SecuritySummary{
		Enabled:           complianceCfg.Enabled,
		PIIDetection:      complianceCfg.PIIDetection,
		ContentModeration: complianceCfg.ContentModeration,
		BlockedCategories: complianceCfg.BlockedCategories,
	}

	// MCP
	mcpRegistry := mcp.GetRegistry()
	mcpSummary := MCPSummary{
		Enabled:   mcpRegistry.HasTools(),
		ToolCount: len(mcpRegistry.GetTools()),
	}

	// Canary
	canaryEngine := canary.GetEngine()
	canarySummary := CanarySummary{
		Enabled:       canaryEngine.IsEnabled(),
		TestCount:     len(canaryEngine.GetTests()),
		ChannelHealth: canaryEngine.GetAllChannelHealth(),
	}

	data := DashboardData{
		System:   sysInfo,
		Channels: channels,
		Metrics:  metricsData,
		Routing:  routingSummary,
		Security: securitySummary,
		MCP:      mcpSummary,
		Canary:   canarySummary,
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      data,
		"timestamp": time.Now().Unix(),
	})
}
