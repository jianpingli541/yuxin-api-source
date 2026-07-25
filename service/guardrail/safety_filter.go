package guardrail

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// SafetyCheckResult 安全检查结果
type SafetyCheckResult struct {
	Passed   bool     `json:"passed"`
	Categories []string `json:"categories"`
	Reason   string   `json:"reason"`
}

// GuardrailConfig 内容过滤配置
type GuardrailConfig struct {
	Enabled        bool     `json:"enabled"`
	BlockCategories []string `json:"block_categories"`
	// 自定义敏感词库
	CustomBlocklist []string `json:"custom_blocklist"`
}

// defaultGuardrailConfig 默认配置
var defaultConfig = GuardrailConfig{
	Enabled: false,
	BlockCategories: []string{
		"hate",
		"harassment",
		"self-harm",
		"sexual",
		"violence",
	},
	CustomBlocklist: []string{},
}

// CheckContent 检查内容是否安全
func CheckContent(content string) *SafetyCheckResult {
	if !defaultConfig.Enabled {
		return &SafetyCheckResult{Passed: true}
	}

	lower := strings.ToLower(content)
	var matched []string

	// 自定义敏感词检查
	for _, word := range defaultConfig.CustomBlocklist {
		if strings.Contains(lower, strings.ToLower(word)) {
			matched = append(matched, "custom:"+word)
		}
	}

	// 类别检查（基于关键词匹配）
	categoryKeywords := map[string][]string{
		"hate":        {"种族歧视", "民族仇恨", "种族灭绝"},
		"violence":    {"暴力", "恐怖袭击", "爆炸物制作"},
		"self-harm":   {"自杀", "自残", "伤害自己"},
		"sexual":      {"色情", "性剥削"},
		"harassment":  {"骚扰", "跟踪", "威胁"},
	}

	for _, category := range defaultConfig.BlockCategories {
		keywords, ok := categoryKeywords[category]
		if !ok {
			continue
		}
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				matched = append(matched, category)
				break
			}
		}
	}

	if len(matched) > 0 {
		return &SafetyCheckResult{
			Passed:     false,
			Categories: matched,
			Reason:     "内容触发安全策略: " + strings.Join(matched, ", "),
		}
	}

	return &SafetyCheckResult{Passed: true}
}

// GuardrailMiddleware Gin 中间件
func GuardrailMiddleware() func(c *http.Request) bool {
	return func(r *http.Request) bool {
		if r.Method != "POST" {
			return true
		}

		if r.Body == nil {
			return true
		}

		// 读取请求体
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return true
		}

		// 检查 messages 字段
		if msgs, ok := body["messages"].([]interface{}); ok {
			for _, msg := range msgs {
				if m, ok := msg.(map[string]interface{}); ok {
					if content, ok := m["content"].(string); ok {
						result := CheckContent(content)
						if !result.Passed {
							common.SysLog("Guardrail blocked: " + result.Reason)
							return false
						}
					}
				}
			}
		}

		// 检查 prompt 字段
		if prompt, ok := body["prompt"].(string); ok {
			result := CheckContent(prompt)
			if !result.Passed {
				common.SysLog("Guardrail blocked: " + result.Reason)
				return false
			}
		}

		return true
	}
}

// UpdateConfig 更新过滤配置（从系统配置读取）
func UpdateConfig(enabled bool, categories []string, blocklist []string) {
	defaultConfig.Enabled = enabled
	defaultConfig.BlockCategories = categories
	defaultConfig.CustomBlocklist = blocklist
}

// GetConfig 获取当前配置
func GetConfig() GuardrailConfig {
	return defaultConfig
}
