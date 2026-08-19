package common

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"unsafe"

	"github.com/samber/lo"
)

const LocalLogContentLimit = 2048

// affCodeCharset 邀请码字符集(排除易混淆字符 0O1lI)
const affCodeCharset = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerateAffCode 生成邀请码: crypto/rand 随机 6 位(2026-08-04 安全修复,
// 替代 GetRandomString(4) 的 math/rand 弱随机与短长度, 降低碰撞与可预测性)。
func GenerateAffCode() string {
	chars := []byte(affCodeCharset)
	n := len(chars)
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败时回退到带随机种子的旧实现, 保证功能可用
		return GetRandomString(6)
	}
	for i := range buf {
		// 256 % 56 的微小偏斜可忽略(非密钥用途)
		buf[i] = chars[int(buf[i])%n]
	}
	return string(buf)
}

// 注册/改密表单校验正则 (2026-08-18 注册校验加强)。
var (
	passwordUpperRegex   = regexp.MustCompile(`[A-Z]`)
	passwordLowerRegex   = regexp.MustCompile(`[a-z]`)
	passwordDigitRegex   = regexp.MustCompile(`[0-9]`)
	passwordSpecialRegex = regexp.MustCompile(`[^A-Za-z0-9]`)
	usernameFormatRegex  = regexp.MustCompile(`^[A-Za-z0-9_-]{3,20}$`)
)

// ValidatePasswordStrength 密码强度: 8-20 位，且须同时包含大写字母、小写字母、数字与特殊符号。
// (2026-08-18 由测试期 >=6 位临时策略升级为正式策略，与前端 registerFormSchema 对齐)
func ValidatePasswordStrength(password string) bool {
	if len(password) < 8 || len(password) > 20 {
		return false
	}
	return passwordUpperRegex.MatchString(password) &&
		passwordLowerRegex.MatchString(password) &&
		passwordDigitRegex.MatchString(password) &&
		passwordSpecialRegex.MatchString(password)
}

// ValidateUsername 用户名格式: 3-20 位，仅字母、数字、下划线、连字符 (2026-08-18)。
func ValidateUsername(username string) bool {
	return usernameFormatRegex.MatchString(username)
}

// ValidateEmailFormat 邮箱格式校验 (2026-08-18)。
func ValidateEmailFormat(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsSafeBillingExpr 校验计费表达式是否含危险模式(2026-08-04 安全修复)。
// 表达式在前端经 new Function 执行, 服务端必须过滤任何可逃逸到任意 JS
// 的构造: 语句分隔、函数构造、全局对象访问、模板字面量、箭头函数等。
func IsSafeBillingExpr(expr string) bool {
	if strings.TrimSpace(expr) == "" {
		return false
	}
	dangerous := []string{
		";", "`", "$", "=>", "fetch(", "import(", "eval(", "Function",
		"XMLHttpRequest", "document.", "window.", "globalThis", "constructor",
		"require(", "process.", "navigator", "location.", "top.", "self.",
		"this.", "new ", "return ", "throw ",
	}
	for _, d := range dangerous {
		if strings.Contains(expr, d) {
			return false
		}
	}
	return true
}

// LocalLogPreview limits log-only content unless debug logging is enabled.
func LocalLogPreview(content string) string {
	if DebugEnabled || len(content) <= LocalLogContentLimit {
		return content
	}
	return fmt.Sprintf("%s... [truncated, original_length=%d, limit=%d]", content[:LocalLogContentLimit], len(content), LocalLogContentLimit)
}

func GetStringIfEmpty(str string, defaultValue string) string {
	if str == "" {
		return defaultValue
	}
	return str
}

func GetRandomString(length int) string {
	if length <= 0 {
		return ""
	}
	return lo.RandomString(length, lo.AlphanumericCharset)
}

func MapToJsonStr(m map[string]interface{}) string {
	bytes, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func StrToMap(str string) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := Unmarshal([]byte(str), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func StrToJsonArray(str string) ([]interface{}, error) {
	var js []interface{}
	err := json.Unmarshal([]byte(str), &js)
	if err != nil {
		return nil, err
	}
	return js, nil
}

func IsJsonArray(str string) bool {
	var js []interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}

func IsJsonObject(str string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}

func String2Int(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return num
}

func StringsContains(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}
	return false
}

// StringToByteSlice []byte only read, panic on append
func StringToByteSlice(s string) []byte {
	tmp1 := (*[2]uintptr)(unsafe.Pointer(&s))
	tmp2 := [3]uintptr{tmp1[0], tmp1[1], tmp1[1]}
	return *(*[]byte)(unsafe.Pointer(&tmp2))
}

func EncodeBase64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func GetJsonString(data any) string {
	if data == nil {
		return ""
	}
	b, _ := json.Marshal(data)
	return string(b)
}

// NormalizeBillingPreference clamps the billing preference to valid values.
func NormalizeBillingPreference(pref string) string {
	switch strings.TrimSpace(pref) {
	case "subscription_first", "wallet_first", "subscription_only", "wallet_only":
		return strings.TrimSpace(pref)
	default:
		return "subscription_first"
	}
}

// MaskEmail masks a user email to prevent PII leakage in logs
// Returns "***masked***" if email is empty, otherwise shows only the domain part
func MaskEmail(email string) string {
	if email == "" {
		return "***masked***"
	}

	// Find the @ symbol
	atIndex := strings.Index(email, "@")
	if atIndex == -1 {
		// No @ symbol found, return masked
		return "***masked***"
	}

	// Return only the domain part with @ symbol
	return "***@" + email[atIndex+1:]
}

// MaskSensitiveInfo moved to the conversion kit (kitutil) because the types
// package error formatting depends on it; host callers keep this name.
func MaskSensitiveInfo(str string) string {
	return kitutil.MaskSensitiveInfo(str)
}
