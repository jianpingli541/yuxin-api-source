package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreateAndQueryAdminAuditLogs(t *testing.T) {
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&AdminAuditLog{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&AdminAuditLog{}).Error)
	})

	now := common.GetTimestamp()
	logs := []AdminAuditLog{
		{CreatedAt: now - 300, AdminUserId: 1, AdminName: "root", Action: "channel.create", TargetType: "channel", Method: "POST", Path: "/api/channel/", StatusCode: 200, Success: true, Ip: "10.0.0.1"},
		{CreatedAt: now - 200, AdminUserId: 1, AdminName: "root", Action: "channel.update", TargetType: "channel", TargetId: "7", Method: "PUT", Path: "/api/channel/7", StatusCode: 200, Success: true, Ip: "10.0.0.1"},
		{CreatedAt: now - 100, AdminUserId: 2, AdminName: "alice", Action: "user.delete", TargetType: "user", TargetId: "3", Method: "DELETE", Path: "/api/user/3", StatusCode: 403, Success: false, Ip: "10.0.0.2"},
	}
	for i := range logs {
		require.NoError(t, CreateAdminAuditLog(&logs[i]))
	}

	t.Run("no filters returns all newest first", func(t *testing.T) {
		got, total, err := GetAdminAuditLogs(AdminAuditLogFilter{StartIdx: 0, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		require.Len(t, got, 3)
		assert.Equal(t, "user.delete", got[0].Action)
		assert.NotZero(t, got[0].Id)
	})

	t.Run("filter by action", func(t *testing.T) {
		got, total, err := GetAdminAuditLogs(AdminAuditLogFilter{Action: "channel.update", StartIdx: 0, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, got, 1)
		assert.Equal(t, "7", got[0].TargetId)
	})

	t.Run("filter by admin id and name", func(t *testing.T) {
		_, total, err := GetAdminAuditLogs(AdminAuditLogFilter{AdminUserId: 2, StartIdx: 0, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)

		_, total, err = GetAdminAuditLogs(AdminAuditLogFilter{AdminName: "root", StartIdx: 0, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
	})

	t.Run("filter by time range", func(t *testing.T) {
		got, total, err := GetAdminAuditLogs(AdminAuditLogFilter{StartTimestamp: now - 250, EndTimestamp: now - 150, StartIdx: 0, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, got, 1)
		assert.Equal(t, "channel.update", got[0].Action)
	})

	t.Run("filter by target type", func(t *testing.T) {
		_, total, err := GetAdminAuditLogs(AdminAuditLogFilter{TargetType: "channel", StartIdx: 0, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
	})

	t.Run("pagination offset", func(t *testing.T) {
		got, total, err := GetAdminAuditLogs(AdminAuditLogFilter{StartIdx: 1, PageSize: 1})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		require.Len(t, got, 1)
		assert.Equal(t, "channel.update", got[0].Action)
	})
}

func TestTruncateAuditValue(t *testing.T) {
	assert.Equal(t, "", TruncateAuditValue("", 10))
	assert.Equal(t, "short", TruncateAuditValue("short", 10))

	long := strings.Repeat("a", 100)
	got := TruncateAuditValue(long, 30)
	assert.LessOrEqual(t, len(got), 30)
	assert.True(t, strings.HasSuffix(got, "...(truncated)"))

	// 多字节字符不从字符中间截断，结果仍是合法 UTF-8
	multi := strings.Repeat("密", 50) // 每字符 3 字节
	got = TruncateAuditValue(multi, 40)
	assert.LessOrEqual(t, len(got), 40)
	assert.True(t, strings.HasSuffix(got, "...(truncated)"))
	assert.True(t, strings.Count(got, "密") > 0)
	withoutSuffix := strings.TrimSuffix(got, "...(truncated)")
	assert.Equal(t, strings.Repeat("密", strings.Count(withoutSuffix, "密")), withoutSuffix)
}

func TestPrepareAuditValueMasksSecrets(t *testing.T) {
	raw := []byte(`{"name":"prod","key":"sk-live-abc","nested":{"password":"p@ss","note":"ok"},"list":[{"token":"tk-1","x":1}]}`)
	got := PrepareAuditValue(raw, AdminAuditValueMaxBytes)
	assert.Contains(t, got, `"name":"prod"`)
	assert.Contains(t, got, `"note":"ok"`)
	assert.NotContains(t, got, "sk-live-abc")
	assert.NotContains(t, got, "p@ss")
	assert.NotContains(t, got, "tk-1")
	assert.Contains(t, got, `"***"`)

	assert.Equal(t, "", PrepareAuditValue([]byte("not json"), AdminAuditValueMaxBytes))
	assert.Equal(t, "", PrepareAuditValue(nil, AdminAuditValueMaxBytes))
	assert.Equal(t, "", PrepareAuditValue([]byte("  "), AdminAuditValueMaxBytes))
}

func TestPrepareAuditValueTruncatesLongJSON(t *testing.T) {
	raw := []byte(`{"blob":"` + strings.Repeat("x", 10000) + `"}`)
	got := PrepareAuditValue(raw, AdminAuditValueMaxBytes)
	assert.LessOrEqual(t, len(got), AdminAuditValueMaxBytes)
	assert.True(t, strings.HasSuffix(got, "...(truncated)"))
}
