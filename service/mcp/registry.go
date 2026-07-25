package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

// ToolDefinition MCP 工具定义
type ToolDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolResult MCP 工具执行结果
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// ToolHandler 工具处理函数类型
type ToolHandler func(params map[string]interface{}) (*ToolResult, error)

// MCPServer MCP 服务器实例
type MCPServer struct {
	Name    string
	BaseURL string
	tools   []ToolDefinition
	handler map[string]ToolHandler
}

// Registry MCP 工具注册表
type Registry struct {
	mu      sync.RWMutex
	servers map[string]*MCPServer
	tools   map[string]ToolHandler // tool_name -> handler
	defs    map[string]ToolDefinition
}

var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetRegistry 获取全局 MCP 注册表
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = &Registry{
			servers: make(map[string]*MCPServer),
			tools:   make(map[string]ToolHandler),
			defs:    make(map[string]ToolDefinition),
		}
	})
	return globalRegistry
}

// RegisterTool 注册一个 MCP 工具
func (r *Registry) RegisterTool(name string, desc string, schema map[string]interface{}, handler ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defs[name] = ToolDefinition{
		Name:        name,
		Description: desc,
		InputSchema: schema,
	}
	r.tools[name] = handler
}

// GetTools 获取所有已注册的工具
func (r *Registry) GetTools() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ToolDefinition, 0, len(r.defs))
	for _, def := range r.defs {
		tools = append(tools, def)
	}
	return tools
}

// ExecuteTool 执行工具
func (r *Registry) ExecuteTool(name string, params map[string]interface{}) (*ToolResult, error) {
	r.mu.RLock()
	handler, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return handler(params)
}

// HasTools 检查是否有工具已注册
func (r *Registry) HasTools() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.defs) > 0
}

// ToolToJSON 将工具定义转为 OpenAI function calling 格式
func ToolToJSON(tool ToolDefinition) map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.InputSchema,
		},
	}
}

// ToolsToOpenAIFormat 将所有工具转为 OpenAI function calling 格式
func (r *Registry) ToolsToOpenAIFormat() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(r.defs))
	for _, def := range r.defs {
		result = append(result, ToolToJSON(def))
	}
	return result
}

// InjectToolsIntoRequest 将 MCP 工具注入到 OpenAI 请求中
func (r *Registry) InjectToolsIntoRequest(body map[string]interface{}) error {
	if !r.HasTools() {
		return nil
	}

	// 检查请求是否已经包含 tools
	existingTools, hasTools := body["tools"].([]interface{})
	if hasTools && len(existingTools) > 0 {
		return nil // 不覆盖已有的 tools
	}

	// 注入工具
	openaiTools := r.ToolsToOpenAIFormat()
	tools := make([]interface{}, len(openaiTools))
	for i, t := range openaiTools {
		tools[i] = t
	}
	body["tools"] = tools

	// 设置 tool_choice 为 auto（如果未设置）
	if _, exists := body["tool_choice"]; !exists {
		body["tool_choice"] = "auto"
	}

	return nil
}

// HandleToolCall 处理 LLM 返回的 tool_call
func (r *Registry) HandleToolCall(toolName string, args json.RawMessage) (*ToolResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}

	return r.ExecuteTool(toolName, params)
}
