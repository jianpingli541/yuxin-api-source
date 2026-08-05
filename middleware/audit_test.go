package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupAdminAuditMiddlewareTest 在测试范围内打开内存 sqlite 作为 model.DB / LOG_DB，
// 迁移 Audit 依赖的表。设置 MaxOpenConns(1) 保证 :memory: 单连接共享，
// 避免 gopool 异步写库与测试读库跨连接看不到写入。
func setupAdminAuditMiddlewareTest(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	previousRedis := common.RedisEnabled

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Log{},
		&model.AdminAuditLog{},
	))

	model.DB = db
	model.LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousType)
		common.SetLogDatabaseType(previousLogType)
		common.RedisEnabled = previousRedis
	})
}

func createAuditAdminUser(t *testing.T, token string) *model.User {
	t.Helper()
	user := &model.User{
		Username: "audit-admin", Password: "password-placeholder", Role: common.RoleAdminUser,
		Status: common.UserStatusEnabled, Group: "default", AccessToken: &token, AuthVersion: 1,
		AffCode: "audit-admin-aff",
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

// waitForAdminAuditLogs 轮询等待 gopool 异步写入完成（最多 3s），避免测试竞态。
func waitForAdminAuditLogs(t *testing.T, want int) []*model.AdminAuditLog {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var logs []*model.AdminAuditLog
		require.NoError(t, model.DB.Order("id desc").Find(&logs).Error)
		if len(logs) >= want {
			return logs
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d admin audit logs, got %d", want, len(logs))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestAdminAuditMiddlewareRecordsWriteWithOldAndNewValue(t *testing.T) {
	setupAdminAuditMiddlewareTest(t)
	gin.SetMode(gin.TestMode)
	user := createAuditAdminUser(t, "audit.admin.pat-key")

	// 预置一个 channel：旧值快照应包含 name=old-name；Key 在 DB 层 Omit。
	channel := &model.Channel{Name: "old-name", Key: "sk-old-secret", Status: 1, CreatedTime: common.GetTimestamp()}
	require.NoError(t, model.DB.Create(channel).Error)

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.PUT("/api/channel/:id", AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "ok"})
	})

	body := strings.NewReader(`{"name":"new-name","key":"sk-new-secret","status":2}`)
	request := httptest.NewRequest(http.MethodPut, "/api/channel/"+strconv.Itoa(channel.Id), body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer audit.admin.pat-key")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	logs := waitForAdminAuditLogs(t, 1)
	audit := logs[0]
	assert.Equal(t, user.Id, audit.AdminUserId)
	assert.Equal(t, "audit-admin", audit.AdminName)
	assert.Equal(t, "channel.update", audit.Action)
	assert.Equal(t, "channel", audit.TargetType)
	assert.Equal(t, strconv.Itoa(channel.Id), audit.TargetId)
	assert.Equal(t, http.MethodPut, audit.Method)
	assert.Equal(t, "/api/channel/"+strconv.Itoa(channel.Id), audit.Path)
	assert.Equal(t, http.StatusOK, audit.StatusCode)
	assert.True(t, audit.Success)

	// 旧值：含 name；含已掩码的 key 字段；不含原密钥。
	assert.Contains(t, audit.OldValue, "old-name")
	assert.Contains(t, audit.OldValue, `"***"`)
	assert.NotContains(t, audit.OldValue, "sk-old-secret")
	// 新值：含 name；不含新请求体中的密钥。
	assert.Contains(t, audit.NewValue, "new-name")
	assert.NotContains(t, audit.NewValue, "sk-new-secret")
}

func TestAdminAuditMiddlewareSkipsReadOnlyRequests(t *testing.T) {
	setupAdminAuditMiddlewareTest(t)
	gin.SetMode(gin.TestMode)
	createAuditAdminUser(t, "audit.readonly.pat-key")

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/api/channel/:id", AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	request := httptest.NewRequest(http.MethodGet, "/api/channel/1", nil)
	request.Header.Set("Authorization", "Bearer audit.readonly.pat-key")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	// 等异步逻辑完成一段时间，再断言 0 行
	time.Sleep(150 * time.Millisecond)
	var count int64
	require.NoError(t, model.DB.Model(&model.AdminAuditLog{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestAdminAuditMiddlewareCapturesFailureAndRouteMappedAction(t *testing.T) {
	setupAdminAuditMiddlewareTest(t)
	gin.SetMode(gin.TestMode)
	createAuditAdminUser(t, "audit.failure.pat-key")

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.POST("/api/custom-oauth-provider/", AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "bad"})
	})

	body := strings.NewReader(`{"client_secret":"cs-123","name":"demo"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/custom-oauth-provider/", body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer audit.failure.pat-key")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	logs := waitForAdminAuditLogs(t, 1)
	audit := logs[0]
	assert.Equal(t, "custom_oauth.create", audit.Action) // 命中 auditRouteActions
	assert.Equal(t, http.StatusBadRequest, audit.StatusCode)
	assert.False(t, audit.Success)
	assert.NotContains(t, audit.NewValue, "cs-123") // client_secret 包含 "secret" → 掩码
	assert.Contains(t, audit.NewValue, "demo")
}

func TestAuditFallbackAction(t *testing.T) {
	assert.Equal(t, "channel.update", auditFallbackAction("PUT", "/api/channel/:id"))
	assert.Equal(t, "channel.create", auditFallbackAction("POST", "/api/channel/"))
	assert.Equal(t, "user.delete", auditFallbackAction("DELETE", "/api/user/:id"))
	assert.Equal(t, "subscription.update", auditFallbackAction("PATCH", "/api/subscription/admin/plans/:id"))
	assert.Equal(t, "generic", auditFallbackAction("POST", "/metrics"))
	assert.Equal(t, "generic", auditFallbackAction("POST", "/api/:weird"))
}

func TestAuditResponseSuccess(t *testing.T) {
	assert.True(t, auditResponseSuccess(200, []byte(`{"success":true}`)))
	assert.False(t, auditResponseSuccess(200, []byte(`{"success":false}`)))
	assert.False(t, auditResponseSuccess(500, nil))
	assert.True(t, auditResponseSuccess(204, nil))
	assert.True(t, auditResponseSuccess(201, []byte(`{}`)))
}

func TestAuditTargetTypeFromRoute(t *testing.T) {
	assert.Equal(t, "channel", auditTargetTypeFromRoute("/api/channel/:id"))
	assert.Equal(t, "user", auditTargetTypeFromRoute("/api/user/:id/reset_passkey"))
	assert.Equal(t, "subscription", auditTargetTypeFromRoute("/api/subscription/admin/plans/:id"))
	assert.Equal(t, "option", auditTargetTypeFromRoute("/api/option/"))
	assert.Equal(t, "", auditTargetTypeFromRoute("/api/ping"))
	assert.Equal(t, "", auditTargetTypeFromRoute("/metrics"))
}
