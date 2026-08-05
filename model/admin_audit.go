package model

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

// AdminAuditValueMaxBytes 审计旧值/新值字段的最大存储字节数（超出截断）。
const AdminAuditValueMaxBytes = 4096

// auditMaskedValue 脱敏占位符。
const auditMaskedValue = "***"

// AdminAuditLog 管理员操作审计日志：记录管理端写操作的操作者、目标、
// 变更前后值快照与结果。与 Log（LogTypeManage，面向前端 i18n 展示）互补：
// 本表保留结构化前后值对比，供审计追溯与导出。
type AdminAuditLog struct {
	Id          int    `json:"id"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index"`
	AdminUserId int    `json:"admin_user_id" gorm:"index"`
	AdminName   string `json:"admin_name" gorm:"index;default:''"`
	Action      string `json:"action" gorm:"index;default:''"` // 如 channel.create；未登记写操作回退 generic
	TargetType  string `json:"target_type" gorm:"index;default:''"`
	TargetId    string `json:"target_id" gorm:"index;default:''"`
	Method      string `json:"method" gorm:"default:''"`
	Path        string `json:"path"`
	OldValue    string `json:"old_value" gorm:"type:text"` // 变更前 JSON 快照，已脱敏并截断
	NewValue    string `json:"new_value" gorm:"type:text"` // 请求体 JSON，已脱敏并截断
	StatusCode  int    `json:"status_code"`
	Success     bool   `json:"success"`
	Ip          string `json:"ip" gorm:"index;default:''"`
}

// CreateAdminAuditLog 落库一条管理审计日志。CreatedAt 缺省取当前时间。
func CreateAdminAuditLog(log *AdminAuditLog) error {
	if log.CreatedAt == 0 {
		log.CreatedAt = common.GetTimestamp()
	}
	return DB.Create(log).Error
}

// AdminAuditLogFilter 审计日志查询条件，零值字段表示不过滤。
type AdminAuditLogFilter struct {
	Action         string
	AdminUserId    int
	AdminName      string
	TargetType     string
	StartTimestamp int64
	EndTimestamp   int64
	StartIdx       int
	PageSize       int
}

// GetAdminAuditLogs 按条件分页查询审计日志，id 倒序（最新在前）。
func GetAdminAuditLogs(f AdminAuditLogFilter) ([]*AdminAuditLog, int64, error) {
	tx := DB.Model(&AdminAuditLog{})
	if f.Action != "" {
		tx = tx.Where("action = ?", f.Action)
	}
	if f.AdminUserId > 0 {
		tx = tx.Where("admin_user_id = ?", f.AdminUserId)
	}
	if f.AdminName != "" {
		tx = tx.Where("admin_name = ?", f.AdminName)
	}
	if f.TargetType != "" {
		tx = tx.Where("target_type = ?", f.TargetType)
	}
	if f.StartTimestamp > 0 {
		tx = tx.Where("created_at >= ?", f.StartTimestamp)
	}
	if f.EndTimestamp > 0 {
		tx = tx.Where("created_at <= ?", f.EndTimestamp)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	pageSize := f.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	startIdx := f.StartIdx
	if startIdx < 0 {
		startIdx = 0
	}
	logs := make([]*AdminAuditLog, 0, pageSize)
	if err := tx.Order("id desc").Limit(pageSize).Offset(startIdx).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// auditSecretKeyPatterns JSON 键名包含其中任一子串即视为敏感字段，审计快照中掩码。
// 宁多掩不漏泄：允许误掩普通字段（如 token_name）。
var auditSecretKeyPatterns = []string{
	"key", "keys", "secret", "password", "passwd", "pwd",
	"token", "credential", "authorization",
}

func isAuditSecretKey(key string) bool {
	lower := strings.ToLower(key)
	for _, pattern := range auditSecretKeyPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// maskAuditSecrets 递归掩码 map/slice 中的敏感键值。
func maskAuditSecrets(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		for key, child := range val {
			if isAuditSecretKey(key) {
				val[key] = auditMaskedValue
				continue
			}
			maskAuditSecrets(child)
		}
	case []interface{}:
		for _, child := range val {
			maskAuditSecrets(child)
		}
	}
}

// TruncateAuditValue 按字节截断，回退到完整 UTF-8 字符边界并追加截断标记。
// 空串或 maxBytes<=0 原样返回。
func TruncateAuditValue(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	const suffix = "...(truncated)"
	cut := maxBytes - len(suffix)
	if cut < 0 {
		cut = 0
	}
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + suffix
}

// PrepareAuditValue 将原始 JSON 脱敏（掩码敏感键）并截断到 maxBytes 内。
// 非 JSON 内容（二进制/表单/非法 JSON）返回 ""，不落库原始内容以免泄漏。
func PrepareAuditValue(raw []byte, maxBytes int) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	var obj interface{}
	if err := common.Unmarshal(trimmed, &obj); err != nil {
		return ""
	}
	maskAuditSecrets(obj)
	out, err := common.Marshal(obj)
	if err != nil {
		return ""
	}
	return TruncateAuditValue(string(out), maxBytes)
}
