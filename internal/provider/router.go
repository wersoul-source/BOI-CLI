package provider

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

type ProviderStats struct {
	Name               string
	CallCount          int
	SuccessCount       int
	FailCount          int
	RetryCount         int
	RateLimitRemaining int
	RateLimitTotal     int
}

func (s *ProviderStats) UsagePct() int {
	if s.RateLimitTotal <= 0 {
		total := s.SuccessCount + s.FailCount
		if total == 0 {
			return 100
		}
		return 100 - (s.FailCount*100)/total
	}
	pct := (s.RateLimitRemaining * 100) / s.RateLimitTotal
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}
func (s *ProviderStats) IsExhausted() bool { return s.UsagePct() < 5 }
func (s *ProviderStats) AvailablePct() int { return s.UsagePct() }

type RouterOptions struct {
	MaxRetries  int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

type Router struct {
	providers   []Provider
	stats       []*ProviderStats
	activeIndex int
	options     RouterOptions
	sleep       func(context.Context, time.Duration) error
	mu          sync.RWMutex
}

func NewRouter(providers []Provider) *Router {
	return NewRouterWithOptions(providers, RouterOptions{MaxRetries: 1, BaseBackoff: 100 * time.Millisecond, MaxBackoff: 2 * time.Second})
}

func NewRouterWithOptions(providers []Provider, options RouterOptions) *Router {
	if options.MaxRetries < 0 {
		options.MaxRetries = 0
	}
	if options.BaseBackoff < 0 {
		options.BaseBackoff = 0
	}
	if options.MaxBackoff <= 0 {
		options.MaxBackoff = 2 * time.Second
	}
	stats := make([]*ProviderStats, len(providers))
	for i, p := range providers {
		stats[i] = &ProviderStats{Name: p.Name()}
	}
	return &Router{providers: providers, stats: stats, options: options, sleep: sleepContext}
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	order := r.buildOrderLocked()
	r.mu.RUnlock()
	var lastErr error
	for _, idx := range order {
		p := r.providers[idx]
		for attempt := 0; attempt <= r.options.MaxRetries; attempt++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			resp, err := p.Complete(ctx, req)
			if err == nil && resp == nil {
				err = &ProviderError{Provider: p.Name(), Class: ErrorUnavailable, Message: "Provider returned an empty response contract"}
			}
			r.recordAttempt(idx, resp, err, attempt > 0)
			if err == nil {
				r.mu.Lock()
				r.activeIndex = idx
				r.mu.Unlock()
				return resp, nil
			}
			class := ClassifyError(err)
			lastErr = err
			slog.Warn("provider failed", "name", p.Name(), "class", class, "attempt", attempt+1)
			if class == ErrorCancelled {
				return nil, err
			}
			if !retryable(class) || attempt == r.options.MaxRetries {
				break
			}
			if err := r.sleep(ctx, r.backoff(err, attempt)); err != nil {
				return nil, err
			}
		}
		if !failoverAllowed(ClassifyError(lastErr)) {
			return nil, fmt.Errorf("provider %s: %w", p.Name(), lastErr)
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers configured")
}

func (r *Router) recordAttempt(idx int, resp *CompletionResponse, err error, retry bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.stats[idx]
	s.CallCount++
	if retry {
		s.RetryCount++
	}
	if err != nil {
		s.FailCount++
		return
	}
	s.SuccessCount++
	if resp != nil {
		if resp.RateLimitRemaining > 0 {
			s.RateLimitRemaining = resp.RateLimitRemaining
		}
		if resp.RateLimitTotal > 0 {
			s.RateLimitTotal = resp.RateLimitTotal
		}
	}
}

func (r *Router) backoff(err error, attempt int) time.Duration {
	duration := r.options.BaseBackoff << attempt
	if providerErr, ok := err.(*ProviderError); ok && providerErr.RetryAfter > duration {
		duration = providerErr.RetryAfter
	}
	if duration > r.options.MaxBackoff {
		return r.options.MaxBackoff
	}
	return duration
}

func (r *Router) buildOrderLocked() []int {
	indices := make([]int, len(r.providers))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(a, b int) bool {
		ai, bi := indices[a], indices[b]
		ae, be := r.stats[ai].IsExhausted(), r.stats[bi].IsExhausted()
		if ae != be {
			return !ae
		}
		if ai == r.activeIndex {
			return true
		}
		if bi == r.activeIndex {
			return false
		}
		return r.stats[ai].AvailablePct() > r.stats[bi].AvailablePct()
	})
	return indices
}

func (r *Router) ProviderCount() int { return len(r.providers) }
func (r *Router) ProviderNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}
func (r *Router) ActiveProvider() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.activeIndex >= 0 && r.activeIndex < len(r.providers) {
		return r.providers[r.activeIndex].Name()
	}
	return ""
}
func (r *Router) SetActiveProvider(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.providers {
		if p.Name() == name {
			r.activeIndex = i
			return true
		}
	}
	return false
}
func (r *Router) Stats() []*ProviderStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ProviderStats, len(r.stats))
	for i, s := range r.stats {
		copied := *s
		result[i] = &copied
	}
	return result
}
func (r *Router) ActiveStats() *ProviderStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.activeIndex >= 0 && r.activeIndex < len(r.stats) {
		copied := *r.stats[r.activeIndex]
		return &copied
	}
	return nil
}
func ModelFor(name string) string { return "" }
