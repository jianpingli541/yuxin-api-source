package reliability

import (
	"time"

	"github.com/QuantumNous/new-api/setting/config"
)

// Setting holds the unified reliability configuration for the relay layer.
// All knobs default to values that preserve the pre-refactor behaviour: when
// Enabled is false the upstream request path is executed exactly as before
// (single client.Do call, no circuit breaker, no same-channel retry, no
// extra fallback).  Only when Enabled is true do the breaker / retry /
// fallback features activate, each governed by its own threshold.
type Setting struct {
	// Enabled is the master switch.  false == legacy behaviour (default).
	Enabled bool `json:"enabled"`

	// --- Circuit breaker ---
	// BreakerEnabled toggles per-channel circuit breaking.
	BreakerEnabled bool `json:"breaker_enabled"`
	// BreakerConsecutiveFailures: open the breaker after this many
	// consecutive failed upstream calls on the same channel.
	BreakerConsecutiveFailures int `json:"breaker_consecutive_failures"`
	// BreakerIntervalSeconds: cyclic window in the closed state; counts are
	// reset each interval.  0 disables the periodic reset.
	BreakerIntervalSeconds int `json:"breaker_interval_seconds"`
	// BreakerTimeoutSeconds: how long the breaker stays open before a
	// half-open probe is allowed.
	BreakerTimeoutSeconds int `json:"breaker_timeout_seconds"`
	// BreakerMaxRequests: maximum concurrent probes in the half-open state.
	BreakerMaxRequests int `json:"breaker_max_requests"`

	// --- Same-channel retry ---
	// RetryEnabled toggles same-channel retry of transient errors.
	RetryEnabled bool `json:"retry_enabled"`
	// RetryMaxAttempts is the total number of upstream attempts for one
	// relay request (1 == no retry).  Only transient errors are retried and
	// only when the request body can be safely reconstructed (GetBody set).
	RetryMaxAttempts int `json:"retry_max_attempts"`
	// RetryInitialBackoffMs is the first backoff delay; each subsequent
	// attempt doubles it, capped by RetryMaxBackoffMs, with jitter.
	RetryInitialBackoffMs int `json:"retry_initial_backoff_ms"`
	// RetryMaxBackoffMs caps the backoff delay.
	RetryMaxBackoffMs int `json:"retry_max_backoff_ms"`

	// --- Fallback ---
	// FallbackEnabled allows a channel whose breaker is open (or which
	// failed) to yield to the next channel via the existing distributor
	// priority retry loop.  The fast-fail error is marked retryable so the
	// outer controller/relay.go loop selects the next channel.
	FallbackEnabled bool `json:"fallback_enabled"`
}

// setting is the live configuration.  Defaults keep legacy behaviour.
var setting = Setting{
	Enabled:                    false,
	BreakerEnabled:             true,
	BreakerConsecutiveFailures: 5,
	BreakerIntervalSeconds:     60,
	BreakerTimeoutSeconds:      30,
	BreakerMaxRequests:         1,
	RetryEnabled:               true,
	RetryMaxAttempts:           2,
	RetryInitialBackoffMs:      200,
	RetryMaxBackoffMs:          2000,
	FallbackEnabled:            true,
}

func init() {
	config.GlobalConfig.Register("reliability", &setting)
}

// GetSetting returns the current reliability configuration.
func GetSetting() *Setting {
	return &setting
}

// Active reports whether any reliability feature is on.  Used by the chokepoint
// to fast-path the legacy behaviour when everything is off.
func Active() bool {
	return setting.Enabled
}

func breakerTimeout() time.Duration {
	if setting.BreakerTimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(setting.BreakerTimeoutSeconds) * time.Second
}

func breakerInterval() time.Duration {
	if setting.BreakerIntervalSeconds <= 0 {
		return 0
	}
	return time.Duration(setting.BreakerIntervalSeconds) * time.Second
}

func consecutiveFailures() uint32 {
	if setting.BreakerConsecutiveFailures <= 0 {
		return 5
	}
	return uint32(setting.BreakerConsecutiveFailures)
}

func maxHalfOpenRequests() uint32 {
	if setting.BreakerMaxRequests <= 0 {
		return 1
	}
	return uint32(setting.BreakerMaxRequests)
}

func maxAttempts() int {
	if setting.RetryMaxAttempts <= 1 {
		return 1
	}
	return setting.RetryMaxAttempts
}

func initialBackoff() time.Duration {
	if setting.RetryInitialBackoffMs <= 0 {
		return 200 * time.Millisecond
	}
	return time.Duration(setting.RetryInitialBackoffMs) * time.Millisecond
}

func maxBackoff() time.Duration {
	if setting.RetryMaxBackoffMs <= 0 {
		return 2 * time.Second
	}
	return time.Duration(setting.RetryMaxBackoffMs) * time.Millisecond
}
