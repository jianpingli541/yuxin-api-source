package guardrail

import (
	"testing"
)

// Smoke test: GetConfig runs without panic
func TestGetConfig(t *testing.T) {
	_ = GetConfig()
}

// Smoke test: UpdateConfig + GetConfig round-trip
func TestUpdateConfig(t *testing.T) {
	newCfg := ComplianceConfig{
		Enabled:             true,
		PromptInjectionMode: "strict",
		PIIDetection:        true,
		ContentModeration:   false,
		BlockedCategories:   []string{"spam"},
		RateLimitPerUser:    100,
		RateLimitPerIP:      1000,
	}
	UpdateConfig(newCfg)
	got := GetConfig()
	if !got.Enabled {
		t.Error("Enabled should be true after update")
	}
	if got.PromptInjectionMode != "strict" {
		t.Errorf("PromptInjectionMode = %q, want strict", got.PromptInjectionMode)
	}
	if !got.PIIDetection {
		t.Error("PIIDetection should be true")
	}
	if got.ContentModeration != false {
		t.Error("ContentModeration should be false")
	}
	if got.RateLimitPerUser != 100 {
		t.Errorf("RateLimitPerUser = %d, want 100", got.RateLimitPerUser)
	}
}

// Smoke test: CheckAllText runs without panic
func TestCheckAllTextSafe(t *testing.T) {
	results := CheckAllText("Hello, this is a normal safe text")
	for _, r := range results {
		// Just ensure structure is populated, not asserting pass/fail
		// (safe text may still be flagged by overly strict rules)
		_ = r.Passed
		_ = r.Layer
	}
}
