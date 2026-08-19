package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// freshRegistry 返回独立的 Registry 实例(避免全局注册表污染)
func freshRegistry() *Registry {
	return &Registry{
		servers: make(map[string]*MCPServer),
		tools:   make(map[string]ToolHandler),
		defs:    make(map[string]ToolDefinition),
	}
}

func TestGetRegistry_Singleton(t *testing.T) {
	r1 := GetRegistry()
	r2 := GetRegistry()
	if r1 != r2 {
		t.Fatal("GetRegistry should return same instance")
	}
}

func TestRegisterTool_AndGetTools(t *testing.T) {
	r := freshRegistry()
	r.RegisterTool("echo", "echoes input", map[string]interface{}{"type": "object"}, func(params map[string]interface{}) (*ToolResult, error) {
		return &ToolResult{Content: "ok"}, nil
	})
	tools := r.GetTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "echo" {
		t.Fatalf("tool name: %s", tools[0].Name)
	}
	if tools[0].Description != "echoes input" {
		t.Fatalf("tool desc: %s", tools[0].Description)
	}
}

func TestRegisterTool_Overwrite(t *testing.T) {
	r := freshRegistry()
	h1 := func(params map[string]interface{}) (*ToolResult, error) { return &ToolResult{Content: "v1"}, nil }
	h2 := func(params map[string]interface{}) (*ToolResult, error) { return &ToolResult{Content: "v2"}, nil }
	r.RegisterTool("t", "v1", nil, h1)
	r.RegisterTool("t", "v2", nil, h2)
	if len(r.GetTools()) != 1 {
		t.Fatalf("re-register should overwrite, got %d tools", len(r.GetTools()))
	}
	res, err := r.ExecuteTool("t", nil)
	if err != nil || res.Content != "v2" {
		t.Fatalf("expected v2 handler, got %+v err=%v", res, err)
	}
}

func TestExecuteTool_NotFound(t *testing.T) {
	r := freshRegistry()
	_, err := r.ExecuteTool("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "tool not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteTool_HandlerError(t *testing.T) {
	r := freshRegistry()
	r.RegisterTool("fail", "always fails", nil, func(params map[string]interface{}) (*ToolResult, error) {
		return nil, errToolFailed
	})
	_, err := r.ExecuteTool("fail", nil)
	if err == nil || err != errToolFailed {
		t.Fatalf("expected tool error to propagate, got %v", err)
	}
}

func TestHasTools(t *testing.T) {
	r := freshRegistry()
	if r.HasTools() {
		t.Fatal("fresh registry should have no tools")
	}
	r.RegisterTool("t", "", nil, func(params map[string]interface{}) (*ToolResult, error) { return nil, nil })
	if !r.HasTools() {
		t.Fatal("registry should have tools after registration")
	}
}

func TestToolToJSON_Format(t *testing.T) {
	def := ToolDefinition{
		Name:        "calc",
		Description: "calculate",
		InputSchema: map[string]interface{}{"type": "object"},
	}
	got := ToolToJSON(def)
	if got["type"] != "function" {
		t.Fatalf("type: %v", got["type"])
	}
	fn, ok := got["function"].(map[string]interface{})
	if !ok {
		t.Fatal("missing function object")
	}
	if fn["name"] != "calc" || fn["description"] != "calculate" {
		t.Fatalf("function fields: %+v", fn)
	}
	if _, ok := fn["parameters"]; !ok {
		t.Fatal("missing parameters")
	}
}

func TestToolsToOpenAIFormat(t *testing.T) {
	r := freshRegistry()
	r.RegisterTool("a", "tool a", nil, func(p map[string]interface{}) (*ToolResult, error) { return nil, nil })
	r.RegisterTool("b", "tool b", nil, func(p map[string]interface{}) (*ToolResult, error) { return nil, nil })
	out := r.ToolsToOpenAIFormat()
	if len(out) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(out))
	}
	for _, tool := range out {
		if tool["type"] != "function" {
			t.Fatalf("bad format: %+v", tool)
		}
	}
}

func TestInjectToolsIntoRequest_NoTools_NoOp(t *testing.T) {
	r := freshRegistry()
	body := map[string]interface{}{"model": "gpt-4"}
	if err := r.InjectToolsIntoRequest(body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["tools"]; ok {
		t.Fatal("empty registry should not inject tools")
	}
}

func TestInjectToolsIntoRequest_Injects(t *testing.T) {
	r := freshRegistry()
	r.RegisterTool("search", "search tool", nil, func(p map[string]interface{}) (*ToolResult, error) { return nil, nil })
	body := map[string]interface{}{"model": "gpt-4"}
	if err := r.InjectToolsIntoRequest(body); err != nil {
		t.Fatal(err)
	}
	tools, ok := body["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatalf("tools not injected: %+v", body["tools"])
	}
	if body["tool_choice"] != "auto" {
		t.Fatalf("tool_choice should default to auto, got %v", body["tool_choice"])
	}
}

func TestInjectToolsIntoRequest_ExistingToolsNotOverwritten(t *testing.T) {
	r := freshRegistry()
	r.RegisterTool("search", "search tool", nil, func(p map[string]interface{}) (*ToolResult, error) { return nil, nil })
	existing := []interface{}{map[string]interface{}{"type": "function"}}
	body := map[string]interface{}{"model": "gpt-4", "tools": existing}
	if err := r.InjectToolsIntoRequest(body); err != nil {
		t.Fatal(err)
	}
	tools := body["tools"].([]interface{})
	if len(tools) != 1 {
		t.Fatalf("existing tools should not be overwritten, got %d", len(tools))
	}
}

func TestInjectToolsIntoRequest_ExistingToolChoicePreserved(t *testing.T) {
	r := freshRegistry()
	r.RegisterTool("search", "search tool", nil, func(p map[string]interface{}) (*ToolResult, error) { return nil, nil })
	body := map[string]interface{}{"model": "gpt-4", "tool_choice": "none"}
	if err := r.InjectToolsIntoRequest(body); err != nil {
		t.Fatal(err)
	}
	if body["tool_choice"] != "none" {
		t.Fatalf("existing tool_choice should be preserved, got %v", body["tool_choice"])
	}
}

func TestHandleToolCall_ValidArgs(t *testing.T) {
	r := freshRegistry()
	r.RegisterTool("echo", "echo", nil, func(params map[string]interface{}) (*ToolResult, error) {
		return &ToolResult{Content: params["msg"].(string)}, nil
	})
	res, err := r.HandleToolCall("echo", json.RawMessage(`{"msg":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != "hello" {
		t.Fatalf("got %s", res.Content)
	}
}

func TestHandleToolCall_InvalidJSON(t *testing.T) {
	r := freshRegistry()
	r.RegisterTool("echo", "echo", nil, func(params map[string]interface{}) (*ToolResult, error) {
		return &ToolResult{}, nil
	})
	_, err := r.HandleToolCall("echo", json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON args")
	}
}

func TestHandleToolCall_UnknownTool(t *testing.T) {
	r := freshRegistry()
	_, err := r.HandleToolCall("ghost", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

// ---- 内置工具测试 ----

func TestBuiltinWebSearch(t *testing.T) {
	res, err := handleWebSearch(map[string]interface{}{"query": "golang"})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", res.Content)
	}
	if !strings.Contains(res.Content, "golang") {
		t.Fatalf("result should contain query, got %s", res.Content)
	}
}

func TestBuiltinWebSearch_MissingQuery(t *testing.T) {
	res, _ := handleWebSearch(map[string]interface{}{})
	if !res.IsError {
		t.Fatal("missing query should be error")
	}
	if !strings.Contains(res.Content, "missing query") {
		t.Fatalf("unexpected message: %s", res.Content)
	}
}

func TestBuiltinGetTime_DefaultTZ(t *testing.T) {
	res, err := handleGetTime(map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Content, "UTC") {
		t.Fatalf("default timezone should be UTC, got %s", res.Content)
	}
}

func TestBuiltinGetTime_CustomTZ(t *testing.T) {
	res, _ := handleGetTime(map[string]interface{}{"timezone": "Asia/Shanghai"})
	if !strings.Contains(res.Content, "Asia/Shanghai") {
		t.Fatalf("got %s", res.Content)
	}
}

func TestBuiltinCalculator_ValidExpression(t *testing.T) {
	res, err := handleCalculator(map[string]interface{}{"expression": "2 + 3 * 4"})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("valid expression rejected: %s", res.Content)
	}
}

func TestBuiltinCalculator_RejectsInjection(t *testing.T) {
	cases := []string{"rm -rf /", "system('ls')", "<script>", "eval(1)"}
	for _, expr := range cases {
		res, _ := handleCalculator(map[string]interface{}{"expression": expr})
		if !res.IsError {
			t.Fatalf("expression %q should be rejected", expr)
		}
	}
}

func TestBuiltinCalculator_MissingExpression(t *testing.T) {
	res, _ := handleCalculator(map[string]interface{}{})
	if !res.IsError {
		t.Fatal("missing expression should be error")
	}
}

func TestRegisterBuiltinTools_RegistersThree(t *testing.T) {
	r := GetRegistry()
	RegisterBuiltinTools()
	tools := r.GetTools()
	names := map[string]bool{}
	for _, td := range tools {
		names[td.Name] = true
	}
	for _, want := range []string{"web_search", "get_current_time", "calculator"} {
		if !names[want] {
			t.Errorf("builtin tool %q not registered", want)
		}
	}
}

// ---- HTTP handler 测试 ----

func ginTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestMCPToolsListHandler(t *testing.T) {
	RegisterBuiltinTools()
	r := ginTestRouter()
	r.GET("/api/mcp/tools", MCPToolsListHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/mcp/tools", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "web_search") {
		t.Fatalf("missing web_search in response: %s", w.Body.String()[:200])
	}
}

func TestMCPExecuteHandler_Success(t *testing.T) {
	RegisterBuiltinTools()
	r := ginTestRouter()
	r.POST("/api/mcp/execute", MCPExecuteHandler)
	w := httptest.NewRecorder()
	body := `{"tool_name":"get_current_time","params":{"timezone":"UTC"}}`
	req, _ := http.NewRequest("POST", "/api/mcp/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
	}
}

func TestMCPExecuteHandler_UnknownTool(t *testing.T) {
	r := ginTestRouter()
	r.POST("/api/mcp/execute", MCPExecuteHandler)
	w := httptest.NewRecorder()
	body := `{"tool_name":"ghost_tool","params":{}}`
	req, _ := http.NewRequest("POST", "/api/mcp/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestMCPExecuteHandler_InvalidBody(t *testing.T) {
	r := ginTestRouter()
	r.POST("/api/mcp/execute", MCPExecuteHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/mcp/execute", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestMCPConfigHandler_GetStatus(t *testing.T) {
	r := ginTestRouter()
	r.GET("/api/mcp/config", MCPConfigHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/mcp/config", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestMCPConfigHandler_Disable(t *testing.T) {
	r := ginTestRouter()
	r.POST("/api/mcp/config", MCPConfigHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/mcp/config", strings.NewReader(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if GetRegistry().HasTools() {
		t.Fatal("registry should be empty after disable")
	}
}

func TestMCPConfigHandler_Enable(t *testing.T) {
	r := ginTestRouter()
	r.POST("/api/mcp/config", MCPConfigHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/mcp/config", strings.NewReader(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if !GetRegistry().HasTools() {
		t.Fatal("registry should have builtin tools after enable")
	}
}

func TestMCPConfigHandler_InvalidBody(t *testing.T) {
	r := ginTestRouter()
	r.POST("/api/mcp/config", MCPConfigHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/mcp/config", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d", w.Code)
	}
}

var errToolFailed = errForTest("tool failed")

type errForTest string

func (e errForTest) Error() string { return string(e) }
