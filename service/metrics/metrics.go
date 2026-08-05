package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics 全局指标收集器
type Metrics struct {
	TotalRequests         atomic.Int64
	SuccessRequests       atomic.Int64
	FailedRequests        atomic.Int64
	TotalPromptTokens     atomic.Int64
	TotalCompletionTokens atomic.Int64
	TotalCostCents        atomic.Int64
	ChannelRequests       sync.Map
	ChannelLatencies      sync.Map
	CacheHits             atomic.Int64
	CacheMisses           atomic.Int64
	StartTime             time.Time
}

type ChannelMetrics struct {
	Requests         atomic.Int64
	Successes        atomic.Int64
	Failures         atomic.Int64
	PromptTokens     atomic.Int64
	CompletionTokens atomic.Int64
	CostCents        atomic.Int64
}

type LatencyMetrics struct {
	Count atomic.Int64
	Sum   atomic.Int64
	Min   atomic.Int64
	Max   atomic.Int64
}

var GlobalMetrics = &Metrics{StartTime: time.Now()}

// RecordRequest 记录一次 API 请求
func RecordRequest(channelID int, success bool, promptTokens, completionTokens int, costUSD float64, latencyMs int64) {
	GlobalMetrics.TotalRequests.Add(1)
	if success {
		GlobalMetrics.SuccessRequests.Add(1)
	} else {
		GlobalMetrics.FailedRequests.Add(1)
	}
	GlobalMetrics.TotalPromptTokens.Add(int64(promptTokens))
	GlobalMetrics.TotalCompletionTokens.Add(int64(completionTokens))
	GlobalMetrics.TotalCostCents.Add(int64(costUSD * 100))

	if channelID > 0 {
		val, _ := GlobalMetrics.ChannelRequests.LoadOrStore(channelID, &ChannelMetrics{})
		cm := val.(*ChannelMetrics)
		cm.Requests.Add(1)
		if success {
			cm.Successes.Add(1)
		} else {
			cm.Failures.Add(1)
		}
		cm.PromptTokens.Add(int64(promptTokens))
		cm.CompletionTokens.Add(int64(completionTokens))
		cm.CostCents.Add(int64(costUSD * 100))

		val2, _ := GlobalMetrics.ChannelLatencies.LoadOrStore(channelID, &LatencyMetrics{})
		lm := val2.(*LatencyMetrics)
		lm.Count.Add(1)
		lm.Sum.Add(latencyMs)
		for {
			cur := lm.Min.Load()
			if cur == 0 || latencyMs < cur {
				if lm.Min.CompareAndSwap(cur, latencyMs) {
					break
				}
			} else {
				break
			}
		}
		for {
			cur := lm.Max.Load()
			if latencyMs > cur {
				if lm.Max.CompareAndSwap(cur, latencyMs) {
					break
				}
			} else {
				break
			}
		}
	}
}

func RecordCacheHit()  { GlobalMetrics.CacheHits.Add(1) }
func RecordCacheMiss() { GlobalMetrics.CacheMisses.Add(1) }

// PrometheusHandler Prometheus 格式指标输出
func PrometheusHandler(c *gin.Context) {
	w := c.Writer
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	uptime := time.Since(GlobalMetrics.StartTime).Seconds()
	totalReq := GlobalMetrics.TotalRequests.Load()
	successReq := GlobalMetrics.SuccessRequests.Load()
	failedReq := GlobalMetrics.FailedRequests.Load()
	promptTokens := GlobalMetrics.TotalPromptTokens.Load()
	completionTokens := GlobalMetrics.TotalCompletionTokens.Load()
	totalCost := float64(GlobalMetrics.TotalCostCents.Load()) / 100
	cacheHits := GlobalMetrics.CacheHits.Load()
	cacheMisses := GlobalMetrics.CacheMisses.Load()

	fmt.Fprintf(w, "# HELP yuxin_api_uptime_seconds API uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE yuxin_api_uptime_seconds gauge\n")
	fmt.Fprintf(w, "yuxin_api_uptime_seconds %.2f\n\n", uptime)

	fmt.Fprintf(w, "# HELP yuxin_api_requests_total Total API requests\n")
	fmt.Fprintf(w, "# TYPE yuxin_api_requests_total counter\n")
	fmt.Fprintf(w, "yuxin_api_requests_total %d\n", totalReq)
	fmt.Fprintf(w, "yuxin_api_requests_total{type=\"success\"} %d\n", successReq)
	fmt.Fprintf(w, "yuxin_api_requests_total{type=\"failed\"} %d\n\n", failedReq)

	fmt.Fprintf(w, "# HELP yuxin_api_tokens_total Total tokens consumed\n")
	fmt.Fprintf(w, "# TYPE yuxin_api_tokens_total counter\n")
	fmt.Fprintf(w, "yuxin_api_tokens_total{type=\"prompt\"} %d\n", promptTokens)
	fmt.Fprintf(w, "yuxin_api_tokens_total{type=\"completion\"} %d\n\n", completionTokens)

	fmt.Fprintf(w, "# HELP yuxin_api_cost_usd_total Total cost in USD\n")
	fmt.Fprintf(w, "# TYPE yuxin_api_cost_usd_total counter\n")
	fmt.Fprintf(w, "yuxin_api_cost_usd_total %.4f\n\n", totalCost)

	fmt.Fprintf(w, "# HELP yuxin_api_cache_total Cache hit/miss counter\n")
	fmt.Fprintf(w, "# TYPE yuxin_api_cache_total counter\n")
	fmt.Fprintf(w, "yuxin_api_cache_total{type=\"hit\"} %d\n", cacheHits)
	fmt.Fprintf(w, "yuxin_api_cache_total{type=\"miss\"} %d\n\n", cacheMisses)

	fmt.Fprintf(w, "# HELP yuxin_api_channel_requests_total Requests per channel\n")
	fmt.Fprintf(w, "# TYPE yuxin_api_channel_requests_total counter\n")
	GlobalMetrics.ChannelRequests.Range(func(key, value interface{}) bool {
		channelID := key.(int)
		cm := value.(*ChannelMetrics)
		fmt.Fprintf(w, "yuxin_api_channel_requests_total{channel=\"%d\"} %d\n", channelID, cm.Requests.Load())
		fmt.Fprintf(w, "yuxin_api_channel_success_total{channel=\"%d\"} %d\n", channelID, cm.Successes.Load())
		fmt.Fprintf(w, "yuxin_api_channel_failures_total{channel=\"%d\"} %d\n", channelID, cm.Failures.Load())
		fmt.Fprintf(w, "yuxin_api_channel_prompt_tokens_total{channel=\"%d\"} %d\n", channelID, cm.PromptTokens.Load())
		fmt.Fprintf(w, "yuxin_api_channel_completion_tokens_total{channel=\"%d\"} %d\n", channelID, cm.CompletionTokens.Load())
		fmt.Fprintf(w, "yuxin_api_channel_cost_usd_total{channel=\"%d\"} %.4f\n", channelID, float64(cm.CostCents.Load())/100)
		return true
	})
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP yuxin_api_channel_latency_ms Channel latency in milliseconds\n")
	fmt.Fprintf(w, "# TYPE yuxin_api_channel_latency_ms gauge\n")
	GlobalMetrics.ChannelLatencies.Range(func(key, value interface{}) bool {
		channelID := key.(int)
		lm := value.(*LatencyMetrics)
		count := lm.Count.Load()
		sum := lm.Sum.Load()
		avg := float64(0)
		if count > 0 {
			avg = float64(sum) / float64(count)
		}
		fmt.Fprintf(w, "yuxin_api_channel_latency_avg_ms{channel=\"%d\"} %.2f\n", channelID, avg)
		fmt.Fprintf(w, "yuxin_api_channel_latency_min_ms{channel=\"%d\"} %d\n", channelID, lm.Min.Load())
		fmt.Fprintf(w, "yuxin_api_channel_latency_max_ms{channel=\"%d\"} %d\n", channelID, lm.Max.Load())
		return true
	})
}

// GetMetricsSummary JSON 格式指标摘要
func GetMetricsSummary() map[string]interface{} {
	uptime := time.Since(GlobalMetrics.StartTime).Seconds()
	totalReq := GlobalMetrics.TotalRequests.Load()
	cacheHits := GlobalMetrics.CacheHits.Load()
	cacheMisses := GlobalMetrics.CacheMisses.Load()
	totalCache := cacheHits + cacheMisses

	cacheHitRate := float64(0)
	if totalCache > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCache) * 100
	}

	errorRate := float64(0)
	if totalReq > 0 {
		errorRate = float64(GlobalMetrics.FailedRequests.Load()) / float64(totalReq) * 100
	}

	channels := make(map[int]map[string]interface{})
	GlobalMetrics.ChannelRequests.Range(func(key, value interface{}) bool {
		channelID := key.(int)
		cm := value.(*ChannelMetrics)
		lm, _ := GlobalMetrics.ChannelLatencies.Load(channelID)
		count := int64(0)
		sum := int64(0)
		min := int64(0)
		max := int64(0)
		if lm != nil {
			latency := lm.(*LatencyMetrics)
			count = latency.Count.Load()
			sum = latency.Sum.Load()
			min = latency.Min.Load()
			max = latency.Max.Load()
		}
		avgLatency := float64(0)
		if count > 0 {
			avgLatency = float64(sum) / float64(count)
		}

		channels[channelID] = map[string]interface{}{
			"requests":          cm.Requests.Load(),
			"successes":         cm.Successes.Load(),
			"failures":          cm.Failures.Load(),
			"prompt_tokens":     cm.PromptTokens.Load(),
			"completion_tokens": cm.CompletionTokens.Load(),
			"cost_usd":          float64(cm.CostCents.Load()) / 100,
			"latency_avg_ms":    avgLatency,
			"latency_min_ms":    min,
			"latency_max_ms":    max,
		}
		return true
	})

	return map[string]interface{}{
		"uptime_seconds":          uptime,
		"total_requests":          totalReq,
		"success_requests":        GlobalMetrics.SuccessRequests.Load(),
		"failed_requests":         GlobalMetrics.FailedRequests.Load(),
		"error_rate_percent":      errorRate,
		"total_prompt_tokens":     GlobalMetrics.TotalPromptTokens.Load(),
		"total_completion_tokens": GlobalMetrics.TotalCompletionTokens.Load(),
		"total_cost_usd":          float64(GlobalMetrics.TotalCostCents.Load()) / 100,
		"cache_hits":              cacheHits,
		"cache_misses":            cacheMisses,
		"cache_hit_rate_percent":  cacheHitRate,
		"channels":                channels,
	}
}

// MetricsMiddleware 请求指标收集中间件
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latencyMs := time.Since(start).Milliseconds()

		channelID := 0
		if id, exists := c.Get("channel_id"); exists {
			if v, ok := id.(int); ok {
				channelID = v
			}
		}

		promptTokens := 0
		completionTokens := 0
		if usage, exists := c.Get("token_usage"); exists {
			if u, ok := usage.(map[string]int); ok {
				promptTokens = u["prompt_tokens"]
				completionTokens = u["completion_tokens"]
			}
		}

		costUSD := 0.0
		if cost, exists := c.Get("request_cost"); exists {
			if v, ok := cost.(float64); ok {
				costUSD = v
			}
		}

		success := c.Writer.Status() < 400
		RecordRequest(channelID, success, promptTokens, completionTokens, costUSD, latencyMs)
	}
}
