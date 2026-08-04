package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// WebAssets holds the embedded dashboard frontend assets.
type WebAssets struct {
	BuildFS   embed.FS
	IndexPage []byte
}

func SetWebRouter(router *gin.Engine, assets WebAssets) {
	frontendFS := common.EmbedFolder(assets.BuildFS, "web/dist")

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.Cache())
	router.Use(static.Serve("/", frontendFS))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		uri := c.Request.RequestURI
		// R9 修复(2026-08-03):静态资源(JS/CSS/字体/图片)由 static.Serve 直接处理,
		// 不再计入全局限流;仅对 SPA HTML 路由页应用限流,
		// 避免首屏并发拉取 10+ 静态资源导致用户秒触 429。
		isStatic := strings.HasPrefix(uri, "/static/") || strings.HasPrefix(uri, "/favicon") ||
			strings.HasPrefix(uri, "/robots.txt") || strings.HasPrefix(uri, "/manifest") ||
			strings.HasPrefix(uri, "/logo") || strings.HasPrefix(uri, "/.well-known/")
		isSpaRoute := !strings.HasPrefix(uri, "/v1") && !strings.HasPrefix(uri, "/api") &&
			!strings.HasPrefix(uri, "/assets")
		if !isStatic && isSpaRoute {
			middleware.GlobalWebRateLimit()(c)
			if c.IsAborted() {
				return
			}
		}
		if strings.HasPrefix(uri, "/v1") || strings.HasPrefix(uri, "/api") || strings.HasPrefix(uri, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
	})
}
