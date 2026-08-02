package reliability

import (
	"context"
	"fmt"
	"sync"

	"github.com/QuantumNous/new-api/logger"
	"github.com/sony/gobreaker"
)

// Breaker registry.
//
// One circuit breaker per upstream channel id.  State is in-process (like the
// existing per-channel health caches); it resets on restart which is an
// acceptable trade-off for a v1 relay reliability layer and keeps this
// feature free of any external dependency (no Redis, no DB schema change).

var (
	breakerMu sync.RWMutex
	breakers  = make(map[int]*gobreaker.TwoStepCircuitBreaker)
)

// getBreaker returns (creating if necessary) the circuit breaker for a
// channel.  Settings are snapshotted at creation time; changing config at
// runtime only affects breakers created afterwards.  This matches the
// existing in-process config semantics elsewhere in the codebase.
func getBreaker(channelId int) *gobreaker.TwoStepCircuitBreaker {
	breakerMu.RLock()
	cb, ok := breakers[channelId]
	breakerMu.RUnlock()
	if ok {
		return cb
	}
	breakerMu.Lock()
	defer breakerMu.Unlock()
	if cb, ok := breakers[channelId]; ok {
		return cb
	}
	cb = newBreaker(channelId)
	breakers[channelId] = cb
	return cb
}

func newBreaker(channelId int) *gobreaker.TwoStepCircuitBreaker {
	name := fmt.Sprintf("channel-%d", channelId)
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: maxHalfOpenRequests(),
		Interval:    breakerInterval(),
		Timeout:     breakerTimeout(),
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= consecutiveFailures()
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.LogWarn(context.Background(),
				fmt.Sprintf("reliability: circuit breaker %s %s -> %s (channel #%d)", name, from, to, channelId))
		},
	}
	return gobreaker.NewTwoStepCircuitBreaker(st)
}

// ResetBreakers clears all breakers.  Intended for tests and hot-reload.
func ResetBreakers() {
	breakerMu.Lock()
	defer breakerMu.Unlock()
	breakers = make(map[int]*gobreaker.TwoStepCircuitBreaker)
}

// GetBreakerState reports the state of a channel's breaker (or StateClosed
// if the breaker does not exist yet).  Exposed for observability and tests.
func GetBreakerState(channelId int) gobreaker.State {
	breakerMu.RLock()
	cb, ok := breakers[channelId]
	breakerMu.RUnlock()
	if !ok {
		return gobreaker.StateClosed
	}
	return cb.State()
}
