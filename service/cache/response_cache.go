package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/go-redis/redis/v8"
)

// CachedResponse 缓存的 API 响应
type CachedResponse struct {
	Body         string            `json:"body"`
	Headers      map[string]string `json:"headers"`
	StatusCode   int               `json:"status_code"`
	Model        string            `json:"model"`
	InputTokens  int               `json:"input_tokens"`
	OutputTokens int               `json:"output_tokens"`
	CachedAt     int64             `json:"cached_at"`
}

// GenerateCacheKey 根据请求参数生成缓存 Key
func GenerateCacheKey(model string, messages interface{}, temperature float64) string {
	msgBytes, _ := json.Marshal(messages)
	data := fmt.Sprintf("%s:%.2f:%s", model, temperature, string(msgBytes))
	hash := sha256.Sum256([]byte(data))
	return "api_cache:" + hex.EncodeToString(hash[:16])
}

// GetCachedResponse 从 Redis 获取缓存的响应
func GetCachedResponse(cacheKey string) (*CachedResponse, error) {
	if !common.RedisEnabled {
		return nil, fmt.Errorf("redis not enabled")
	}

	val, err := common.RDB.Get(context.Background(), cacheKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cached CachedResponse
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

// SetCachedResponse 将响应写入 Redis 缓存
func SetCachedResponse(cacheKey string, resp *CachedResponse, ttl time.Duration) error {
	if !common.RedisEnabled {
		return fmt.Errorf("redis not enabled")
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	return common.RDB.Set(context.Background(), cacheKey, data, ttl).Err()
}

// InvalidateCache 失效缓存
func InvalidateCache(pattern string) error {
	if !common.RedisEnabled {
		return nil
	}

	iter := common.RDB.Scan(context.Background(), 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(context.Background()) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		return common.RDB.Del(context.Background(), keys...).Err()
	}
	return nil
}
