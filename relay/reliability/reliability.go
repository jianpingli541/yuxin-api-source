// Package reliability adds unified reliability capabilities (circuit breaker,
// same-channel retry with exponential backoff and jitter, and fallback via
// the existing distributor priority loop) to the relay request path.
//
// The layer is fully config-gated and defaults to off, so when Enabled is
// false the chokepoint can call straight through to the underlying HTTP call
// without touching this package at all (preserving the pre-refactor
// behaviour).
//
// Design notes:
//
//   - The breaker is per-channel (keyed by channel id).  Failure counting is
//     driven manually via TwoStepCircuitBreaker.Allow so that the breaker
//     observes HTTP status codes (5xx / 429) in addition to network errors.
//   - Same-channel retry only fires when the request body can be safely
//     re-sent (req.GetBody non-nil).  http.NewRequest populates GetBody only
//     for bodies backed by *bytes.Buffer / *bytes.Reader / *strings.Reader,
//     and DoTaskApiRequest sets it explicitly.  For other callers the inner
//     retry is a no-op and the outer controller/relay.go retry loop still
//     applies.
//   - Billing is NOT coupled to doRequest: PostConsumeQuota runs after a
//     successful DoResponse in the controller / helper layer, so a failed
//     retry attempt never debits quota.
package reliability

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relaykit/types"
)

// CallFn performs a single upstream HTTP attempt.
type CallFn func() (*http.Response, error)

// Execute runs fn through the configured breaker + retry policy.
//
//   - ctx: request context, used to honor client cancellation during backoff.
//   - channelId: upstream channel id; the breaker key.
//   - req: the upstream *http.Request.  When req.GetBody is non-nil the
//     orchestrator resets req.Body to a fresh reader before each retry so
//     the same body is faithfully re-sent.  When GetBody is nil only one
//     attempt is made.
//   - fn: the actual HTTP call; must use the supplied *http.Request.
//
// Return semantics match the legacy doRequest contract:
//   - On final success: (resp, nil); resp may carry any HTTP status.
//   - On final failure: (nil, *types.NewAPIError).
func Execute(ctx context.Context, channelId int, req *http.Request, fn CallFn) (*http.Response, error) {
	if !Active() || req == nil {
		// Legacy behaviour: single attempt, no breaker / retry.
		return fn()
	}

	var done func(bool)
	if setting.BreakerEnabled {
		cb := getBreaker(channelId)
		d, err := cb.Allow()
		if err != nil {
			// Breaker open: fast-fail with a retryable error so the outer
			// controller/relay.go loop falls back to the next channel.
			logger.LogWarn(ctx, "reliability: circuit breaker open for channel, fast-failing")
			return nil, newBreakerFastFailError()
		}
		done = d
	}

	resp, err := executeRetryLoop(ctx, req, fn)
	if done != nil {
		done(classifyBreakerSuccess(resp, err))
	}
	return resp, err
}

// executeRetryLoop implements bounded same-channel retry with exponential
// backoff and jitter.  It returns the legacy error shape: a *NewAPIError
// wrapping the underlying error, or the final http.Response.
func executeRetryLoop(ctx context.Context, req *http.Request, fn CallFn) (*http.Response, error) {
	attempts := 1
	if setting.RetryEnabled {
		attempts = maxAttempts()
	}
	canRetryBody := req.GetBody != nil

	var resp *http.Response
	var err error

	for attemptIdx := 0; attemptIdx < attempts; attemptIdx++ {
		resp, err = fn()
		transient := classifyTransient(resp, err)
		if !transient || attemptIdx == attempts-1 {
			break
		}
		if !canRetryBody {
			logger.LogDebug(ctx, "reliability: transient failure but request body is not re-readable; skipping same-channel retry")
			break
		}
		// Discard the previous attempt's response before retrying.
		if resp != nil {
			drainAndClose(resp)
		}
		if resetErr := resetRequestBody(req); resetErr != nil {
			logger.LogWarn(ctx, "reliability: failed to reset request body for retry: "+resetErr.Error())
			break
		}
		delay := nextBackoff(attemptIdx)
		logger.LogDebug(ctx, "reliability: transient failure, retrying same channel after backoff")
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeDoRequestFailed,
			types.ErrOptionWithHideErrMsg("upstream error: do request failed"))
	}
	return resp, nil
}

// classifyTransient reports whether the outcome of a single attempt should
// trigger a same-channel retry.
func classifyTransient(resp *http.Response, err error) bool {
	if err != nil {
		return isTransientNetErr(err)
	}
	if resp != nil {
		return isTransientStatusCode(resp.StatusCode)
	}
	return false
}

// classifyBreakerSuccess reports whether the final outcome counts as a
// breaker success (i.e. the channel is healthy).  Network errors and
// retryable HTTP statuses count as failures; everything else is success.
func classifyBreakerSuccess(resp *http.Response, err error) bool {
	if err != nil {
		return false
	}
	if resp != nil && isTransientStatusCode(resp.StatusCode) {
		return false
	}
	return true
}

// drainAndClose drains up to a small cap and closes the response body so the
// underlying connection can be reused.
func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.CopyN(io.Discard, resp.Body, 4096)
	_ = resp.Body.Close()
}

// resetRequestBody replaces req.Body with a fresh reader from req.GetBody().
func resetRequestBody(req *http.Request) error {
	if req.GetBody == nil {
		return errors.New("request body not re-readable")
	}
	body, err := req.GetBody()
	if err != nil {
		return err
	}
	req.Body = body
	return nil
}

// newBreakerFastFailError builds the error returned when the breaker is
// open.  Status 503 falls inside the AutomaticRetryStatusCodeRanges
// (500-503 inclusive), so the outer loop treats it as retryable and falls
// back to the next channel.
func newBreakerFastFailError() *types.NewAPIError {
	return types.NewError(errors.New("reliability: upstream channel circuit breaker is open"),
		types.ErrorCodeDoRequestFailed,
		types.ErrOptionWithStatusCode(http.StatusServiceUnavailable),
		types.ErrOptionWithHideErrMsg("upstream error: do request failed"))
}
