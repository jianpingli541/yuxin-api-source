package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetAdminAuditLogs 管理员操作审计日志查询端点（AdminAuth 鉴权）。
// 过滤参数：action（精确匹配，如 channel.create）、admin_user_id、username
// （操作者用户名）、target_type、start_timestamp / end_timestamp（秒级时间戳，
// 含端点）；分页走 p / page_size，id 倒序返回最新条目。
func GetAdminAuditLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	filter := model.AdminAuditLogFilter{
		Action:     c.Query("action"),
		AdminName:  c.Query("username"),
		TargetType: c.Query("target_type"),
		StartIdx:   pageInfo.GetStartIdx(),
		PageSize:   pageInfo.GetPageSize(),
	}
	filter.AdminUserId, _ = strconv.Atoi(c.Query("admin_user_id"))
	filter.StartTimestamp, _ = strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	filter.EndTimestamp, _ = strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	logs, total, err := model.GetAdminAuditLogs(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}
