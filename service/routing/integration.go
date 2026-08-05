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
	}

	// 从系统配置读取默认策略
	if val, ok := common.OptionMap["RoutingStrategy"]; ok && val != "" {
		return RoutingStrategy(val)
	}

	return GetRouter().defaultStrategy
}

// SmartSelectChannel 智能选择渠道
func SmartSelectChannel(c *gin.Context, modelName, group string, channel *model.Channel) *model.Channel {
	if channel == nil {
		return nil
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
	}

	selected, err := GetRouter().SelectChannel(ctx, channels)
	if err != nil || selected == nil {
		return channel
	}

	return selected
}
