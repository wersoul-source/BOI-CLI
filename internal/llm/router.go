package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
)

// ProviderStats tracks usage and rate limit for a provider.
type ProviderStats struct {
	Name               string
	CallCount          int
	SuccessCount       int
	FailCount          int
	RateLimitRemaining int
	RateLimitTotal     int
}

// UsagePct returns estimated remaining capacity as percentage (0-100).
// Returns 100 if rate limit is unknown.
func (s *ProviderStats) UsagePct() int {
	if s.RateLimitTotal <= 0 {
		// Unknown rate limit — estimate from success/fail
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

// IsExhausted returns true if the provider is nearly out of capacity (<5%).
func (s *ProviderStats) IsExhausted() bool {
	return s.UsagePct() < 5
}

// AvailablePct returns remaining percentage for display.
func (s *ProviderStats) AvailablePct() int {
	pct := s.UsagePct()
	return pct
}

type Router struct {
	providers     []Provider
	stats         []*ProviderStats
	activeIndex   int
	mu            sync.RWMutex
}

func NewRouter(providers []Provider) *Router {
	stats := make([]*ProviderStats, len(providers))
	for i, p := range providers {
		stats[i] = &ProviderStats{Name: p.Name()}
	}
	return &Router{
		providers:   providers,
		stats:       stats,
		activeIndex: 0,
	}
}

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	r.mu.RLock()
	// Try active provider first, then prefer non-exhausted ones
	order := r.buildOrder()
	r.mu.RUnlock()

	var lastErr error
	for _, idx := range order {
		if idx < 0 || idx >= len(r.providers) {
			continue
		}
		p := r.providers[idx]
		s := r.stats[idx]

		slog.Debug("trying provider", "name", p.Name(), "available", s.AvailablePct())

		resp, err := p.Complete(ctx, req)
		s.CallCount++

		if err != nil {
			s.FailCount++
			slog.Warn("provider failed", "name", p.Name(), "error", err)
			if isHTTPError(err) {
				lastErr = err
				continue
			}
			return nil, fmt.Errorf("provider %s: %w", p.Name(), err)
		}

		s.SuccessCount++
		if resp.RateLimitRemaining > 0 {
			s.RateLimitRemaining = resp.RateLimitRemaining
		}
		if resp.RateLimitTotal > 0 {
			s.RateLimitTotal = resp.RateLimitTotal
		}

		// Track which provider was used
		r.mu.Lock()
		for i := range r.stats {
			if r.stats[i].Name == p.Name() {
				r.activeIndex = i
				break
			}
		}
		r.mu.Unlock()

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all providers exhausted, last error: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers configured")
}

// buildOrder returns provider indices: active first, then by availability.
func (r *Router) buildOrder() []int {
	indices := make([]int, len(r.providers))
	for i := range r.providers {
		indices[i] = i
	}

	// Sort: prefer non-exhausted, then by highest availability
	sort.SliceStable(indices, func(a, b int) bool {
		aExhausted := r.stats[a].IsExhausted()
		bExhausted := r.stats[b].IsExhausted()

		if !aExhausted && bExhausted {
			return true
		}
		if aExhausted && !bExhausted {
			return false
		}
		// Both same exhaustion status — prefer active
		if indices[a] == r.activeIndex {
			return true
		}
		if indices[b] == r.activeIndex {
			return false
		}
		// Higher availability first
		return r.stats[a].AvailablePct() > r.stats[b].AvailablePct()
	})

	return indices
}

func isHTTPError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	errorPatterns := []string{
		"status 404", "status 429", "status 500", "status 502", "status 503", "status 504",
		"internal server error", "service unavailable", "bad gateway", "gateway timeout",
		"rate limit", "connection refused", "context deadline exceeded",
	}
	for _, pattern := range errorPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

func (r *Router) ProviderCount() int {
	return len(r.providers)
}

func (r *Router) ProviderNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}

// ActiveProvider returns the currently active provider name.
func (r *Router) ActiveProvider() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.activeIndex >= 0 && r.activeIndex < len(r.providers) {
		return r.providers[r.activeIndex].Name()
	}
	return ""
}

// SetActiveProvider switches the preferred provider by name.
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

// Stats returns all provider stats for display.
func (r *Router) Stats() []*ProviderStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ProviderStats, len(r.stats))
	copy(result, r.stats)
	return result
}

// ActiveStats returns stats for the currently active provider.
func (r *Router) ActiveStats() *ProviderStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.activeIndex >= 0 && r.activeIndex < len(r.stats) {
		s := *r.stats[r.activeIndex]
		return &s
	}
	return nil
}

// ModelFor returns the model name configured for the given provider.
func ModelFor(name string) string {
	// Placeholder — model is per-provider from PSC config
	// This can be extended to load from config.yaml
	return ""
}
