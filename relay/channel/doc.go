// Package channel 定义统一的渠道（上游供应商）抽象层与通用上游请求工具。
//
// Adaptor 与 TaskAdaptor 接口把各 AI 上游渠道约束为统一的调用契约，
// 具体渠道实现位于本目录的各子目录（openai/、claude/、gemini/ 等）。
// 本包同时提供请求组装、上游调用与可靠性处理等中转通用能力。
package channel
