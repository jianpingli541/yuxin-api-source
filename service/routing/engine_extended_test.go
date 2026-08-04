package routing

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func priorityPtr(p int64) *int64 { return &p }
func weightPtr(w uint) *uint     { return &w }

func mkChannel(id int, name string, priority int64, weight uint, responseTime int, usedQuota int64) *model.Channel {
	return &model.Channel{
		Id:           id,
		Name:         name,
		Priority:     priorityPtr(priority),
		Weight:       weightPtr(weight),
		ResponseTime: responseTime,
		UsedQuota:    usedQuota,
		Status:       1,
	}
}

func TestGetRouter_Singleton(t *testing.T) {
	r1 := GetRouter()
	r2 := GetRouter()
	if r1 != r2 {
		t.Fatal("GetRouter returned different instances")
	}
	if r1.defaultStrategy != StrategyPriorityWeight {
		t.Fatalf("default strategy: got %v, want priority_weight", r1.defaultStrategy)
	}
}

func TestSetStrategyAndDefaultStrategy(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyPriorityWeight, qualityScores: map[int]float64{}}
	r.SetStrategy(StrategyCostOptimized)
	if r.defaultStrategy != StrategyCostOptimized {
		t.Fatalf("strategy not updated: %v", r.defaultStrategy)
	}
}

func TestUpdateAndGetQualityScore(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyPriorityWeight, qualityScores: map[int]float64{}}
	if got := r.GetQualityScore(42); got != 75.0 {
		t.Fatalf("default quality: got %v want 75.0", got)
	}
	r.UpdateQualityScore(42, 88.5)
	if got := r.GetQualityScore(42); got != 88.5 {
		t.Fatalf("quality update: got %v want 88.5", got)
	}
	r.UpdateQualityScore(42, 50)
	if got := r.GetQualityScore(42); got != 50 {
		t.Fatalf("quality re-update: got %v want 50", got)
	}
}

func TestUpdateQualityScore_ConcurrentSafe(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyPriorityWeight, qualityScores: map[int]float64{}}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			r.UpdateQualityScore(i, float64(i))
		}(i)
		go func(i int) {
			defer wg.Done()
			_ = r.GetQualityScore(i)
		}(i)
	}
	wg.Wait()
	if v := r.GetQualityScore(7); v != 7 {
		t.Fatalf("concurrent update: got %v want 7", v)
	}
}

func TestSelectChannel_Empty(t *testing.T) {
	r := GetRouter()
	_, err := r.SelectChannel(&RoutingContext{Strategy: StrategyPriorityWeight}, nil)
	if err == nil {
		t.Fatal("expected error on empty channel list")
	}
}

func TestSelectChannel_Single(t *testing.T) {
	r := GetRouter()
	c := mkChannel(1, "only", 1, 10, 200, 0)
	got, err := r.SelectChannel(&RoutingContext{Strategy: StrategyPriorityWeight}, []*model.Channel{c})
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 1 {
		t.Fatalf("got %d want 1", got.Id)
	}
}

func TestSelectByPriorityWeight_HighestPriorityWins(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyPriorityWeight, qualityScores: map[int]float64{}}
	chs := []*model.Channel{
		mkChannel(1, "low", 1, 10, 100, 0),
		mkChannel(2, "high", 5, 10, 100, 0),
		mkChannel(3, "mid", 3, 10, 100, 0),
	}
	got := r.selectByPriorityWeight(chs)
	if got.Id != 2 {
		t.Fatalf("expected id=2 (highest priority), got %d", got.Id)
	}
}

func TestSelectByLatency_Fastest(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyLatencyOptimized, qualityScores: map[int]float64{}}
	chs := []*model.Channel{
		mkChannel(1, "slow", 1, 10, 1000, 0),
		mkChannel(2, "fast", 1, 10, 100, 0),
		mkChannel(3, "med", 1, 10, 500, 0),
	}
	got := r.selectByLatency(chs)
	if got.Id != 2 {
		t.Fatalf("expected id=2 (fastest), got %d", got.Id)
	}
}

func TestSelectByLatency_SkipsZero(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyLatencyOptimized, qualityScores: map[int]float64{}}
	chs := []*model.Channel{
		mkChannel(1, "zero", 1, 10, 0, 0),
		mkChannel(2, "slow", 1, 10, 800, 0),
	}
	got := r.selectByLatency(chs)
	if got.Id != 2 {
		t.Fatalf("expected id=2 (non-zero wins), got %d", got.Id)
	}
}

func TestSelectByQuality_HighestQuality(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyQualityFirst, qualityScores: map[int]float64{}}
	r.UpdateQualityScore(1, 60)
	r.UpdateQualityScore(2, 95)
	r.UpdateQualityScore(3, 80)
	chs := []*model.Channel{
		mkChannel(1, "low", 1, 10, 100, 0),
		mkChannel(2, "high", 1, 10, 100, 0),
		mkChannel(3, "mid", 1, 10, 100, 0),
	}
	got := r.selectByQuality(chs)
	if got.Id != 2 {
		t.Fatalf("expected id=2 (highest quality), got %d", got.Id)
	}
}

func TestSelectByCost_AccountsForUsageAndLatency(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyCostOptimized, qualityScores: map[int]float64{}}
	chs := []*model.Channel{
		mkChannel(1, "cheap_fast", 1, 10, 100, 0),
		mkChannel(2, "slow_used", 1, 10, 500, 500000),
	}
	got := r.selectByCost(chs)
	if got.Id != 1 {
		t.Fatalf("expected cheapest id=1, got %d", got.Id)
	}
}

func TestSelectChannel_DispatchByStrategy(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyPriorityWeight, qualityScores: map[int]float64{}}
	r.UpdateQualityScore(1, 50)
	r.UpdateQualityScore(2, 99)
	chs := []*model.Channel{
		mkChannel(1, "a", 1, 10, 100, 0),
		mkChannel(2, "b", 1, 10, 50, 0),
	}
	tests := []struct {
		strategy RoutingStrategy
		want     int
	}{
		{StrategyCostOptimized, 2}, // 零用量时成本分由延迟主导,50ms 渠道更便宜
		{StrategyLatencyOptimized, 2},
		{StrategyQualityFirst, 2},
	}
	for _, tt := range tests {
		got, err := r.SelectChannel(&RoutingContext{Strategy: tt.strategy}, chs)
		if err != nil {
			t.Fatal(err)
		}
		if got.Id != tt.want {
			t.Errorf("strategy %v: got id=%d want %d", tt.strategy, got.Id, tt.want)
		}
	}
}

func TestSelectChannel_FallsBackToDefault(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyLatencyOptimized, qualityScores: map[int]float64{}}
	chs := []*model.Channel{
		mkChannel(1, "slow", 1, 10, 1000, 0),
		mkChannel(2, "fast", 1, 10, 50, 0),
	}
	got, err := r.SelectChannel(&RoutingContext{Strategy: ""}, chs)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 2 {
		t.Fatalf("expected fallback to latency_optimized id=2, got %d", got.Id)
	}
}

func TestWeightedRandomSelect(t *testing.T) {
	chs := []*model.Channel{
		mkChannel(1, "a", 1, 90, 100, 0),
		mkChannel(2, "b", 1, 10, 100, 0),
	}
	count := map[int]int{}
	for i := 0; i < 200; i++ {
		got := weightedRandomSelect(chs)
		count[got.Id]++
	}
	if count[1] < count[2]*3 {
		t.Fatalf("weighted select heavily biased wrong way: %+v", count)
	}
}

func TestWeightedRandomSelect_ZeroWeightFallsBack(t *testing.T) {
	chs := []*model.Channel{
		mkChannel(1, "a", 1, 0, 100, 0),
	}
	got := weightedRandomSelect(chs)
	if got.Id != 1 {
		t.Fatalf("zero weight single channel: got %d want 1", got.Id)
	}
}

func TestScoreChannelCost_ZeroUsage(t *testing.T) {
	c := mkChannel(1, "x", 1, 1, 200, 0)
	s := scoreChannelCost(c)
	if s <= 0 {
		t.Fatalf("expected positive cost score, got %v", s)
	}
}

func TestScoreChannelCost_HeavyUsage(t *testing.T) {
	c := mkChannel(1, "x", 1, 1, 100, 100000000)
	s := scoreChannelCost(c)
	if s <= 0 {
		t.Fatalf("expected positive cost score, got %v", s)
	}
}

func TestScoreAllChannels_CostStrategy(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyCostOptimized, qualityScores: map[int]float64{}}
	r.UpdateQualityScore(1, 50)
	r.UpdateQualityScore(2, 80)
	chs := []*model.Channel{
		mkChannel(1, "x", 1, 10, 100, 0),
		mkChannel(2, "y", 1, 10, 500, 0),
	}
	scores := r.ScoreAllChannels(&RoutingContext{Strategy: StrategyCostOptimized}, chs)
	if len(scores) != 2 {
		t.Fatalf("got %d scores", len(scores))
	}
	if scores[0].FinalScore > scores[1].FinalScore {
		t.Fatalf("cost strategy not sorted ascending: %v vs %v", scores[0].FinalScore, scores[1].FinalScore)
	}
}

func TestScoreAllChannels_QualityStrategy(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyQualityFirst, qualityScores: map[int]float64{}}
	r.UpdateQualityScore(1, 50)
	r.UpdateQualityScore(2, 90)
	chs := []*model.Channel{mkChannel(1, "x", 1, 10, 100, 0), mkChannel(2, "y", 1, 10, 100, 0)}
	scores := r.ScoreAllChannels(&RoutingContext{Strategy: StrategyQualityFirst}, chs)
	if len(scores) != 2 {
		t.Fatalf("expected 2 scores")
	}
	if scores[0].FinalScore > scores[1].FinalScore {
		t.Fatalf("quality strategy should still sort ascending by finalscore (100-quality)")
	}
}

func TestScoreAllChannels_DefaultStrategyWeighted(t *testing.T) {
	r := &SmartRouter{defaultStrategy: StrategyPriorityWeight, qualityScores: map[int]float64{}}
	r.UpdateQualityScore(1, 50)
	r.UpdateQualityScore(2, 50)
	chs := []*model.Channel{mkChannel(1, "x", 1, 10, 100, 0), mkChannel(2, "y", 1, 10, 100, 0)}
	scores := r.ScoreAllChannels(&RoutingContext{Strategy: StrategyPriorityWeight}, chs)
	if len(scores) != 2 {
		t.Fatalf("expected 2 scores")
	}
}

func TestGetDefaultConfig_Default(t *testing.T) {
	cfg := GetDefaultConfig()
	if cfg.Strategy != StrategyPriorityWeight {
		t.Fatalf("default cfg strategy: got %v", cfg.Strategy)
	}
}

func TestSetDefaultConfig_AndRead(t *testing.T) {
	SetDefaultConfig(Config{Strategy: StrategyCostOptimized, FallbackChain: []int{1, 2, 3}})
	got := GetDefaultConfig()
	if got.Strategy != StrategyCostOptimized {
		t.Fatalf("set/get mismatch: %v", got.Strategy)
	}
	if len(got.FallbackChain) != 3 {
		t.Fatalf("fallback chain len: %d", len(got.FallbackChain))
	}
	SetDefaultConfig(Config{Strategy: StrategyPriorityWeight})
}