package routing

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// SmartRouter 智能路由引擎
type SmartRouter struct {
	defaultStrategy RoutingStrategy
	qualityScores   map[int]float64 // channel_id -> quality_score (0-100)
	scoreMutex      sync.RWMutex
}

var (
	globalRouter *SmartRouter
	once         sync.Once
)

// GetRouter 获取全局路由器实例
func GetRouter() *SmartRouter {
	once.Do(func() {
		globalRouter = &SmartRouter{
			defaultStrategy: StrategyPriorityWeight,
			qualityScores:   make(map[int]float64),
		}
	})
	return globalRouter
}

// SetStrategy 设置默认路由策略
func (r *SmartRouter) SetStrategy(strategy RoutingStrategy) {
	r.defaultStrategy = strategy
}

// UpdateQualityScore 更新渠道质量评分（由 Canary 测试模块调用）
func (r *SmartRouter) UpdateQualityScore(channelID int, score float64) {
	r.scoreMutex.Lock()
	defer r.scoreMutex.Unlock()
	r.qualityScores[channelID] = score
}

// GetQualityScore 获取渠道质量评分
func (r *SmartRouter) GetQualityScore(channelID int) float64 {
	r.scoreMutex.RLock()
	defer r.scoreMutex.RUnlock()
	if score, ok := r.qualityScores[channelID]; ok {
		return score
	}
	return 75.0 // 默认分数
}

// SelectChannel 根据策略选择最佳渠道
func (r *SmartRouter) SelectChannel(ctx *RoutingContext, channels []*model.Channel) (*model.Channel, error) {
	if len(channels) == 0 {
		return nil, errors.New("no available channels")
	}
	if len(channels) == 1 {
		return channels[0], nil
	}

	strategy := ctx.Strategy
	if strategy == "" {
		strategy = r.defaultStrategy
	}

	var selected *model.Channel
	switch strategy {
	case StrategyCostOptimized:
		selected = r.selectByCost(channels)
	case StrategyLatencyOptimized:
		selected = r.selectByLatency(channels)
	case StrategyQualityFirst:
		selected = r.selectByQuality(channels)
	default:
		selected = r.selectByPriorityWeight(channels)
	}

	common.SysLog(fmt.Sprintf("[SmartRouter] strategy=%s selected channel #%d (%s)",
		strategy, selected.Id, selected.Name))
	return selected, nil
}

// selectByCost 成本优化选择
func (r *SmartRouter) selectByCost(channels []*model.Channel) *model.Channel {
	best := channels[0]
	bestScore := scoreChannelCost(best)
	for _, ch := range channels[1:] {
		s := scoreChannelCost(ch)
		if s < bestScore {
			bestScore = s
			best = ch
		}
	}
	return best
}

// selectByLatency 延迟优化选择
func (r *SmartRouter) selectByLatency(channels []*model.Channel) *model.Channel {
	best := channels[0]
	bestTime := best.ResponseTime
	if bestTime == 0 {
		bestTime = 999999
	}
	for _, ch := range channels[1:] {
		if ch.ResponseTime > 0 && ch.ResponseTime < bestTime {
			bestTime = ch.ResponseTime
			best = ch
		}
	}
	return best
}

// selectByQuality 质量优先选择
func (r *SmartRouter) selectByQuality(channels []*model.Channel) *model.Channel {
	best := channels[0]
	bestScore := r.GetQualityScore(best.Id)
	for _, ch := range channels[1:] {
		score := r.GetQualityScore(ch.Id)
		if score > bestScore {
			bestScore = score
			best = ch
		}
	}
	return best
}

// selectByPriorityWeight 优先级+权重选择（兼容现有逻辑）
func (r *SmartRouter) selectByPriorityWeight(channels []*model.Channel) *model.Channel {
	priorityGroups := make(map[int64][]*model.Channel)
	var maxPriority int64 = -1
	for _, ch := range channels {
		p := int64(0)
		if ch.Priority != nil {
			p = *ch.Priority
		}
		priorityGroups[p] = append(priorityGroups[p], ch)
		if p > maxPriority {
			maxPriority = p
		}
	}

	if maxPriority < 0 {
		return channels[0]
	}

	topGroup := priorityGroups[maxPriority]
	if len(topGroup) == 1 {
		return topGroup[0]
	}

	return weightedRandomSelect(topGroup)
}

// scoreChannelCost 计算渠道成本评分（越低越便宜）
func scoreChannelCost(ch *model.Channel) float64 {
	latencyFactor := float64(ch.ResponseTime) / 1000.0
	usageFactor := 0.0
	if ch.UsedQuota > 0 {
		usageFactor = 1.0 / (1.0 + float64(ch.UsedQuota)/1000000.0)
	}
	return latencyFactor + usageFactor*10
}

// weightedRandomSelect 加权随机选择
func weightedRandomSelect(channels []*model.Channel) *model.Channel {
	totalWeight := uint(0)
	for _, ch := range channels {
		if ch.Weight != nil {
			totalWeight += *ch.Weight
		} else {
			totalWeight += 1
		}
	}

	if totalWeight == 0 {
		return channels[0]
	}

	target := common.GetRandomInt(int(totalWeight))
	current := 0
	for _, ch := range channels {
		w := 1
		if ch.Weight != nil && *ch.Weight > 0 {
			w = int(*ch.Weight)
		}
		current += w
		if target < current {
			return ch
		}
	}

	return channels[len(channels)-1]
}

// ScoreAllChannels 对候选渠道列表评分排序
func (r *SmartRouter) ScoreAllChannels(ctx *RoutingContext, channels []*model.Channel) []ChannelScore {
	scores := make([]ChannelScore, 0, len(channels))
	for _, ch := range channels {
		score := ChannelScore{
			Channel:      ch,
			CostScore:    scoreChannelCost(ch),
			LatencyScore: float64(ch.ResponseTime) / 1000.0,
			QualityScore: r.GetQualityScore(ch.Id),
		}

		switch ctx.Strategy {
		case StrategyCostOptimized:
			score.FinalScore = score.CostScore
		case StrategyLatencyOptimized:
			score.FinalScore = score.LatencyScore
		case StrategyQualityFirst:
			score.FinalScore = 100 - score.QualityScore
		default:
			score.FinalScore = score.CostScore*0.4 + score.LatencyScore*0.3 + (100-score.QualityScore)*0.3
		}

		scores = append(scores, score)
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].FinalScore < scores[j].FinalScore
	})

	return scores
}
