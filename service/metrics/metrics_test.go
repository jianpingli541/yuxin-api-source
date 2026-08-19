package metrics

import (
	"testing"
)

// Smoke test: RecordCacheHit/Miss doesn't panic
func TestRecordCacheCounters(t *testing.T) {
	// These should not panic
	RecordCacheHit()
	RecordCacheHit()
	RecordCacheMiss()

	// Verify GlobalMetrics is initialized
	if GlobalMetrics == nil {
		t.Fatal("GlobalMetrics should be initialized")
	}
	if GlobalMetrics.CacheHits.Load() < 2 {
		t.Errorf("CacheHits = %d, want >= 2", GlobalMetrics.CacheHits.Load())
	}
	if GlobalMetrics.CacheMisses.Load() < 1 {
		t.Errorf("CacheMisses = %d, want >= 1", GlobalMetrics.CacheMisses.Load())
	}
}

// Smoke test: GetMetricsSummary returns non-nil map
func TestGetMetricsSummary(t *testing.T) {
	summary := GetMetricsSummary()
	if summary == nil {
		t.Fatal("summary should not be nil")
	}
}
