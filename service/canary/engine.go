package canary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// CanaryTest Canary测试用例
type CanaryTest struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Category       string   `json:"category"` // reasoning/knowledge/code/creative
	Prompt         string   `json:"prompt"`
	ExpectedLength int      `json:"expected_length"` // 期望响应长度（字符数）
	Keywords       []string `json:"keywords"`        // 响应中应包含的关键词
}

// CanaryResult Canary测试结果
type CanaryResult struct {
	ChannelID    int     `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	ModelName    string  `json:"model_name"`
	TestID       string  `json:"test_id"`
	Score        float64 `json:"score"`          // 0-100
	Pass         bool    `json:"pass"`
	LatencyMs    int64   `json:"latency_ms"`
	ResponseLen  int     `json:"response_length"`
	KeywordsFound int    `json:"keywords_found"`
	Error        string  `json:"error,omitempty"`
	TestedAt     int64   `json:"tested_at"`
}

// ChannelHealth 渠道健康度汇总
type ChannelHealth struct {
	ChannelID    int     `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	ModelName    string  `json:"model_name"`
	Score        float64 `json:"score"`         // 加权平均分 0-100
	PassRate     float64 `json:"pass_rate"`     // 通过率 %
	AvgLatencyMs int64   `json:"avg_latency_ms"`
	LastTestAt   int64   `json:"last_test_at"`
	TotalTests   int     `json:"total_tests"`
	PassedTests  int     `json:"passed_tests"`
}

// 内置 Canary 测试用例集
var builtinTests = []CanaryTest{
	{
		ID:             "reasoning-001",
		Name:           "基础逻辑推理",
		Description:    "测试简单数学推理能力",
		Category:       "reasoning",
		Prompt:         "一个农场有鸡和兔共35只，脚共94只。请问鸡和兔各有多少只？请写出计算过程。",
		ExpectedLength: 100,
		Keywords:       []string{"鸡", "兔", "23", "12"},
	},
	{
		ID:             "knowledge-001",
		Name:           "历史知识准确性",
		Description:    "测试基本历史事实",
		Category:       "knowledge",
		Prompt:         "请简述秦始皇统一六国的历史意义，至少列出3点。",
		ExpectedLength: 150,
		Keywords:       []string{"统一", "文字", "度量衡", "秦"},
	},
	{
		ID:             "code-001",
		Name:           "代码生成能力",
		Description:    "测试Python代码生成",
		Category:       "code",
		Prompt:         "用Python写一个函数，实现二分查找算法，要求有注释。",
		ExpectedLength: 200,
		Keywords:       []string{"def", "binary_search", "mid", "return"},
	},
	{
		ID:             "creative-001",
		Name:           "创意写作",
		Description:    "测试创意文本生成",
		Category:       "creative",
		Prompt:         "请用3句话描述春天的景色，要求有诗意。",
		ExpectedLength: 50,
		Keywords:       []string{"春", "花"},
	},
	{
		ID:             "instruction-001",
		Name:           "指令遵循能力",
		Description:    "测试是否严格遵循格式要求",
		Category:       "reasoning",
		Prompt:         "请用JSON格式回答：中国的首都是哪里？人口多少？只输出JSON，不要其他文字。JSON字段为 capital 和 population。",
		ExpectedLength: 50,
		Keywords:       []string{"capital", "北京", "population"},
	},
}

// CanaryEngine Canary 引擎
type CanaryEngine struct {
	mu       sync.RWMutex
	results  map[int][]CanaryResult // channel_id -> results
	tests    []CanaryTest
	enabled  bool
}

var (
	globalEngine *CanaryEngine
	engineOnce   sync.Once
)

// GetEngine 获取全局引擎实例
func GetEngine() *CanaryEngine {
	engineOnce.Do(func() {
		globalEngine = &CanaryEngine{
			results: make(map[int][]CanaryResult),
			tests:   builtinTests,
			enabled: false,
		}
	})
	return globalEngine
}

// SetEnabled 启用/禁用 Canary 监控
func (e *CanaryEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
	if enabled {
		common.SysLog("[Canary] 质量监控已启用")
		go e.runPeriodicTests()
	}
}

// IsEnabled 检查是否启用
func (e *CanaryEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// GetTests 获取测试用例列表
func (e *CanaryEngine) GetTests() []CanaryTest {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.tests
}

// RunSingleTest 对单个渠道运行单个测试
func (e *CanaryEngine) RunSingleTest(channelID int, channelName, modelName, baseURL, apiKey string, test CanaryTest) CanaryResult {
	result := CanaryResult{
		ChannelID:   channelID,
		ChannelName: channelName,
		ModelName:   modelName,
		TestID:      test.ID,
		TestedAt:    time.Now().Unix(),
	}

	start := time.Now()
	response, err := callLLM(baseURL, apiKey, modelName, test.Prompt)
	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		result.Score = 0
		result.Pass = false
		return result
	}

	// 计算得分
	responseLen := len(response)
	keywordsFound := 0
	for _, kw := range test.Keywords {
		if strings.Contains(response, kw) {
			keywordsFound++
		}
	}
	result.ResponseLen = responseLen
	result.KeywordsFound = keywordsFound

	// 综合评分算法
	lengthScore := math.Min(float64(responseLen)/float64(test.ExpectedLength), 1.5) * 40 // 长度分 0-60
	keywordScore := float64(keywordsFound) / float64(len(test.Keywords)) * 40             // 关键词分 0-40
	latencyScore := 20.0
	if result.LatencyMs > 10000 {
		latencyScore = 0
	} else if result.LatencyMs > 5000 {
		latencyScore = 10
	} else if result.LatencyMs > 2000 {
		latencyScore = 15
	}

	result.Score = lengthScore + keywordScore + latencyScore
	result.Pass = result.Score >= 60

	return result
}

// RunAllTestsForChannel 对单个渠道运行所有测试
func (e *CanaryEngine) RunAllTestsForChannel(channelID int, channelName, modelName, baseURL, apiKey string) []CanaryResult {
	var results []CanaryResult
	for _, test := range e.tests {
		result := e.RunSingleTest(channelID, channelName, modelName, baseURL, apiKey, test)
		results = append(results, result)
	}

	// 存储结果
	e.mu.Lock()
	e.results[channelID] = results
	e.mu.Unlock()

	return results
}

// GetChannelHealth 获取渠道健康度
func (e *CanaryEngine) GetChannelHealth(channelID int) *ChannelHealth {
	e.mu.RLock()
	defer e.mu.RUnlock()

	results, ok := e.results[channelID]
	if !ok || len(results) == 0 {
		return nil
	}

	health := &ChannelHealth{
		ChannelID:  channelID,
		TotalTests: len(results),
	}

	totalScore := 0.0
	passedCount := 0
	totalLatency := int64(0)

	for _, r := range results {
		totalScore += r.Score
		totalLatency += r.LatencyMs
		if r.Pass {
			passedCount++
		}
		if health.ChannelName == "" {
			health.ChannelName = r.ChannelName
			health.ModelName = r.ModelName
		}
		if r.TestedAt > health.LastTestAt {
			health.LastTestAt = r.TestedAt
		}
	}

	health.Score = totalScore / float64(len(results))
	health.PassRate = float64(passedCount) / float64(len(results)) * 100
	health.AvgLatencyMs = totalLatency / int64(len(results))
	health.PassedTests = passedCount

	return health
}

// GetAllChannelHealth 获取所有渠道健康度
func (e *CanaryEngine) GetAllChannelHealth() map[int]*ChannelHealth {
	e.mu.RLock()
	defer e.mu.RUnlock()

	healthMap := make(map[int]*ChannelHealth)
	for channelID := range e.results {
		healthMap[channelID] = e.GetChannelHealth(channelID)
	}
	return healthMap
}

// GetScore 获取渠道质量分数（供 SmartRouter 使用）
func (e *CanaryEngine) GetScore(channelID int) float64 {
	health := e.GetChannelHealth(channelID)
	if health == nil {
		return 75.0 // 默认分数
	}
	return health.Score
}

// runPeriodicTests 定期运行测试（每30分钟）
func (e *CanaryEngine) runPeriodicTests() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if !e.IsEnabled() {
			break
		}
		common.SysLog("[Canary] 执行定期质量检查...")
		// 定期测试逻辑（需要遍历所有渠道）
		// 实际实现时从数据库获取渠道列表
	}
}

// callLLM 调用 LLM API
func callLLM(baseURL, apiKey, model, prompt string) (string, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"

	reqBody := map[string]interface{}{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.7,
		"max_tokens":  500,
	}

	bodyBytes, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("无响应内容")
	}

	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content, _ := message["content"].(string)

	return content, nil
}
