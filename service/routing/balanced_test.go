package routing

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

func chWith(id int, chType int, respTime int) *model.Channel {
	return &model.Channel{Id: id, Type: chType, ResponseTime: respTime}
}

// 速度维度：同质量下选延迟更低者
func TestBalancedPrefersLowerLatency(t *testing.T) {
	r := GetRouter()
	fast := chWith(1, constant.ChannelTypeOpenAI, 1000)
	slow := chWith(2, constant.ChannelTypeOpenAI, 8000)
	got := r.selectByBalanced([]*model.Channel{slow, fast}, 0.5, 0.5)
	if got.Id != 1 {
		t.Fatalf("expected fast channel #1, got #%d", got.Id)
	}
}

// 字节倾斜：同配置下字节系渠道得分反超
func TestBalancedDoubaoBoostWins(t *testing.T) {
	r := GetRouter()
	plain := chWith(1, constant.ChannelTypeOpenAI, 2000)
	doubao := chWith(2, constant.ChannelTypeVolcEngine, 2000)
	got := r.selectByBalanced([]*model.Channel{plain, doubao}, 0.5, 0.5)
	if got.Id != 2 {
		t.Fatalf("expected doubao channel #2 with boost, got #%d", got.Id)
	}
}

// 极端速度偏好下，倾斜不足以弥补巨大延迟差
func TestBalancedSpeedBiasBeatsBoost(t *testing.T) {
	r := GetRouter()
	fastPlain := chWith(1, constant.ChannelTypeOpenAI, 500)
	slowDoubao := chWith(2, constant.ChannelTypeVolcEngine, 20000)
	got := r.selectByBalanced([]*model.Channel{slowDoubao, fastPlain}, 0.2, 0.8)
	if got.Id != 1 {
		t.Fatalf("expected fast channel #1 under speed bias, got #%d", got.Id)
	}
}

// 策略分发：balanced 走 selectByBalanced
func TestSelectChannelBalancedDispatch(t *testing.T) {
	r := GetRouter()
	channels := []*model.Channel{
		chWith(1, constant.ChannelTypeOpenAI, 1000),
		chWith(2, constant.ChannelTypeOpenAI, 9000),
	}
	got, err := r.SelectChannel(&RoutingContext{Strategy: StrategyBalanced}, channels)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 1 {
		t.Fatalf("expected #1, got #%d", got.Id)
	}
}

// X-Yuxin-Route 权重映射
func TestBalancedWeightsMapping(t *testing.T) {
	wq, ws := balancedWeights(&RoutingContext{RouteBias: "quality"})
	if wq != 0.8 || ws != 0.2 {
		t.Fatalf("quality bias: got %.1f/%.1f", wq, ws)
	}
	wq, ws = balancedWeights(&RoutingContext{RouteBias: "speed"})
	if wq != 0.2 || ws != 0.8 {
		t.Fatalf("speed bias: got %.1f/%.1f", wq, ws)
	}
	wq, ws = balancedWeights(&RoutingContext{})
	if wq != 0.5 || ws != 0.5 {
		t.Fatalf("default: got %.1f/%.1f", wq, ws)
	}
}
