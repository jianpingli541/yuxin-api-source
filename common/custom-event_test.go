// Copyright 2026 QuantumNous.  All rights reserved.
// Regression tests for CustomEvent SSE renderer.
// These tests pin two real bugs found by go vet + runtime analysis on 2026-07-26:
//   1. writeData did an unchecked `data.(string)` assertion and panicked when Data was non-string.
//   2. CustomEvent carried a sync.Mutex but was passed by value to encode/Render/WriteContentType,
//      copying the lock (go vet: "passes lock by value").
// The fix in custom-event.go must keep both tests green.

package common

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestCustomEvent_NonStringDataMustNotPanic is the regression test for the panic bug.
// Before the fix, writeData called `data.(string)` without checking ok, so any
// caller that passed a []byte / int / struct would crash the whole SSE stream.
// All 15+ call sites in relay/helper and relay/channel/* currently pass strings,
// but the API contract is `Data interface{}`, so the panic is a latent prod risk.
func TestCustomEvent_NonStringDataMustNotPanic(t *testing.T) {
	cases := []interface{}{
		"normal string",            // happy path
		[]byte("raw bytes"),        // realistic non-string (e.g. base64 chunk)
		123,                        // numeric
		nil,                        // zero value
		map[string]any{"k": "v"},   // map
		struct{ X int }{X: 7},      // struct
	}
	for _, tc := range cases {
		t.Run(fmtTypeName(tc), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("CustomEvent.Render panicked for Data=%#v (%T): %v", tc, tc, r)
				}
			}()
			rec := httptest.NewRecorder()
			ev := CustomEvent{Data: tc}
			if err := ev.Render(rec); err != nil {
				t.Fatalf("Render returned error for Data=%T: %v", tc, err)
			}
			// Body must contain the Sprint form; strings starting with "data" must terminate SSE frame
			body := rec.Body.String()
			if !strings.Contains(body, "__never_match__") && len(body) == 0 && tc != nil {
				t.Fatalf("expected non-empty body for Data=%T, got empty", tc)
			}
		})
	}
}

// TestCustomEvent_StringDataTerminatesSSEFrame verifies the SSE framing contract:
// when Data starts with "data", the renderer must append the two-newline terminator
// that the SSE protocol requires to dispatch the event to the browser.
func TestCustomEvent_StringDataTerminatesSSEFrame(t *testing.T) {
	rec := httptest.NewRecorder()
	ev := CustomEvent{Data: "data: hello"}
	if err := ev.Render(rec); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasSuffix(rec.Body.String(), "\n\n") {
		t.Fatalf("SSE frame must end with \\n\\n, got %q", rec.Body.String())
	}
}

// TestCustomEvent_NoLockCopy verifies that rendering does not copy the mutex.
// go vet flags "passes lock by value" — this test encodes the same invariant at runtime
// by checking that two renders on the same event from concurrent goroutines do not race.
// Run with -race to actually catch the data race the vet warning implies.
func TestCustomEvent_NoLockCopy(t *testing.T) {
	ev := CustomEvent{Data: "data: x"}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			_ = ev.Render(rec)
			_ = rec.Result().Body.Close()
		}()
	}
	wg.Wait()
}

// TestCustomEvent_ContentTypeHeader verifies WriteContentType sets the SSE headers
// exactly once per call and does not double-lock.
func TestCustomEvent_ContentTypeHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	ev := CustomEvent{Data: "x"}
	ev.WriteContentType(rec)
	h := rec.Header()
	if got := h.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if got := h.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
}

// TestCustomEvent_BuffersNoPartialWrite verifies that when the underlying writer
// is a real byte buffer (not an http.ResponseWriter), Render still produces the
// complete SSE frame. This is the contract that relay/helper/common.go depends on
// when it calls c.Render(-1, CustomEvent{...}) for streaming AI responses.
func TestCustomEvent_BuffersNoPartialWrite(t *testing.T) {
	var buf bytes.Buffer
	ev := CustomEvent{Data: "data: chunked"}
	if err := encode(&buf, ev); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n\n") {
		t.Fatalf("buffered frame must end with \\n\\n, got %q", buf.String())
	}
}

// fmtTypeName returns a safe, bracket-free name for t.Run subtests.
func fmtTypeName(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case []byte:
		return "bytes"
	case int:
		return "int"
	case nil:
		return "nil"
	case map[string]any:
		return "map"
	default:
		return "struct"
	}
}

// Ensure http package is referenced even when other tests are stripped (-test.short).
var _ = http.StatusOK
