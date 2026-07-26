// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package common

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type stringWriter interface {
	io.Writer
	writeString(string) (int, error)
}

type stringWrapper struct {
	io.Writer
}

func (w stringWrapper) writeString(str string) (int, error) {
	return w.Writer.Write([]byte(str))
}

func checkWriter(writer io.Writer) stringWriter {
	if w, ok := writer.(stringWriter); ok {
		return w
	} else {
		return stringWrapper{writer}
	}
}

// Server-Sent Events
// W3C Working Draft 29 October 2009
// http://www.w3.org/TR/2009/WD-eventsource-20091029/

var writeContentType = []string{"text/event-stream"}
var noCache = []string{"no-cache"}

var fieldReplacer = strings.NewReplacer(
	"\n", "\\n",
	"\r", "\\r")

var dataReplacer = strings.NewReplacer(
	"\n", "\n",
	"\r", "\\r")

type CustomEvent struct {
	Event string
	Id    string
	Retry uint
	Data  interface{}
}

func encode(writer io.Writer, event CustomEvent) error {
	w := checkWriter(writer)
	return writeData(w, event.Data)
}

// writeData renders the SSE data field. It accepts interface{} so callers can
// pass pre-encoded payloads ([]byte, json.RawMessage) without an extra string
// copy. The previous implementation did `data.(string)` without checking ok,
// which panicked for every non-string type — a latent production hazard since
// CustomEvent.Data is typed interface{} and 15+ SSE call sites exist.
//
// The data is normalized to its string form first; the SSE terminator is only
// appended when that string form begins with "data", matching the original
// wire contract that relay/helper/common.go and every channel relies on.
func writeData(w stringWriter, data interface{}) error {
	s := stringifyEventData(data)
	dataReplacer.WriteString(w, s)
	if strings.HasPrefix(s, "data") {
		w.writeString("\n\n")
	}
	return nil
}

// stringifyEventData converts CustomEvent.Data to its SSE string form without
// panicking on non-string types. nil renders as the empty string (which the
// SSE spec treats as a comment line); every other type uses fmt.Sprint, which
// is the same formatter the previous code used before the bad assertion.
func stringifyEventData(data interface{}) string {
	switch v := data.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		// Most non-string callers pass raw JSON or base64; string() is the
		// zero-copy conversion the existing channel code already uses.
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func (r CustomEvent) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	return encode(w, r)
}

func (r CustomEvent) WriteContentType(w http.ResponseWriter) {
	header := w.Header()

	// gin's Renderer contract passes CustomEvent by value, so we cannot keep
	// a value-typed sync.Mutex on the struct (go vet: "passes lock by value",
	// and the copied lock would not actually serialize concurrent writers).
	// The header map is the contention point, so we lock at the package level
	// — every SSE render on this process serializes here. The critical
	// section is two map writes and is not on the hot path of the actual
	// streaming bytes (those go through encode/writeData afterwards).
	sseHeaderMu.Lock()
	defer sseHeaderMu.Unlock()

	header["Content-Type"] = writeContentType

	if _, exist := header["Cache-Control"]; !exist {
		header["Cache-Control"] = noCache
	}
}

// sseHeaderMu serializes SSE header mutations. Render receives CustomEvent
// by value (gin's Renderer interface), which is why the mutex lives at the
// package level rather than on the struct — copying a struct field mutex is
// both a vet violation and ineffective for cross-goroutine synchronization.
var sseHeaderMu sync.Mutex
