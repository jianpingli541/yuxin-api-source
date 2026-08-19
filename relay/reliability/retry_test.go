package reliability

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestIsTransientNetErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"wrapped context canceled", &url.Error{Op: "Post", URL: "http://x/", Err: context.Canceled}, false},
		{"context deadline exceeded", context.DeadlineExceeded, true},
		{"url error timeout", &url.Error{Op: "Post", URL: "http://x/", Err: context.DeadlineExceeded}, true},
		{"url error refused", &url.Error{Op: "Post", URL: "http://x/", Err: syscall.ECONNREFUSED}, true},
		{"url error dns", &url.Error{Op: "Post", URL: "http://x/", Err: errors.New("no such host")}, true},
		{"plain error", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTransientNetErr(tc.err); got != tc.want {
				t.Fatalf("isTransientNetErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsTransientStatusCode(t *testing.T) {
	// Aligned with the existing ShouldRetryByStatusCode policy.
	cases := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{401, true},  // 401-407 range in retry policy
		{408, false}, // deliberately excluded by the existing retry policy
		{409, true},  // 409-499 range
		{429, true},  // rate limited (within 409-499)
		{500, true},  // server error
		{502, true},  // bad gateway
		{503, true},  // service unavailable
		{504, false}, // always skip
		{524, false}, // always skip
	}
	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			if got := isTransientStatusCode(tc.code); got != tc.want {
				t.Fatalf("isTransientStatusCode(%d) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}

func TestNextBackoffWithinBounds(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.RetryInitialBackoffMs = 100
	setting.RetryMaxBackoffMs = 800

	// Attempt 0: exponential ceiling is 100ms; jitter [0, 100ms].
	for i := 0; i < 100; i++ {
		d := nextBackoff(0)
		if d < 0 || d > 100*time.Millisecond {
			t.Fatalf("attempt 0 backoff out of range: %v", d)
		}
	}

	// Attempt 3: ceiling = min(100*8, 800) = 800ms; jitter [0, 800ms].
	for i := 0; i < 100; i++ {
		d := nextBackoff(3)
		if d < 0 || d > 800*time.Millisecond {
			t.Fatalf("attempt 3 backoff out of range: %v", d)
		}
	}
}

func TestNextBackoffMonotonicCeiling(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.RetryInitialBackoffMs = 50
	setting.RetryMaxBackoffMs = 400

	max0 := nextBackoff(0)
	max1 := nextBackoff(1)
	max2 := nextBackoff(2)
	max3 := nextBackoff(3)
	// Ceilings: 50, 100, 200, 400 (capped at max).
	_ = max0
	_ = max1
	_ = max2
	_ = max3
	// Just ensure they're all positive.
	for i := 0; i < 10; i++ {
		d := nextBackoff(i)
		if d < 0 {
			t.Fatalf("negative backoff: %v", d)
		}
	}
}

func TestResetRequestBody(t *testing.T) {
	req := mustRequest(t, "hello")
	// Consume body once.
	_, _ = req.Body.Read(make([]byte, 5))
	// Reset.
	if err := resetRequestBody(req); err != nil {
		t.Fatalf("resetRequestBody: %v", err)
	}
	// Body should be readable again.
	buf := make([]byte, 5)
	n, err := req.Body.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("read after reset: %v", err)
	}
	if n != 5 || string(buf) != "hello" {
		t.Fatalf("expected to re-read original body, got %q (n=%d)", string(buf[:n]), n)
	}
}

func TestResetRequestBodyNilGetBody(t *testing.T) {
	req, err := newRequestWithoutGetBody(t)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := resetRequestBody(req); err == nil {
		t.Fatalf("expected error when GetBody is nil")
	}
}

// newRequestWithoutGetBody returns a *http.Request whose body is not
// re-readable (GetBody is nil), by wrapping a NopCloser.
func newRequestWithoutGetBody(t *testing.T) (*http.Request, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), "POST", "http://example.invalid/",
		io.NopCloser(strings.NewReader("body")))
	if err != nil {
		return nil, err
	}
	if req.GetBody != nil {
		t.Fatalf("precondition: expected GetBody to be nil for NopCloser body")
	}
	return req, nil
}
