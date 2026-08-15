// payment-setup 配置外部支付通道（微信 Native / 支付宝 Face-to-Face）。
//
// 背景：new-api 默认通过 GUI 面板录入，但 root 账号 2FA 加持下运维不便
// 反复登入浏览器；本工具用最小化启动方式（仅初始化 DB + OptionMap），
// 走 model.UpdateOption 链路写库，与 GUI 保存完全等价。
//
// 用法（在 gateway-new-api 容器内执行，共享 SQL_DSN 环境变量）：
//
//	/payment-setup --check              只读：当前凭据状态 + SDK 构造自检
//	/payment-setup --write /tmp/c.json  从 JSON 写入全部支付字段
//	/payment-setup --verify             客户端构造 + 回调 URL 可达性
//	/payment-setup --remove             清空全部 Wechat/Alipay options（回滚）
//
// 设计原则：
//   - 不替代 GUI：GUI 仍可调整任一字段，本工具仅是脚本化入口
//   - 凭据边界：工具不内置任何默认凭据，调用方必须显式传入 JSON
//   - 校验一致：走 model.UpdateOption（与 GUI 保存完全相同链路）
//   - 最小启动：跳过 HTTP/Redis/ClickHouse/i18n 等无关子系统
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
)

type wechatConfig struct {
	Enabled      bool    `json:"WechatEnabled"`
	MerchantId   string  `json:"WechatMerchantId"`
	AppId        string  `json:"WechatAppId"`
	ApiV3Key     string  `json:"WechatApiV3Key"`
	PrivateKey   string  `json:"WechatPrivateKey"`
	CertSerialNo string  `json:"WechatCertSerialNo"`
	NotifyUrl    string  `json:"WechatNotifyUrl"`
	ReturnUrl    string  `json:"WechatReturnUrl"`
	UnitPrice    float64 `json:"WechatUnitPrice"`
	MinTopUp     int     `json:"WechatMinTopUp"`
}

type alipayConfig struct {
	Enabled       bool    `json:"AlipayEnabled"`
	AppId         string  `json:"AlipayAppId"`
	PrivateKey    string  `json:"AlipayPrivateKey"`
	PublicKey     string  `json:"AlipayPublicKey"`
	UseCertMode   bool    `json:"AlipayUseCertMode"`
	AppCertPubKey string  `json:"AlipayAppCertPublicKey"`
	PublicCert    string  `json:"AlipayPublicCert"`
	RootCert      string  `json:"AlipayRootCert"`
	Sandbox       bool    `json:"AlipaySandbox"`
	NotifyUrl     string  `json:"AlipayNotifyUrl"`
	ReturnUrl     string  `json:"AlipayReturnUrl"`
	UnitPrice     float64 `json:"AlipayUnitPrice"`
	MinTopUp      int     `json:"AlipayMinTopUp"`
}

type paymentConfig struct {
	Wechat wechatConfig `json:"wechat"`
	Alipay alipayConfig `json:"alipay"`
}

// allKeys 列出本工具管理的全部 option key（与 model/option.go 注册保持一致）。
var allKeys = []string{
	"WechatEnabled", "WechatMerchantId", "WechatAppId", "WechatApiV3Key",
	"WechatPrivateKey", "WechatCertSerialNo", "WechatNotifyUrl", "WechatReturnUrl",
	"WechatUnitPrice", "WechatMinTopUp",
	"AlipayEnabled", "AlipayAppId", "AlipayPrivateKey", "AlipayPublicKey",
	"AlipayUseCertMode", "AlipayAppCertPublicKey", "AlipayPublicCert", "AlipayRootCert",
	"AlipaySandbox", "AlipayNotifyUrl", "AlipayReturnUrl",
	"AlipayUnitPrice", "AlipayMinTopUp",
}

func main() {
	var (
		cmdCheck  = flag.Bool("check", false, "仅检查当前 DB 状态与客户端构造")
		cmdWrite  = flag.String("write", "", "从 JSON 路径写入凭据")
		cmdRemove = flag.Bool("remove", false, "清空全部 Wechat/Alipay options")
		cmdVerify = flag.Bool("verify", false, "验证：客户端构造 + 回调 URL 可达")
	)
	flag.Parse()

	if err := bootstrap(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: 启动失败: %v\n", err)
		os.Exit(1)
	}

	switch {
	case *cmdCheck:
		runCheck()
	case *cmdWrite != "":
		runWrite(*cmdWrite)
	case *cmdRemove:
		runRemove()
	case *cmdVerify:
		runVerify()
	default:
		flag.Usage()
		os.Exit(2)
	}
}

// bootstrap 只初始化 DB 与 OptionMap，不启 HTTP / Redis / ClickHouse / i18n。
// 必需 env: SQL_DSN（docker exec 进 gateway-new-api 天然继承）。
func bootstrap() error {
	if os.Getenv("SQL_DSN") == "" {
		return fmt.Errorf("环境变量 SQL_DSN 未设置；请在 gateway-new-api 容器内执行本工具")
	}
	if err := model.InitDB(); err != nil {
		return fmt.Errorf("InitDB: %w", err)
	}
	// InitOptionMap 内部末尾会调用 loadOptionsFromDatabase，把 DB 真值覆盖进内存。
	model.InitOptionMap()
	return nil
}

func runCheck() {
	fmt.Println("=== [DB] options 表 Wechat*/Alipay* 当前状态 ===")
	for _, k := range allKeys {
		fmt.Printf("  %-22s = %s\n", k, maskValue(k, readOption(k)))
	}
	fmt.Println()
	fmt.Println("=== [InMemory] setting 内存状态 ===")
	fmt.Printf("  WechatEnabled  = %v\n", setting.WechatEnabled)
	fmt.Printf("  AlipayEnabled  = %v\n", setting.AlipayEnabled)
	fmt.Printf("  WechatMinTopUp = %d\n", setting.WechatMinTopUp)
	fmt.Printf("  AlipayMinTopUp = %d\n", setting.AlipayMinTopUp)
	fmt.Println()
	printSDKStatus()
}

func printSDKStatus() {
	fmt.Println("=== [SDK] 客户端构造测试 ===")
	if _, _, err := service.GetWechatClient(context.Background()); err != nil {
		fmt.Printf("  微信 client 构造失败: %v\n", err)
	} else {
		fmt.Println("  微信 client 构造成功（凭据完整）")
	}
	if _, err := service.GetAlipayClient(context.Background()); err != nil {
		fmt.Printf("  支付宝 client 构造失败: %v\n", err)
	} else {
		fmt.Println("  支付宝 client 构造成功（凭据完整）")
	}
}

func runWrite(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatal("读取配置失败", err)
	}
	var cfg paymentConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		fatal("JSON 解析失败", err)
	}
	fixPEMLiterals(&cfg)

	values := map[string]string{
		"WechatEnabled":          boolStr(cfg.Wechat.Enabled),
		"WechatMerchantId":       cfg.Wechat.MerchantId,
		"WechatAppId":            cfg.Wechat.AppId,
		"WechatApiV3Key":         cfg.Wechat.ApiV3Key,
		"WechatPrivateKey":       cfg.Wechat.PrivateKey,
		"WechatCertSerialNo":     cfg.Wechat.CertSerialNo,
		"WechatNotifyUrl":        cfg.Wechat.NotifyUrl,
		"WechatReturnUrl":        cfg.Wechat.ReturnUrl,
		"WechatUnitPrice":        fmt.Sprintf("%v", cfg.Wechat.UnitPrice),
		"WechatMinTopUp":         fmt.Sprintf("%d", cfg.Wechat.MinTopUp),
		"AlipayEnabled":          boolStr(cfg.Alipay.Enabled),
		"AlipayAppId":            cfg.Alipay.AppId,
		"AlipayPrivateKey":       cfg.Alipay.PrivateKey,
		"AlipayPublicKey":        cfg.Alipay.PublicKey,
		"AlipayUseCertMode":      boolStr(cfg.Alipay.UseCertMode),
		"AlipayAppCertPublicKey": cfg.Alipay.AppCertPubKey,
		"AlipayPublicCert":       cfg.Alipay.PublicCert,
		"AlipayRootCert":         cfg.Alipay.RootCert,
		"AlipaySandbox":          boolStr(cfg.Alipay.Sandbox),
		"AlipayNotifyUrl":        cfg.Alipay.NotifyUrl,
		"AlipayReturnUrl":        cfg.Alipay.ReturnUrl,
		"AlipayUnitPrice":        fmt.Sprintf("%v", cfg.Alipay.UnitPrice),
		"AlipayMinTopUp":         fmt.Sprintf("%d", cfg.Alipay.MinTopUp),
	}

	fmt.Printf("=== [Write] %s ===\n", path)
	n := 0
	for _, k := range allKeys {
		if err := model.UpdateOption(k, values[k]); err != nil {
			fatal("写入 "+k, err)
		}
		fmt.Printf("  ✓ %s\n", k)
		n++
	}
	fmt.Printf("\n共写入 %d 个字段（与 GUI 保存等价）\n", n)
	fmt.Println(strings.Repeat("─", 60))
	runVerify()
	fmt.Println(strings.Repeat("─", 60))
	abs, _ := filepath.Abs(path)
	fmt.Printf("安全提示：凭据文件请删除  rm -f %s\n", abs)
	fmt.Println("（本工具不自动删，避免误删）")
}

func runRemove() {
	fmt.Println("=== [Remove] 清空 Wechat/Alipay options ===")
	for _, k := range allKeys {
		if readOption(k) == "" {
			fmt.Printf("  - %s (空，跳过)\n", k)
			continue
		}
		if err := model.UpdateOption(k, ""); err != nil {
			fatal("清空 "+k, err)
		}
		fmt.Printf("  ✓ %s\n", k)
	}
	fmt.Println("\n注意：正在运行的网关进程需要重启或等待 options 同步周期后才能生效。")
}

func runVerify() {
	fmt.Println("=== [Verify] 凭据就位 + SDK 构造 + 回调 URL 可达 ===")
	printSDKStatus()
	fmt.Println()
	fmt.Println("=== [Reach] 回调 URL 可达性（HEAD 探测） ===")
	probeURL(setting.WechatNotifyUrl, "微信 notify")
	probeURL(setting.AlipayNotifyUrl, "支付宝 notify")
	if setting.WechatNotifyUrl == "" || setting.AlipayNotifyUrl == "" {
		fmt.Println("  提示：NotifyUrl 留空时系统使用 ServerAddress+/api/{wechat,alipay}/notify")
		fmt.Printf("  当前 ServerAddress = %q\n", readOption("ServerAddress"))
	}
}

// fixPEMLiterals 兜底：运维有时把 PEM 压成一行并用字面 \n 分隔。
func fixPEMLiterals(c *paymentConfig) {
	c.Wechat.PrivateKey = fixPEM(c.Wechat.PrivateKey)
	c.Alipay.PrivateKey = fixPEM(c.Alipay.PrivateKey)
	c.Alipay.PublicKey = fixPEM(c.Alipay.PublicKey)
	c.Alipay.AppCertPubKey = fixPEM(c.Alipay.AppCertPubKey)
	c.Alipay.PublicCert = fixPEM(c.Alipay.PublicCert)
	c.Alipay.RootCert = fixPEM(c.Alipay.RootCert)
}

func fixPEM(s string) string {
	if s == "" || strings.Contains(s, "BEGIN") {
		return s
	}
	return strings.ReplaceAll(s, `\n`, "\n")
}

func maskValue(key, v string) string {
	if v == "" {
		return "<empty>"
	}
	if strings.Contains(strings.ToLower(key), "key") || strings.Contains(key, "Secret") {
		if len(v) > 12 {
			return v[:8] + fmt.Sprintf("...(masked, %d bytes)", len(v))
		}
		return "***"
	}
	return v
}

func readOption(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return common.OptionMap[key]
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func probeURL(rawURL, label string) {
	if rawURL == "" {
		fmt.Printf("  - %s: URL 未配置（使用默认 ServerAddress）\n", label)
		return
	}
	if _, err := url.Parse(rawURL); err != nil {
		fmt.Printf("  ✗ %s %s 解析失败: %v\n", label, rawURL, err)
		return
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr, Timeout: 8 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		fmt.Printf("  ✗ %s 构造请求失败: %v\n", label, err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  ✗ %s %s 不可达: %v\n", label, rawURL, err)
		return
	}
	resp.Body.Close()
	// notify 端点无凭据 GET 通常返回 4xx/5xx，TCP/TLS 通即视为可达
	fmt.Printf("  ✓ %s %s 可达 (HTTP %d)\n", label, rawURL, resp.StatusCode)
}

func fatal(msg string, err error) {
	fmt.Fprintf(os.Stderr, "FATAL: %s: %v\n", msg, err)
	os.Exit(1)
}
