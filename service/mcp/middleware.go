package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// MCPMiddleware MCP 协议中间件
// 拦截 OpenAI 兼容的请求，注入 MCP 工具，并处理工具调用
func MCPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否是聊天补全请求
		if !strings.HasSuffix(c.Request.URL.Path, "/v1/chat/completions") {
			c.Next()
			return
		}

		// 检查是否启用 MCP
		registry := GetRegistry()
		if !registry.HasTools() {
			c.Next()
			return
		}

		// 检查是否跳过 MCP（通过 Header）
		if c.GetHeader("X-MCP-Bypass") == "true" {
			c.Next()
			return
		}

		// 读取请求体
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			common.SysLog("[MCP] 读取请求体失败: " + err.Error())
			c.Next()
			return
		}

		// 解析请求体
		var body map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		// 检查是否为流式请求
		isStream, _ := body["stream"].(bool)

		// 注入 MCP 工具
		if err := registry.InjectToolsIntoRequest(body); err != nil {
			common.SysLog("[MCP] 注入工具失败: " + err.Error())
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		// 重新编码请求体
		newBodyBytes, err := json.Marshal(body)
		if err != nil {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			return
		}

		// 替换请求体
		c.Request.Body = io.NopCloser(bytes.NewBuffer(newBodyBytes))
		c.Request.ContentLength = int64(len(newBodyBytes))

		// 存储原始请求信息，供后续处理
		c.Set("mcp_original_body", body)
		c.Set("mcp_is_stream", isStream)

		// 继续处理请求
		c.Next()

		// 如果是流式请求，需要特殊处理响应
		if isStream {
			handleStreamResponse(c, registry)
		} else {
			handleNonStreamResponse(c, registry)
		}
	}
}

// handleNonStreamResponse 处理非流式响应
func handleNonStreamResponse(c *gin.Context, registry *Registry) {
	// 检查是否有 tool_calls
	if c.Writer.Status() != http.StatusOK {
		return
	}

	// 这里可以实现工具调用的自动执行
	// 当前版本仅注入工具定义，工具调用由客户端处理
	common.SysLog("[MCP] 已注入工具到非流式响应")
}

// handleStreamResponse 处理流式响应
func handleStreamResponse(c *gin.Context, registry *Registry) {
	// 流式响应处理更复杂，需要解析 SSE 数据
	// 当前版本仅注入工具定义
	common.SysLog("[MCP] 已注入工具到流式响应")
}

// MCPEnabledMiddleware 检查 MCP 是否启用的中间件
func MCPEnabledMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		registry := GetRegistry()
		c.Set("mcp_enabled", registry.HasTools())
		c.Next()
	}
}
