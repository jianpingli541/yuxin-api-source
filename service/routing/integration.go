package routing

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetStrategyFromContext 从请求上下文获取路由策略
func GetStrategyFromContext(c *gin.Context) RoutingStrategy {
	// 优先从 Header 读取
	if c != nil {
		strategy := c.GetHeader("X-Routing-Strategy")
		if strategy != "" {
			return RoutingStrategy(strategy)
		}
		// X-Yuxin-Route: 智能调度客户端友好别名（balanced/quality/speed）。
		// 仅当其命中已知策略时生效，避免任意字符串污染策略分发。
		switch c.GetHeader("X-Yuxin-Route") {
		case "balanced":
			return StrategyBalanced
		case "quality":
			return StrategyQualityFirst
		case "speed":
			return StrategyLatencyOptimized
		}
	}

	// 从系统配置读取默认策略
	if val, ok := common.OptionMap["RoutingStrategy"]; ok && val != "" {
		return RoutingStrategy(val)
	}

	return GetRouter().defaultStrategy
}

// GetRouteBiasFromContext 读取 X-Yuxin-Route 的偏好微调（quality/speed/空=均衡）。
// 与 GetStrategyFromContext 独立：balanced 策略下用它调权。
func GetRouteBiasFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	switch c.GetHeader("X-Yuxin-Route") {
	case "quality":
		return "quality"
	case "speed":
		return "speed"
	default:
		return ""
	}
}

// SmartSelectChannel 智能选择渠道
func SmartSelectChannel(c *gin.Context, modelName, group string, channel *model.Channel) *model.Channel {
	if channel == nil {
		return nil
	}

	// 智能调度（model 别名）已在 Distribute 阶段完成 (模型, 渠道) 联合选定，
	// 此处直接放行，避免二次选择把调度结果打散。
	if c != nil && c.GetBool("auto_route_resolved") {
		return channel
	}

	strategy := GetStrategyFromContext(c)
	if strategy == StrategyPriorityWeight || strategy == "" {
		return channel
	}

	abilities, err := model.GetGroupEnabledAbilitiesByModel(group, modelName)
	if err != nil || len(abilities) <= 1 {
		return channel
	}

	var channels []*model.Channel
	for _, ab := range abilities {
		ch, err := model.CacheGetChannel(ab.ChannelId)
		if err != nil || ch == nil {
			continue
		}
		if ch.Status != common.ChannelStatusEnabled {
			continue
		}
		if ab.Priority != nil && channel.Priority != nil && *ab.Priority != *channel.Priority {
			continue
		}
		channels = append(channels, ch)
	}

	if len(channels) <= 1 {
		return channel
	}

	ctx := &RoutingContext{
		ModelName: modelName,
		UserGroup: group,
		Strategy:  strategy,
		RouteBias: GetRouteBiasFromContext(c),
	}

	selected, err := GetRouter().SelectChannel(ctx, channels)
	if err != nil || selected == nil {
		return channel
	}

	return selected
}
