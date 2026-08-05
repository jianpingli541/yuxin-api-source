// Package controller 是 API 网关的 HTTP 控制器层。
//
// 承载各 REST 端点（用户、令牌、渠道、计费、审计等）的处理函数，
// 是 router 包路由注册的主要入口；数据访问经由 model，业务逻辑委托 service。
package controller
