package model_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

// AutoRouteSettings 智能调度（模型别名路由）配置。
//
// 客户端在请求中把 model 设为注册表中的别名（如 "auto"），网关在
// 分发阶段（middleware/distributor.go）把别名解析为候选模型池，
// 再按路由策略（X-Yuxin-Route 头或系统默认）在池内选定最终的
// (模型, 渠道) 组合。计费与 X-Yuxin-Model 响应头均以实际生效模型为准。
type AutoRouteSettings struct {
	// Enabled 智能调度总开关；关闭后 model:"auto" 按未知模型报 503。
	Enabled bool `json:"enabled"`
	// Aliases 别名 → 候选模型池。key 为别名（客户端 model 字段值），
	// value 为候选真实模型名列表（须已在上游渠道配置且对调用方分组可用）。
	Aliases map[string][]string `json:"aliases"`
	// DoubaoBoost 字节跳动生态（VolcEngine/Doubao 渠道类型）候选的得分倾斜系数，
	// 0.15 表示 +15%。用于"优先尝鲜字节系新模型"：新模型上架进池即自动享受倾斜，
	// 无需改代码。0 表示关闭倾斜。
	DoubaoBoost float64 `json:"doubao_boost"`
}

var defaultAutoRouteSettings = AutoRouteSettings{
	Enabled: false, // 默认关闭，管理员在系统设置中显式开启并配置别名池
	Aliases: map[string][]string{
		"auto": {},
	},
	DoubaoBoost: 0.15,
}

var autoRouteSettings = defaultAutoRouteSettings

func init() {
	config.GlobalConfig.Register("auto_route", &autoRouteSettings)
}

func GetAutoRouteSettings() *AutoRouteSettings {
	return &autoRouteSettings
}

// ResolveAlias 将模型名解析为候选池；非别名返回 nil, false。
func ResolveAlias(modelName string) ([]string, bool) {
	if !autoRouteSettings.Enabled {
		return nil, false
	}
	pool, ok := autoRouteSettings.Aliases[strings.TrimSpace(modelName)]
	if !ok || len(pool) == 0 {
		return nil, false
	}
	return pool, true
}

// GetDoubaoBoost 返回字节生态倾斜系数（负值/异常值按 0 处理）。
func GetDoubaoBoost() float64 {
	if autoRouteSettings.DoubaoBoost < 0 {
		return 0
	}
	return autoRouteSettings.DoubaoBoost
}
