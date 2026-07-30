package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type Router struct {
	providers []Provider
}

func NewRouter(providers []Provider) *Router {
	return &Router{providers: providers}
}

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	var lastErr error
	for _, p := range r.providers {
		slog.Debug("trying provider", "name", p.Name())
		resp, err := p.Complete(ctx, req)
		if err != nil {
			slog.Warn("provider failed", "name", p.Name(), "error", err)
			if isHTTPError(err) {
				lastErr = err
				continue
			}
			return nil, fmt.Errorf("provider %s: %w", p.Name(), err)
		}
		return resp, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers exhausted, last error: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers configured")
}

func isHTTPError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	errorPatterns := []string{
		"status 404",
		"status 429",
		"status 500",
		"status 502",
		"status 503",
		"status 504",
		"internal server error",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"rate limit",
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
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}
