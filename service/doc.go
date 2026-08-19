// Package service 是 API 网关的业务服务层。
//
// 封装计费、令牌鉴权、渠道亲和、金丝雀、缓存、webhook 等业务能力，
// 供 controller 与 relay 调用；authz/、cache/、canary/ 等子包为独立域服务单元。
package service
