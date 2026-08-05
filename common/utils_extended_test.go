package common

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestBytes2Size(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1024 B"},          // 边界: num/1024 == 1 不触发 KB 分支
		{2048, "2 KB"},
		{1500, "1500 B"},
		{2097152, "2 MB"},
		{1073741824, "1024 MB"},   // 1GB 不触发 GB 分支,渲染为 MB
		{2147483648, "2.00 GB"},
	}
	for _, tt := range tests {
		got := Bytes2Size(tt.in)
		if got != tt.want {
			t.Errorf("Bytes2Size(%d): got %q want %q", tt.in, got, tt.want)
		}
	}
}

func TestSeconds2Time(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0 秒"},
		{59, "59 秒"},
		{60, "1 分钟 0 秒"},
		{125, "2 分钟 5 秒"},
		{3600, "1 小时 0 秒"}, // 零值分支被跳过
		{3725, "1 小时 2 分钟 5 秒"},
		{86400, "1 天 0 秒"},
	}
	for _, tt := range tests {
		got := Seconds2Time(tt.in)
		if got != tt.want {
			t.Errorf("Seconds2Time(%d): got %q want %q", tt.in, got, tt.want)
		}
	}
}

func TestInterface2String(t *testing.T) {
	tests := []struct {
		in   interface{}
		want string
	}{
		{"plain", "plain"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{nil, ""},
	}
	for _, tt := range tests {
		got := Interface2String(tt.in)
		if got != tt.want {
			t.Errorf("Interface2String(%v): got %s want %s", tt.in, got, tt.want)
		}
	}
}

func TestInterface2String_FallbackToFmt(t *testing.T) {
	type custom struct{ X int }
	got := Interface2String(custom{X: 1})
	if !strings.Contains(got, "{1}") {
		t.Errorf("expected fallback to %%v, got %s", got)
	}
}

func TestUnescapeHTML(t *testing.T) {
	got := UnescapeHTML("<a>x</a>")
	if _, ok := got.(template.HTML); !ok {
		t.Fatalf("not a template.HTML: %T", got)
	}
}

func TestIntMax(t *testing.T) {
	if IntMax(1, 2) != 2 {
		t.Fatal("IntMax(1,2) != 2")
	}
	if IntMax(5, 5) != 5 {
		t.Fatal("IntMax(5,5) != 5")
	}
	if IntMax(-1, -2) != -1 {
		t.Fatal("IntMax(-1,-2) != -1")
	}
}

func TestGetUUID_Format(t *testing.T) {
	u := GetUUID()
	if len(u) != 32 {
		t.Fatalf("UUID len: got %d want 32 (%s)", len(u), u)
	}
	matched, _ := regexp.MatchString(`^[0-9a-f]{32}$`, u)
	if !matched {
		t.Fatalf("not lowercase hex: %s", u)
	}
	if GetUUID() == u {
		t.Fatal("UUID should be unique")
	}
}

func TestGenerateRandomCharsKey(t *testing.T) {
	for _, n := range []int{0, 5, 16, 64} {
		k, err := GenerateRandomCharsKey(n)
		if err != nil {
			t.Fatal(err)
		}
		if len(k) != n {
			t.Fatalf("GenerateRandomCharsKey(%d): len=%d", n, len(k))
		}
	}
}

func TestGenerateRandomKey_Base64(t *testing.T) {
	k, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatal(err)
	}
	// 32 -> bytes 24 -> base64 ~32 chars
	if len(k) < 16 {
		t.Fatalf("hex len: %d want >=16", len(k))
	}
	// should be valid base64
	for _, c := range k {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '+' || c == '/' || c == '=') {
			t.Fatalf("non-base64 char: %s", k)
			break
		}
	}
}

func TestGenerateKey_Length48(t *testing.T) {
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(k) != 48 {
		t.Fatalf("GenerateKey len: got %d want 48", len(k))
	}
	k2, _ := GenerateKey()
	if k2 == k {
		t.Fatal("GenerateKey should be unique")
	}
}

func TestGetRandomInt_Bounds(t *testing.T) {
	for _, max := range []int{1, 10, 1000} {
		v := GetRandomInt(max)
		if v < 0 || v >= max {
			t.Fatalf("GetRandomInt(%d): %d out of bounds", max, v)
		}
	}
}

func TestGetTimestamp_Monotonic(t *testing.T) {
	t1 := GetTimestamp()
	t2 := GetTimestamp()
	if t2 < t1 {
		t.Fatal("timestamp went backwards")
	}
}

func TestGetTimeString_Format(t *testing.T) {
	s := GetTimeString()
	// format is "20060102150405" + 9-digit nanosecond tail = 23 chars
	if len(s) != 23 {
		t.Fatalf("time string len: got %d want 23 (%s)", len(s), s)
	}
	matched, _ := regexp.MatchString(`^[0-9]{23}$`, s)
	if !matched {
		t.Fatalf("not 23 digits: %s", s)
	}
}

func TestNewRequestId_LengthAndUniqueness(t *testing.T) {
	rid := NewRequestId()
	if len(rid) < 30 {
		t.Fatalf("RequestId too short: got %d (%s)", len(rid), rid)
	}
	if rid == NewRequestId() {
		t.Fatal("RequestId should be unique")
	}
}

func TestMax(t *testing.T) {
	if Max(1, 2) != 2 {
		t.Fatal("Max(1,2)")
	}
	if Max(-5, -1) != -1 {
		t.Fatal("Max(-5,-1)")
	}
	if Max(0, 0) != 0 {
		t.Fatal("Max(0,0)")
	}
}

func TestMessageWithRequestId(t *testing.T) {
	got := MessageWithRequestId("hello", "abc123")
	if got != "hello (request id: abc123)" {
		t.Fatalf("got %q", got)
	}
}

func TestGetPointer(t *testing.T) {
	v := 42
	p := GetPointer(v)
	if p == nil || *p != 42 {
		t.Fatal("GetPointer round-trip failed")
	}
}

func TestAny2Type_JSONSuccess(t *testing.T) {
	type Bar struct {
		X int `json:"x"`
	}
	in := map[string]any{"x": 42}
	got, err := Any2Type[Bar](in)
	if err != nil {
		t.Fatal(err)
	}
	if got.X != 42 {
		t.Fatalf("got %+v", got)
	}
}

func TestAny2Type_InvalidJSON(t *testing.T) {
	type Bar struct {
		X int
	}
	got, err := Any2Type[Bar](func() {})
	if err == nil {
		t.Fatalf("expected error, got %+v", got)
	}
}

func TestBuildURL_StripsAndConcatenates(t *testing.T) {
	got := BuildURL("https://api.example.com/", "/v1/models")
	if got != "https://api.example.com/v1/models" {
		t.Fatalf("BuildURL: %s", got)
	}
}

func TestBuildURL_NoSlash(t *testing.T) {
	got := BuildURL("https://api.example.com", "v1/models")
	if got != "https://api.example.com/v1/models" {
		t.Fatalf("BuildURL: %s", got)
	}
}

func TestBuildURL_EmptyEndpoint(t *testing.T) {
	got := BuildURL("https://api.example.com/", "")
	if got != "https://api.example.com/" {
		t.Fatalf("BuildURL empty endpoint: %s", got)
	}
}

func TestSaveTmpFile_WritesContent(t *testing.T) {
	path, err := SaveTmpFile("test_save", strings.NewReader("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("empty path")
	}
	defer os.Remove(path)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Fatalf("content: %s", string(data))
	}
}

func TestIsRunningInContainer_NoPanic(t *testing.T) {
	_ = IsRunningInContainer()
}

func TestRandomSleep_NoPanic(t *testing.T) {
	RandomSleep()
}

// ensure json import is used
var _ = json.Marshal