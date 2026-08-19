package routing

import (
	"github.com/QuantumNous/new-api/model"
)

// RoutingStrategy 路由策略类型
type RoutingStrategy string

const (
	// StrategyPriorityWeight 优先级权重策略（现有逻辑）
	StrategyPriorityWeight RoutingStrategy = "priority_weight"
	// StrategyCostOptimized 成本优化策略 - 选择最便宜的渠道
	StrategyCostOptimized RoutingStrategy = "cost_optimized"
	// StrategyLatencyOptimized 延迟优化策略 - 选择响应最快的渠道
	StrategyLatencyOptimized RoutingStrategy = "latency_optimized"
	// StrategyQualityFirst 质量优先策略 - 选择质量最高的渠道
	StrategyQualityFirst RoutingStrategy = "quality_first"
	// StrategyBalanced 双维均衡策略 - 效果(质量分)×速度(延迟)加权融合，
	// 叠加字节生态倾斜（doubao_boost）。智能调度（model 别名）默认策略。
	StrategyBalanced RoutingStrategy = "balanced"
)

// ChannelScore 渠道评分
type ChannelScore struct {
	Channel      *model.Channel
	CostScore    float64 // 成本评分（越低越好）
	LatencyScore float64 // 延迟评分（越低越好）
	QualityScore float64 // 质量评分（越高越好）
	FinalScore   float64 // 综合评分
}

// RoutingContext 路由上下文
type RoutingContext struct {
	ModelName string
	UserGroup string
	Strategy  RoutingStrategy
	// RouteBias balanced 策略下的偏好微调："quality" / "speed" / ""（均衡）。
	// 由 X-Yuxin-Route 请求头映射而来。
	RouteBias    string
	MaxRetries   int
	CurrentRetry int
}

// Router 路由器接口
type Router interface {
	// SelectChannel 选择最佳渠道
	SelectChannel(ctx *RoutingContext, channels []*model.Channel) (*model.Channel, error)
	// GetStrategy 获取路由策略
	GetStrategy() RoutingStrategy
}
