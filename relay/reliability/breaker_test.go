package reliability

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/sony/gobreaker"
)

// helper to make a minimal *http.Request with a re-readable body.
func mustRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), "POST", "http://example.invalid/", strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	return req
}

// okCall returns a successful 200 response with no body.
func okCall() CallFn {
	return func() (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
}

// transientErr returns a network error shaped like what client.Do returns
// (*url.Error with Timeout() == true), which the classifier treats as
// transient.
func transientErr(msg string) error {
	return &url.Error{Op: "Post", URL: "http://example.invalid/", Err: errors.New(msg)}
}

func TestExecuteDisabledPreservesLegacyBehaviour(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.Enabled = false

	req := mustRequest(t, "hello")
	resp, err := Execute(context.Background(), 9001, req, okCall())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %v", resp)
	}
	if got := GetBreakerState(9001); got != gobreaker.StateClosed {
		t.Fatalf("expected breaker untouched when disabled, got %v", got)
	}
}

func TestExecuteRetrySuccess(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.Enabled = true
	setting.BreakerEnabled = true
	setting.RetryEnabled = true
	setting.RetryMaxAttempts = 3
	setting.BreakerConsecutiveFailures = 5
	ResetBreakers()

	calls := 0
	fn := func() (*http.Response, error) {
		calls++
		if calls < 3 {
			return nil, transientErr("connection refused")
		}
		return okCall()()
	}

	req := mustRequest(t, "payload")
	resp, err := Execute(context.Background(), 9002, req, fn)
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("unexpected response: %v", resp)
	}
	if got := GetBreakerState(9002); got != gobreaker.StateClosed {
		t.Fatalf("expected breaker closed (succeeded eventually), got %v", got)
	}
}

func TestExecuteRetryExhausted(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.Enabled = true
	setting.BreakerEnabled = true
	setting.RetryEnabled = true
	setting.RetryMaxAttempts = 2
	setting.BreakerConsecutiveFailures = 100 // do not trip during this test
	ResetBreakers()

	calls := 0
	fn := func() (*http.Response, error) {
		calls++
		return nil, transientErr("dial tcp: timeout")
	}
	req := mustRequest(t, "payload")

	resp, err := Execute(context.Background(), 9003, req, fn)
	if err == nil {
		t.Fatalf("expected an error after exhaustion")
	}
	if resp != nil {
		t.Fatalf("expected nil resp, got %v", resp)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
	var nerr *types.NewAPIError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *types.NewAPIError, got %T", err)
	}
}

func TestExecuteBreakerTripsAfterConsecutiveFailures(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.Enabled = true
	setting.BreakerEnabled = true
	setting.RetryEnabled = false
	setting.BreakerConsecutiveFailures = 2
	ResetBreakers()

	failFn := func() (*http.Response, error) {
		return nil, transientErr("boom")
	}

	// Two failing calls should trip the breaker.
	for i := 0; i < 2; i++ {
		req := mustRequest(t, "x")
		_, _ = Execute(context.Background(), 9004, req, failFn)
	}

	if got := GetBreakerState(9004); got != gobreaker.StateOpen {
		t.Fatalf("expected breaker open after %d consecutive failures, got %v", 2, got)
	}

	// Next call should fast-fail with a 503 retryable error.
	req := mustRequest(t, "x")
	resp, err := Execute(context.Background(), 9004, req, failFn)
	if err == nil {
		t.Fatalf("expected fast-fail error")
	}
	if resp != nil {
		t.Fatalf("expected nil resp on fast-fail, got %v", resp)
	}
	var nerr *types.NewAPIError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *types.NewAPIError, got %T", err)
	}
	if nerr.StatusCode != 503 {
		t.Fatalf("expected status 503 for retryable fast-fail, got %d", nerr.StatusCode)
	}
}

func TestExecuteNonTransientErrorNoRetry(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.Enabled = true
	setting.BreakerEnabled = true
	setting.RetryEnabled = true
	setting.RetryMaxAttempts = 4
	setting.BreakerConsecutiveFailures = 100
	ResetBreakers()

	// A non-*url.Error, non-net.Error error is NOT transient.
	calls := 0
	fn := func() (*http.Response, error) {
		calls++
		return nil, errors.New("business error")
	}
	req := mustRequest(t, "x")
	_, err := Execute(context.Background(), 9005, req, fn)
	if err == nil {
		t.Fatalf("expected an error")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 attempt for non-transient error, got %d", calls)
	}
}

func TestExecuteHTTP5xxRetried(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.Enabled = true
	setting.BreakerEnabled = true
	setting.RetryEnabled = true
	setting.RetryMaxAttempts = 3
	setting.BreakerConsecutiveFailures = 100
	ResetBreakers()

	calls := 0
	fn := func() (*http.Response, error) {
		calls++
		if calls < 2 {
			return &http.Response{
				StatusCode: 502,
				Status:     "502 Bad Gateway",
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		}
		return okCall()()
	}
	req := mustRequest(t, "x")
	resp, err := Execute(context.Background(), 9006, req, fn)
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExecuteNoRetryWithoutGetBody(t *testing.T) {
	old := setting
	defer func() { setting = old }()
	setting.Enabled = true
	setting.BreakerEnabled = true
	setting.RetryEnabled = true
	setting.RetryMaxAttempts = 4
	setting.BreakerConsecutiveFailures = 100
	ResetBreakers()

	// Request with no GetBody closure.
	req, err := http.NewRequestWithContext(context.Background(), "POST", "http://example.invalid/",
		io.NopCloser(strings.NewReader("body")))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	// Make sure GetBody is nil (NewRequest didn't set it for NopCloser).
	if req.GetBody != nil {
		t.Fatalf("precondition: GetBody must be nil")
	}

	calls := 0
	fn := func() (*http.Response, error) {
		calls++
		return nil, transientErr("timeout")
	}
	_, _ = Execute(context.Background(), 9007, req, fn)
	if calls != 1 {
		t.Fatalf("expected 1 attempt when GetBody is nil, got %d", calls)
	}
}
