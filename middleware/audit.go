package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// auditResponseWriter 包装 gin.ResponseWriter，捕获响应状态码并将响应体复制一份到
// 有限大小的缓冲区，用于判断业务是否成功（解析响应 JSON 的 success 字段）。
// 缓冲区有上限，避免大响应（如密钥导出）占用过多内存；超出上限则不再缓存，
// 此时仅依据 HTTP 状态码判断成败。
type auditResponseWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	maxSize int
}

func (w *auditResponseWriter) Write(b []byte) (int, error) {
	if w.body.Len() < w.maxSize {
		remain := w.maxSize - w.body.Len()
		if remain >= len(b) {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remain])
		}
	}
	return w.ResponseWriter.Write(b)
}

func (w *auditResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// auditRouteActions 将「METHOD + 路由模板」映射为语言无关的操作标识 action。
// 这些是未被 handler 手动埋点的写操作，由中间件兜底记录；前端依据 action 用 i18n 本地化展示。
// 未命中的写操作回退为 action="generic"，前端展示 "METHOD route"。
var auditRouteActions = map[string]string{
	// 用户管理
	"POST /api/user/topup/complete":                    "user.topup_complete",
	"DELETE /api/user/:id/reset_passkey":               "user.reset_passkey",
	"DELETE /api/user/:id/oauth/bindings/:provider_id": "user.oauth_unbind",

	// 系统设置（root）
	"POST /api/option/payment_compliance":       "option.payment_compliance",
	"POST /api/option/rest_model_ratio":         "option.reset_ratio",
	"DELETE /api/option/channel_affinity_cache": "option.clear_affinity_cache",

	// 自定义 OAuth（root）
	"POST /api/custom-oauth-provider/":      "custom_oauth.create",
	"PUT /api/custom-oauth-provider/:id":    "custom_oauth.update",
	"DELETE /api/custom-oauth-provider/:id": "custom_oauth.delete",

	// 性能/缓存（root）
	"DELETE /api/performance/disk_cache": "performance.clear_disk_cache",
	"POST /api/performance/gc":           "performance.gc",
	"DELETE /api/performance/logs":       "performance.clear_logs",

	// 兑换码
	"PUT /api/redemption/":           "redemption.update",
	"DELETE /api/redemption/:id":     "redemption.delete",
	"DELETE /api/redemption/invalid": "redemption.delete_invalid",

	// 预填组
	"POST /api/prefill_group/":      "prefill_group.create",
	"PUT /api/prefill_group/":       "prefill_group.update",
	"DELETE /api/prefill_group/:id": "prefill_group.delete",

	// 供应商
	"POST /api/vendors/":      "vendor.create",
	"PUT /api/vendors/":       "vendor.update",
	"DELETE /api/vendors/:id": "vendor.delete",

	// 模型元数据
	"POST /api/models/":              "model.create",
	"PUT /api/models/":               "model.update",
	"DELETE /api/models/:id":         "model.delete",
	"POST /api/models/sync_upstream": "model.sync_upstream",

	// 部署
	"POST /api/deployments/":      "deployment.create",
	"PUT /api/deployments/:id":    "deployment.update",
	"DELETE /api/deployments/:id": "deployment.delete",

	// 订阅（管理员）
	"POST /api/subscription/admin/plans":    "subscription.plan_create",
	"PUT /api/subscription/admin/plans/:id": "subscription.plan_update",
	"POST /api/subscription/admin/bind":     "subscription.bind",

	// 日志
	"POST /api/system-task/log-cleanup": "log.cleanup_start",
}

// adminAuditState 贯穿一次管理写请求的审计上下文：响应捕获器、请求体快照，
// 以及必须在 handler 执行前读取的目标资源旧值。
type adminAuditState struct {
	writer      *auditResponseWriter
	requestBody []byte // 请求体原始字节（已限长），事后脱敏截断再落库
	oldValue    []byte // 目标资源变更前的 JSON 序列化（读取路径已剔除密钥类字段）
	targetType  string
	targetId    string
}

// auditRequestBodyCaptureLimit 请求体捕获上限 2MB：管理端配置类请求远小于此，
// 超限请求只记动作/状态，不落新值快照。
const auditRequestBodyCaptureLimit = 2 * 1024 * 1024

// auditCaptureRequestBody 读取并还原请求体供审计使用。
// 只捕获长度明确的 application/json 请求体：分块传输/压缩请求体读取后无法保证
// 完整还原给 handler，直接跳过（审计行仍会记录，只是没有新值快照）。
func auditCaptureRequestBody(c *gin.Context) []byte {
	if c.Request == nil || c.Request.Body == nil {
		return nil
	}
	if c.Request.ContentLength <= 0 || c.Request.ContentLength > auditRequestBodyCaptureLimit {
		return nil
	}
	if ct := c.Request.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "application/json") {
		return nil
	}
	if ce := c.Request.Header.Get("Content-Encoding"); ce != "" && ce != "identity" {
		return nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil
	}
	// 还原请求体，确保后续 handler 读取不受影响
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

// auditTargetTypeByRoutePrefix 由路由模板前缀推断目标资源类型（长前缀在前）。
var auditTargetTypeByRoutePrefix = []struct {
	prefix     string
	targetType string
}{
	{"/api/subscription/admin/", "subscription"},
	{"/api/custom-oauth-provider/", "custom_oauth_provider"},
	{"/api/prefill_group/", "prefill_group"},
	{"/api/deployments/", "deployment"},
	{"/api/redemption/", "redemption"},
	{"/api/channel/", "channel"},
	{"/api/vendors/", "vendor"},
	{"/api/models/", "model"},
	{"/api/option/", "option"},
	{"/api/token/", "token"},
	{"/api/user/", "user"},
}

func auditTargetTypeFromRoute(route string) string {
	for _, entry := range auditTargetTypeByRoutePrefix {
		if strings.HasPrefix(route, entry.prefix) {
			return entry.targetType
		}
	}
	return ""
}

// auditOldValueLoaders 按目标类型提供旧值读取器，全部走剔除密钥字段的查询路径
// （channel/user 的 selectAll=false 会 Omit key/password/access_token）；
// 其余类型的序列化结果再由 model.PrepareAuditValue 统一掩码兜底。
var auditOldValueLoaders = map[string]func(id int) (interface{}, error){
	"channel":    func(id int) (interface{}, error) { return model.GetChannelById(id, false) },
	"user":       func(id int) (interface{}, error) { return model.GetUserById(id, false) },
	"token":      func(id int) (interface{}, error) { return model.GetTokenById(id) },
	"redemption": func(id int) (interface{}, error) { return model.GetRedemptionById(id) },
}

// auditCaptureOldValue 在 handler 执行前读取目标资源的旧值快照。
// 仅当路由带 :id 参数且该类型有登记读取器时生效；任何错误静默跳过（旧值留空），
// 审计不能反过来阻断主流程。
func auditCaptureOldValue(c *gin.Context, targetType string) []byte {
	loader, ok := auditOldValueLoaders[targetType]
	if !ok || model.DB == nil {
		return nil
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return nil
	}
	value, err := loader(id)
	if err != nil || value == nil {
		return nil
	}
	data, err := common.Marshal(value)
	if err != nil {
		return nil
	}
	return data
}

// beginAdminAudit 在管理/root 写操作进入 handler 前包装 ResponseWriter、
// 捕获请求体并预读目标资源旧值（旧值必须在改库前读）。仅对写方法
// （POST/PUT/PATCH/DELETE）生效；只读请求返回 nil，调用方据此跳过事后记录。
//
// 该函数由 authHelper 在鉴权通过、c.Next() 之前调用：因为任何管理/root 接口都
// 必然经过 AdminAuth/RootAuth，将审计内聚到鉴权链路即可保证「新增接口自动留痕」，
// 无需在路由上再单独挂一层审计中间件（避免漏挂）。
func beginAdminAudit(c *gin.Context) *adminAuditState {
	method := c.Request.Method
	if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
		return nil
	}
	state := &adminAuditState{
		writer: &auditResponseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
			maxSize:        64 * 1024,
		},
	}
	c.Writer = state.writer

	route := c.FullPath()
	state.targetType = auditTargetTypeFromRoute(route)
	state.targetId = c.Param("id")
	state.requestBody = auditCaptureRequestBody(c)
	if state.targetType != "" && state.targetId != "" {
		state.oldValue = auditCaptureOldValue(c, state.targetType)
	}
	return state
}

// finishAdminAudit 在 c.Next() 之后对管理写操作落审计记录：
//  1. AdminAuditLog（结构化前后值快照）：所有管理写操作一律留痕，独立于 handler
//     是否手动埋点，经 gopool 异步写库，失败只记日志不阻塞响应。
//  2. Log（LogTypeManage）兜底：若 handler 已手动埋点（设置 ContextKeyAuditLogged）
//     则跳过，避免重复。
func finishAdminAudit(c *gin.Context, state *adminAuditState) {
	if state == nil {
		return
	}
	writer := state.writer
	method := c.Request.Method

	operatorId := c.GetInt("id")
	operatorName := c.GetString("username")
	ip := c.ClientIP()
	status := writer.Status()
	success := auditResponseSuccess(status, writer.body.Bytes())

	route := c.FullPath()
	action := auditRouteActions[method+" "+route]
	if action == "" {
		action = auditFallbackAction(method, route)
	}

	// 结构化审计：前后值统一脱敏截断后异步入库。
	auditLog := &model.AdminAuditLog{
		CreatedAt:   common.GetTimestamp(),
		AdminUserId: operatorId,
		AdminName:   operatorName,
		Action:      action,
		TargetType:  state.targetType,
		TargetId:    state.targetId,
		Method:      method,
		Path:        c.Request.URL.Path,
		OldValue:    model.PrepareAuditValue(state.oldValue, model.AdminAuditValueMaxBytes),
		NewValue:    model.PrepareAuditValue(state.requestBody, model.AdminAuditValueMaxBytes),
		StatusCode:  status,
		Success:     success,
		Ip:          ip,
	}
	gopool.Go(func() {
		if err := model.CreateAdminAuditLog(auditLog); err != nil {
			common.SysError("failed to record admin audit log: " + err.Error())
		}
	})

	// handler 已手动记录更精细的 Log 审计，跳过兜底。
	if common.GetContextKeyBool(c, constant.ContextKeyAuditLogged) {
		return
	}

	routeParams := map[string]string{}
	for _, p := range c.Params {
		routeParams[p.Key] = p.Value
	}

	// op.params 为语言无关参数，供前端 i18n 渲染；generic 时携带 method/route。
	opParams := map[string]interface{}{}
	if action == "generic" {
		opParams["method"] = method
		opParams["route"] = route
	}

	// content 为英文兜底文本（供导出等非本地化消费者使用）。
	content := method + " " + route

	operatorRole := c.GetInt("role")
	adminInfo := map[string]interface{}{
		"admin_id":       operatorId,
		"admin_username": operatorName,
		"admin_role":     operatorRole,
		"auth_method":    auditAuthMethod(c),
	}
	auditInfo := map[string]interface{}{
		"method":  method,
		"route":   route,
		"path":    c.Request.URL.Path,
		"status":  status,
		"success": success,
	}
	if len(routeParams) > 0 {
		auditInfo["params"] = routeParams
	}

	gopool.Go(func() {
		model.RecordOperationAuditLog(operatorId, content, ip, action, opParams, adminInfo, auditInfo)
	})
}

// auditFallbackAction 为未登记的写操作生成「资源.动作」形式的 action。
// 例：PUT /api/channel/:id -> channel.update；无法从路由推断资源时返回 generic。
func auditFallbackAction(method string, route string) string {
	resource := ""
	trimmed := strings.TrimPrefix(route, "/api/")
	if trimmed != route && trimmed != "" {
		segments := strings.Split(trimmed, "/")
		if len(segments) > 0 && segments[0] != "" && !strings.HasPrefix(segments[0], ":") {
			resource = segments[0]
		}
	}
	if resource == "" {
		return "generic"
	}
	var verb string
	switch method {
	case http.MethodPost:
		verb = "create"
	case http.MethodPut:
		verb = "update"
	case http.MethodPatch:
		verb = "update"
	case http.MethodDelete:
		verb = "delete"
	default:
		verb = strings.ToLower(method)
	}
	return resource + "." + verb
}

func auditAuthMethod(c *gin.Context) string {
	if c.GetBool("use_access_token") {
		return "access_token"
	}
	return "session"
}

// auditResponseSuccess 依据 HTTP 状态码与响应体推断操作是否成功。
// 优先解析响应 JSON 中的 success 字段；无法解析时退回到状态码判断。
func auditResponseSuccess(status int, body []byte) bool {
	if status >= 400 {
		return false
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var resp struct {
			Success *bool `json:"success"`
		}
		if err := common.Unmarshal(trimmed, &resp); err == nil && resp.Success != nil {
			return *resp.Success
		}
	}
	return status < 400
}
