package routing

import (
	"sync"
)

// Config 路由配置
type Config struct {
	Strategy      RoutingStrategy `json:"strategy"`
	FallbackChain []int           `json:"fallback_chain,omitempty"`
}

var (
	defaultConfig = Config{
		Strategy: StrategyPriorityWeight,
	}
	configMutex sync.RWMutex
)

// GetDefaultConfig 获取默认路由配置
func GetDefaultConfig() Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return defaultConfig
}

// SetDefaultConfig 设置默认路由配置
func SetDefaultConfig(cfg Config) {
	configMutex.Lock()
	defer configMutex.Unlock()
	defaultConfig = cfg
}
