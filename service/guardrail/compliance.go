package guardrail

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 安全检查层级
type SecurityLayer int

const (
	LayerPromptInjection SecurityLayer = iota
	LayerPIIDetection
	LayerContentModeration
	LayerRateLimit
)

// 检查结果
type CheckResult struct {
	Passed  bool     `json:"passed"`
	Layer   string   `json:"layer"`
	Reason  string   `json:"reason,omitempty"`
	Details []string `json:"details,omitempty"`
}

// 合规配置
type ComplianceConfig struct {
	Enabled             bool     `json:"enabled"`
	PromptInjectionMode string   `json:"prompt_injection_mode"` // strict/medium/loose
	PIIDetection        bool     `json:"pii_detection"`
	ContentModeration   bool     `json:"content_moderation"`
	BlockedCategories   []string `json:"blocked_categories"`
	RateLimitPerUser    int      `json:"rate_limit_per_user"` // 每分钟
	RateLimitPerIP      int      `json:"rate_limit_per_ip"`   // 每分钟
}

var (
	config     *ComplianceConfig
	configLock sync.RWMutex

	// 敏感信息正则
	piiPatterns = map[string]*regexp.Regexp{
		"email":    regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		"phone":    regexp.MustCompile(`\b(1[3-9]\d{9}|0\d{2,3}-?\d{7,8})\b`),
		"idcard":   regexp.MustCompile(`\b\d{17}[\dXx]\b`),
		"bankcard": regexp.MustCompile(`\b\d{16,19}\b`),
		"apikey":   regexp.MustCompile(`\b(sk-|key-|api[_-]?key[_-]?)[A-Za-z0-9]{20,}\b`),
		"password": regexp.MustCompile(`(?i)(password|passwd|pwd)[:\s=]+\S+`),
		"token":    regexp.MustCompile(`(?i)(bearer|token)[:\s=]+\S+`),
	}

	// Prompt注入模式
	injectionPatterns = map[string]*regexp.Regexp{
		"role_override":   regexp.MustCompile(`(?i)(ignore|forget|disregard).{0,20}(previous|all|the).{0,20}(instructions?|rules?|prompt)`),
		"system_prompt":   regexp.MustCompile(`(?i)(you\s+are|act\s+as|pretend\s+to\s+be|become)\s+(a|an)\s+`),
		"jailbreak":       regexp.MustCompile(`(?i)(DAN|jailbreak|break\s+free|no\s+restrictions|without\s+limits)`),
		"hypothetical":    regexp.MustCompile(`(?i)(hypothetically|in\s+theory|what\s+if|suppose)\s+`),
		"roleplay_bypass": regexp.MustCompile(`(?i)(roleplay|RP|角色扮演)\s+.*?(bypass|ignore|override)`),
		"code_injection":  regexp.MustCompile(`(?i)(eval|exec|system|shell_exec|passthru)\s*\(`),
		"sql_injection":   regexp.MustCompile(`(?i)(UNION\s+SELECT|DROP\s+TABLE|DELETE\s+FROM|INSERT\s+INTO|UPDATE\s+\w+\s+SET)`),
		"xss":             regexp.MustCompile(`<script[^>]*>.*?</script>`),
	}

	// 内容审核类别
	contentCategories = map[string][]string{
		"violence": {
			"暴力", "杀人", "袭击", "恐怖主义", "枪击", "爆炸", "自杀式袭击",
			"weapon", "attack", "terrorist", "massacre", "shoot", "bomb",
		},
		"self_harm": {
			"自杀", "自残", "割腕", "跳楼", "上吊", "服药自杀",
			"suicide", "self-harm", "cutting", "kill myself", "end my life",
		},
		"sexual": {
			"色情", "性爱", "裸体", "性行为", "嫖娼", "卖淫",
			"porn", "sex", "naked", "nude", "explicit", "adult content",
		},
		"illegal": {
			"毒品", "贩毒", "制毒", "冰毒", "大麻", "可卡因",
			"赌博", "洗钱", "走私", "行贿", "受贿",
			"drugs", "cocaine", "heroin", "meth", "gambling", "money laundering",
		},
		"hate_speech": {
			"种族歧视", "民族仇恨", "性别歧视", "歧视", "仇恨言论",
			"racist", "discrimination", "hate speech", "superiority",
		},
	}

	// 速率限制缓存
	rateLimitCache = make(map[string][]int64)
	rateLimitLock  sync.RWMutex
)

func init() {
	// 默认配置
	config = &ComplianceConfig{
		Enabled:             true,
		PromptInjectionMode: "medium",
		PIIDetection:        true,
		ContentModeration:   true,
		BlockedCategories:   []string{"violence", "self_harm", "sexual", "illegal", "hate_speech"},
		RateLimitPerUser:    60,
		RateLimitPerIP:      100,
	}
}

// GetConfig 获取配置
func GetConfig() ComplianceConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return *config
}

// UpdateConfig 更新配置
func UpdateConfig(cfg ComplianceConfig) {
	configLock.Lock()
	defer configLock.Unlock()
	config = &cfg
}

// CheckAll 执行所有安全检查
func CheckAll(request *http.Request, body []byte) []CheckResult {
	var results []CheckResult

	cfg := GetConfig()
	if !cfg.Enabled {
		return results
	}

	// 解析请求体
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return results
	}

	// 提取文本内容
	texts := extractTexts(data)

	// 层级1: Prompt注入检测
	if check, _ := checkPromptInjection(texts); !check.Passed {
		results = append(results, check)
	}

	// 层级2: PII检测
	if cfg.PIIDetection {
		if check, _ := checkPII(texts); !check.Passed {
			results = append(results, check)
		}
	}

	// 层级3: 内容审核
	if cfg.ContentModeration {
		if check, _ := checkContent(texts, cfg.BlockedCategories); !check.Passed {
			results = append(results, check)
		}
	}

	// 层级4: 速率限制
	if check, _ := checkRateLimit(request, cfg); !check.Passed {
		results = append(results, check)
	}

	return results
}

// CheckAllText 对纯文本执行安全检查（便捷方法）
func CheckAllText(text string) []CheckResult {
	var results []CheckResult

	cfg := GetConfig()
	if !cfg.Enabled {
		return results
	}

	texts := []string{text}

	if check, _ := checkPromptInjection(texts); !check.Passed {
		results = append(results, check)
	}

	if cfg.PIIDetection {
		if check, _ := checkPII(texts); !check.Passed {
			results = append(results, check)
		}
	}

	if cfg.ContentModeration {
		if check, _ := checkContent(texts, cfg.BlockedCategories); !check.Passed {
			results = append(results, check)
		}
	}

	return results
}

// extractTexts 从请求体中提取所有文本
func extractTexts(data map[string]interface{}) []string {
	var texts []string

	if messages, ok := data["messages"].([]interface{}); ok {
		for _, msg := range messages {
			if m, ok := msg.(map[string]interface{}); ok {
				if content, ok := m["content"].(string); ok {
					texts = append(texts, content)
				}
			}
		}
	}

	if prompt, ok := data["prompt"].(string); ok {
		texts = append(texts, prompt)
	}

	if input, ok := data["input"].(string); ok {
		texts = append(texts, input)
	}

	return texts
}

// checkPromptInjection 检测提示词注入
func checkPromptInjection(texts []string) (CheckResult, bool) {
	for _, text := range texts {
		for name, pattern := range injectionPatterns {
			if pattern.MatchString(text) {
				return CheckResult{
					Passed:  false,
					Layer:   "prompt_injection",
					Reason:  "检测到提示词注入尝试",
					Details: []string{fmt.Sprintf("匹配规则: %s", name)},
				}, false
			}
		}
	}
	return CheckResult{Passed: true, Layer: "prompt_injection"}, true
}

// checkPII 检测敏感信息
func checkPII(texts []string) (CheckResult, bool) {
	var details []string

	for _, text := range texts {
		for name, pattern := range piiPatterns {
			matches := pattern.FindAllString(text, -1)
			if len(matches) > 0 {
				details = append(details, fmt.Sprintf("%s: %d处", name, len(matches)))
			}
		}
	}

	if len(details) > 0 {
		return CheckResult{
			Passed:  false,
			Layer:   "pii_detection",
			Reason:  "检测到敏感信息",
			Details: details,
		}, false
	}

	return CheckResult{Passed: true, Layer: "pii_detection"}, true
}

// checkContent 内容审核
func checkContent(texts []string, blockedCategories []string) (CheckResult, bool) {
	for _, text := range texts {
		textLower := strings.ToLower(text)

		for _, category := range blockedCategories {
			keywords, ok := contentCategories[category]
			if !ok {
				continue
			}

			for _, keyword := range keywords {
				if strings.Contains(textLower, strings.ToLower(keyword)) {
					return CheckResult{
						Passed:  false,
						Layer:   "content_moderation",
						Reason:  fmt.Sprintf("内容违反 %s 类别政策", category),
						Details: []string{fmt.Sprintf("触发词: %s", keyword)},
					}, false
				}
			}
		}
	}

	return CheckResult{Passed: true, Layer: "content_moderation"}, true
}

// checkRateLimit 速率限制
func checkRateLimit(request *http.Request, cfg ComplianceConfig) (CheckResult, bool) {
	now := time.Now().Unix()

	// 获取用户标识和IP
	userID := request.Header.Get("Authorization")
	clientIP := request.RemoteAddr

	// 检查用户限制
	if userID != "" {
		if !checkLimit(userID, cfg.RateLimitPerUser, now) {
			return CheckResult{
				Passed: false,
				Layer:  "rate_limit",
				Reason: "用户请求频率超限",
			}, false
		}
	}

	// 检查IP限制
	if clientIP != "" {
		if !checkLimit(clientIP, cfg.RateLimitPerIP, now) {
			return CheckResult{
				Passed: false,
				Layer:  "rate_limit",
				Reason: "IP请求频率超限",
			}, false
		}
	}

	return CheckResult{Passed: true, Layer: "rate_limit"}, true
}

func checkLimit(key string, limit int, now int64) bool {
	rateLimitLock.Lock()
	defer rateLimitLock.Unlock()

	// 清理过期记录
	timestamps, ok := rateLimitCache[key]
	if !ok {
		timestamps = []int64{}
	}

	var valid []int64
	for _, ts := range timestamps {
		if now-ts < 60 { // 1分钟内
			valid = append(valid, ts)
		}
	}

	// 添加当前时间戳
	valid = append(valid, now)
	rateLimitCache[key] = valid

	return len(valid) <= limit
}

// ComplianceMiddleware Gin中间件
func ComplianceMiddleware() func(*http.Request) bool {
	return func(r *http.Request) bool {
		cfg := GetConfig()
		if !cfg.Enabled {
			return true
		}

		// 读取请求体（需要缓冲）
		body, err := readRequestBody(r)
		if err != nil {
			return true
		}

		results := CheckAll(r, body)
		for _, result := range results {
			if !result.Passed {
				fmt.Printf("[安全合规] 拦截: %s - %s\n", result.Layer, result.Reason)
				return false
			}
		}

		return true
	}
}

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return []byte{}, nil
	}

	body := make([]byte, r.ContentLength)
	_, err := r.Body.Read(body)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return body, nil
}
