package middleware

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 严格白名单:无域名阶段只允许本机 origin
// 正式域名落地后,在 .env 加 CORS_ALLOWED_ORIGINS=https://your.domain 并改读此处
var corsAllowedOrigins = []string{
	"http://localhost",
	"http://localhost:3000",
	"http://127.0.0.1",
	"http://127.0.0.1:3000",
	"http://103.55.131.130", // 当前裸 IP(无域名过渡)
}

func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = false
	config.AllowOrigins = corsAllowedOrigins
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Requested-With", "Accept", "X-Routing-Strategy"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowWildcard = false
	return cors.New(config)
}

func Version() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-New-Api-Version", common.Version)
		c.Next()
	}
}
