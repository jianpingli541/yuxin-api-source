package controller

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// 登录锁定: 密码阶段按 用户名+客户端IP 计数, 连续失败达到阈值后临时锁定。
// 按 IP 维度隔离, 避免攻击者通过撞错误密码对管理员账户实施锁定 DoS。
const (
	loginFailThreshold   = 5
	loginLockDurationSec = 900 // 15 分钟
)

func loginFailKey(username, clientIP string) string {
	return "login_fail:" + username + "|" + clientIP
}

func loginLockKey(username, clientIP string) string {
	return "login_lock:" + username + "|" + clientIP
}

// isLoginLocked 检查 用户名+来源IP 是否处于登录锁定状态。
func isLoginLocked(c *gin.Context, username string) bool {
	if !common.RedisEnabled || common.RDB == nil {
		return false
	}
	n, err := common.RDB.Exists(c.Request.Context(), loginLockKey(username, c.ClientIP())).Result()
	return err == nil && n > 0
}

// recordLoginFailure 记录一次密码失败; 达到阈值后锁定并写系统日志。
func recordLoginFailure(c *gin.Context, username string) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	ctx := c.Request.Context()
	ip := c.ClientIP()
	key := loginFailKey(username, ip)
	count, err := common.RDB.Incr(ctx, key).Result()
	if err != nil {
		return
	}
	if count == 1 {
		common.RDB.Expire(ctx, key, time.Duration(loginLockDurationSec)*time.Second)
	}
	if count >= loginFailThreshold {
		common.RDB.Set(ctx, loginLockKey(username, ip), "1", time.Duration(loginLockDurationSec)*time.Second)
		common.RDB.Del(ctx, key)
		common.SysLog(fmt.Sprintf("Login locked for user %s from %s after %d failed attempts", username, ip, count))
	}
}

// clearLoginFailures 登录成功后清除失败计数与锁定。
func clearLoginFailures(c *gin.Context, username string) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	common.RDB.Del(c.Request.Context(), loginFailKey(username, c.ClientIP()), loginLockKey(username, c.ClientIP()))
}
