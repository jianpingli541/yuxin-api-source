package routing

import (
	"testing"
)

// Smoke test: GetDefaultConfig returns StrategyPriorityWeight (valid default)
func TestGetDefaultConfig(t *testing.T) {
	cfg := GetDefaultConfig()
	if cfg.Strategy == "" {
		t.Fatal("default strategy must not be empty")
	}
	// Valid strategies: priority/cost/latency/quality
	valid := map[RoutingStrategy]bool{
		StrategyPriorityWeight: true,
		StrategyCostOptimized: true,
		StrategyLatencyOptimized: true,
		StrategyQualityFirst: true,
	}
	if !valid[cfg.Strategy] {
		t.Errorf("default strategy %q is not in valid set", cfg.Strategy)
	}
}

// Smoke test: SetDefaultConfig + GetDefaultConfig round-trip
func TestSetDefaultConfig(t *testing.T) {
	newCfg := Config{
		Strategy:      StrategyCostOptimized,
		FallbackChain: []int{1, 2, 3},
	}
	SetDefaultConfig(newCfg)
	got := GetDefaultConfig()
	if got.Strategy != StrategyCostOptimized {
		t.Errorf("strategy = %v, want %v", got.Strategy, StrategyCostOptimized)
	}
	if len(got.FallbackChain) != 3 {
		t.Errorf("fallback chain length = %d, want 3", len(got.FallbackChain))
	}
}
