package canary

import (
	"testing"
)

// Smoke test: GetEngine returns singleton
func TestGetEngineSingleton(t *testing.T) {
	e1 := GetEngine()
	e2 := GetEngine()
	if e1 == nil {
		t.Fatal("engine should not be nil")
	}
	if e1 != e2 {
		t.Error("GetEngine should return same instance (singleton)")
	}
}
