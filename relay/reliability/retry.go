package reliability

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// Transient error classification.
//
// The retry policy uses two signals:
//
//  1. The Go error returned by client.Do (network-level failures).
//  2. The HTTP status code when client.Do returned a response (server-level
//     failures such as 5xx / 429).
//
// For (2) we deliberately reuse operation_setting.ShouldRetryByStatusCode so
// that the same-channel retry policy stays aligned with the cross-channel
// retry policy already enforced by controller/relay.go's shouldRetry helper.
// A divergence between the two would let the inner loop keep banging on a
// dead channel that the outer loop has already declared non-retryable, and
// would also require operators to maintain a second rule set.

// isTransientNetErr reports whether err is an upstream network error worth
// retrying (timeouts, connection refused, DNS failures).  Client-side
// cancellations (context.Canceled) are NOT transient: the client already
// gave up, retrying would be wasted work.
func isTransientNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		// client aborted; don't retry
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		// upstream timeout
		return true
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		// url.Error wraps dial / read / TLS / refused errors.
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout()
	}
	return false
}

// isTransientStatusCode reports whether the HTTP status code should trigger
// a same-channel retry.  Reuses the operation setting so this policy stays
// consistent with the existing cross-channel retry rule.
func isTransientStatusCode(code int) bool {
	return operation_setting.ShouldRetryByStatusCode(code)
}

// backoffRand guards a single rand source for jitter.  math/rand is fine for
// jitter; we don't need cryptographic randomness.
var (
	backoffRandMu sync.Mutex
	backoffRand   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// nextBackoff returns a backoff duration for the given attempt index
// (0-based, i.e. attempt == 0 is the first retry after the initial try).
// Exponential growth with full jitter, capped at maxBackoff().
func nextBackoff(attempt int) time.Duration {
	base := initialBackoff()
	max := maxBackoff()
	// doubling: base * 2^attempt
	exp := base << attempt
	if exp <= 0 || exp > max {
		exp = max
	}
	backoffRandMu.Lock()
	jitter := backoffRand.Int63n(int64(exp))
	backoffRandMu.Unlock()
	return time.Duration(jitter)
}
