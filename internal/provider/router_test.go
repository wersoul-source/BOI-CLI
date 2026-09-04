package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type scriptedProvider struct {
	name        string
	errors      []error
	calls       int
	nilResponse bool
}

func (p *scriptedProvider) Name() string { return p.name }
func (p *scriptedProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	p.calls++
	if p.calls <= len(p.errors) && p.errors[p.calls-1] != nil {
		return nil, p.errors[p.calls-1]
	}
	if p.nilResponse {
		return nil, nil
	}
	return &CompletionResponse{Content: p.name, Provider: p.name}, nil
}
func (p *scriptedProvider) Stream(context.Context, CompletionRequest) (<-chan Token, error) {
	return nil, errors.New("unused")
}

func TestRouterRetriesTransientProvider(t *testing.T) {
	p := &scriptedProvider{name: "one", errors: []error{&ProviderError{Provider: "one", Class: ErrorTransient}}}
	r := NewRouterWithOptions([]Provider{p}, RouterOptions{MaxRetries: 1})
	resp, err := r.Complete(context.Background(), CompletionRequest{})
	if err != nil || resp.Provider != "one" {
		t.Fatalf("unexpected response=%v err=%v", resp, err)
	}
	stats := r.Stats()[0]
	if stats.CallCount != 2 || stats.RetryCount != 1 || stats.FailCount != 1 || stats.SuccessCount != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestRouterFailsOverAfterBoundedRetries(t *testing.T) {
	transient := &ProviderError{Provider: "one", Class: ErrorUnavailable}
	one := &scriptedProvider{name: "one", errors: []error{transient, transient}}
	two := &scriptedProvider{name: "two"}
	r := NewRouterWithOptions([]Provider{one, two}, RouterOptions{MaxRetries: 1})
	resp, err := r.Complete(context.Background(), CompletionRequest{})
	if err != nil || resp.Provider != "two" {
		t.Fatalf("unexpected response=%v err=%v", resp, err)
	}
	if one.calls != 2 || two.calls != 1 {
		t.Fatalf("unexpected calls one=%d two=%d", one.calls, two.calls)
	}
}

func TestRouterAuthFailsOverWithoutRetry(t *testing.T) {
	one := &scriptedProvider{name: "one", errors: []error{&ProviderError{Provider: "one", Class: ErrorAuth}}}
	two := &scriptedProvider{name: "two"}
	r := NewRouterWithOptions([]Provider{one, two}, RouterOptions{MaxRetries: 2})
	resp, err := r.Complete(context.Background(), CompletionRequest{})
	if err != nil || resp.Provider != "two" || one.calls != 1 {
		t.Fatalf("response=%v err=%v calls=%d", resp, err, one.calls)
	}
}

func TestRouterInvalidRequestDoesNotFailOver(t *testing.T) {
	one := &scriptedProvider{name: "one", errors: []error{&ProviderError{Provider: "one", Class: ErrorInvalidRequest}}}
	two := &scriptedProvider{name: "two"}
	r := NewRouterWithOptions([]Provider{one, two}, RouterOptions{MaxRetries: 2})
	if _, err := r.Complete(context.Background(), CompletionRequest{}); err == nil {
		t.Fatal("expected error")
	}
	if one.calls != 1 || two.calls != 0 {
		t.Fatalf("unexpected calls one=%d two=%d", one.calls, two.calls)
	}
}

func TestRouterCancellationDoesNotFailOver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	one := &scriptedProvider{name: "one"}
	two := &scriptedProvider{name: "two"}
	r := NewRouterWithOptions([]Provider{one, two}, RouterOptions{MaxRetries: 2})
	if _, err := r.Complete(ctx, CompletionRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if one.calls != 0 || two.calls != 0 {
		t.Fatal("providers called after cancellation")
	}
}

func TestHTTPErrorClassification(t *testing.T) {
	cases := map[int]ErrorClass{http.StatusBadRequest: ErrorInvalidRequest, http.StatusUnauthorized: ErrorAuth, http.StatusNotFound: ErrorNotFound, http.StatusTooManyRequests: ErrorRateLimit, http.StatusServiceUnavailable: ErrorUnavailable}
	for status, want := range cases {
		if got := ClassifyError(NewHTTPError("test", status, "body", http.Header{})); got != want {
			t.Errorf("status %d: got %s want %s", status, got, want)
		}
	}
}

func TestStatsReturnsDefensiveCopy(t *testing.T) {
	r := NewRouter([]Provider{&scriptedProvider{name: "one"}})
	stats := r.Stats()
	stats[0].CallCount = 99
	if r.Stats()[0].CallCount != 0 {
		t.Fatal("caller mutated router stats")
	}
}

func TestRouterRejectsNilSuccessfulResponseAndFailsOver(t *testing.T) {
	one := &scriptedProvider{name: "one", nilResponse: true}
	two := &scriptedProvider{name: "two"}
	r := NewRouterWithOptions([]Provider{one, two}, RouterOptions{MaxRetries: 0})
	resp, err := r.Complete(context.Background(), CompletionRequest{})
	if err != nil || resp == nil || resp.Provider != "two" {
		t.Fatalf("response=%#v err=%v", resp, err)
	}
	if stats := r.Stats()[0]; stats.SuccessCount != 0 || stats.FailCount != 1 {
		t.Fatalf("nil response stats=%+v", stats)
	}
}
