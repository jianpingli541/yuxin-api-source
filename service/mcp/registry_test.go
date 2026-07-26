package mcp

import (
	"testing"
)

// Smoke test: registry singleton works
func TestGetRegistrySingleton(t *testing.T) {
	r1 := GetRegistry()
	r2 := GetRegistry()
	if r1 == nil {
		t.Fatal("registry should not be nil")
	}
	if r1 != r2 {
		t.Error("GetRegistry should return same instance (singleton)")
	}
}

// Smoke test: ToolToJSON produces OpenAI function calling format
func TestToolToJSON(t *testing.T) {
	tool := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
	}
	m := ToolToJSON(tool)
	if m["type"] != "function" {
		t.Errorf("top-level type = %v, want function", m["type"])
	}
	fn, ok := m["function"].(map[string]interface{})
	if !ok {
		t.Fatal("function field should be a map")
	}
	if fn["name"] != "test_tool" {
		t.Errorf("function.name = %v, want test_tool", fn["name"])
	}
	if fn["description"] != "A test tool" {
		t.Errorf("function.description = %v", fn["description"])
	}
}
