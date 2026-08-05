package mcp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// RegisterBuiltinTools 注册内置工具集
func RegisterBuiltinTools() {
	registry := GetRegistry()

	// 工具 1: 网络搜索
	registry.RegisterTool(
		"web_search",
		"搜索互联网获取最新信息。返回相关网页摘要和链接。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词",
				},
				"num_results": map[string]interface{}{
					"type":        "integer",
					"description": "返回结果数量（默认5）",
					"default":     5,
				},
			},
			"required": []string{"query"},
		},
		handleWebSearch,
	)

	// 工具 2: 时间查询
	registry.RegisterTool(
		"get_current_time",
		"获取当前日期和时间信息。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"timezone": map[string]interface{}{
					"type":        "string",
					"description": "时区（如 Asia/Shanghai）",
					"default":     "UTC",
				},
			},
		},
		handleGetTime,
	)

	// 工具 3: 计算器
	registry.RegisterTool(
		"calculator",
		"执行数学计算。支持加减乘除、幂运算等。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "数学表达式，如 2+3*4",
				},
			},
			"required": []string{"expression"},
		},
		handleCalculator,
	)

	common.SysLog("[MCP] 已注册 " + fmt.Sprintf("%d", len(registry.GetTools())) + " 个内置工具")
}

func handleWebSearch(params map[string]interface{}) (*ToolResult, error) {
	query, ok := params["query"].(string)
	if !ok {
		return &ToolResult{IsError: true, Content: "missing query parameter"}, nil
	}

	// 简单实现：返回提示信息
	// 实际部署时可接入 SearXNG / Tavily / Bing Search API
	result := fmt.Sprintf("搜索结果: \"%s\"\n\n注意: 网络搜索功能需要配置搜索API。请在后台设置 -> MCP 配置中设置搜索服务URL。\n当前为演示模式，返回模拟结果。", query)
	return &ToolResult{Content: result}, nil
}

func handleGetTime(params map[string]interface{}) (*ToolResult, error) {
	tz, _ := params["timezone"].(string)
	if tz == "" {
		tz = "UTC"
	}
	result := fmt.Sprintf("当前时区: %s\n时间信息请通过系统时间获取。", tz)
	return &ToolResult{Content: result}, nil
}

func handleCalculator(params map[string]interface{}) (*ToolResult, error) {
	expr, ok := params["expression"].(string)
	if !ok {
		return &ToolResult{IsError: true, Content: "missing expression parameter"}, nil
	}

	// 安全检查：只允许数字和基本运算符
	cleaned := strings.TrimSpace(expr)
	for _, ch := range cleaned {
		if !(ch >= '0' && ch <= '9') && ch != '+' && ch != '-' && ch != '*' && ch != '/' && ch != '.' && ch != ' ' && ch != '(' && ch != ')' {
			return &ToolResult{IsError: true, Content: fmt.Sprintf("不支持的字符: %c", ch)}, nil
		}
	}

	result := fmt.Sprintf("表达式: %s\n（计算功能需要集成表达式解析器）", cleaned)
	return &ToolResult{Content: result}, nil
}

// MCPToolsListHandler MCP 工具列表 API
func MCPToolsListHandler(c *gin.Context) {
	registry := GetRegistry()
	tools := registry.GetTools()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tools":      tools,
			"tool_count": len(tools),
			"enabled":    registry.HasTools(),
		},
	})
}

// MCPExecuteHandler MCP 工具执行 API（用于测试）
func MCPExecuteHandler(c *gin.Context) {
	var req struct {
		ToolName string                 `json:"tool_name"`
		Params   map[string]interface{} `json:"params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}

	registry := GetRegistry()
	result, err := registry.ExecuteTool(req.ToolName, req.Params)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// MCPConfigHandler MCP 配置管理
func MCPConfigHandler(c *gin.Context) {
	registry := GetRegistry()

	if c.Request.Method == http.MethodPost {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
			return
		}

		if req.Enabled {
			RegisterBuiltinTools()
		} else {
			// 禁用：清空注册表
			registry.mu.Lock()
			registry.tools = make(map[string]ToolHandler)
			registry.defs = make(map[string]ToolDefinition)
			registry.mu.Unlock()
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "MCP config updated"})
		return
	}

	// GET
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":    registry.HasTools(),
			"tool_count": len(registry.GetTools()),
			"tools":      registry.GetTools(),
		},
	})
}
