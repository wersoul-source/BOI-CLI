package adapters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	llm "github.com/boi-family/boi-cli/internal/provider"
)

type adapterCase struct {
	name    string
	build   func(baseURL string) llm.Provider
	okBody  string
	emptyOK string
	auth    func(t *testing.T, r *http.Request)
}

func cases() []adapterCase {
	return []adapterCase{
		{
			name:    "openai",
			build:   func(u string) llm.Provider { return NewOpenAIProvider("openai", "sk-secret", u, "m") },
			okBody:  `{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":3,"completion_tokens":2},"model":"m"}`,
			emptyOK: `{"choices":[]}`,
			auth: func(t *testing.T, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer sk-secret" {
					t.Error("missing bearer auth")
				}
			},
		},
		{
			name:    "anthropic",
			build:   func(u string) llm.Provider { return NewAnthropicProvider("anthropic", "sk-secret", u, "m") },
			okBody:  `{"content":[{"text":"hi"}],"usage":{"input_tokens":3,"output_tokens":2},"model":"m"}`,
			emptyOK: `{"content":[]}`,
			auth: func(t *testing.T, r *http.Request) {
				if r.Header.Get("x-api-key") != "sk-secret" {
					t.Error("missing x-api-key")
				}
			},
		},
		{
			name:    "google",
			build:   func(u string) llm.Provider { return NewGoogleProvider("google", "sk-secret", u, "m") },
			okBody:  `{"candidates":[{"content":{"parts":[{"text":"hi"}]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2}}`,
			emptyOK: `{"candidates":[]}`,
			auth: func(t *testing.T, r *http.Request) {
				if r.Header.Get("x-goog-api-key") != "sk-secret" {
					t.Error("missing x-goog-api-key")
				}
				if strings.Contains(r.URL.String(), "sk-secret") {
					t.Error("API key must not appear in the URL")
				}
			},
		},
	}
}

var req = llm.CompletionRequest{Messages: []llm.Message{{Role: "user", Content: "hello"}}}

func serve(t *testing.T, status int, body string, check func(*http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCompleteSuccess(t *testing.T) {
	for _, c := range cases() {
		t.Run(c.name, func(t *testing.T) {
			srv := serve(t, 200, c.okBody, func(r *http.Request) { c.auth(t, r) })
			resp, err := c.build(srv.URL).Complete(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Content != "hi" || resp.InputTokens != 3 || resp.OutputTokens != 2 {
				t.Fatalf("unexpected response %+v", resp)
			}
		})
	}
}

func TestCompleteHTTPError(t *testing.T) {
	for _, c := range cases() {
		for _, status := range []int{400, 401, 429, 500} {
			t.Run(c.name, func(t *testing.T) {
				srv := serve(t, status, `{"error":"x"}`, nil)
				_, err := c.build(srv.URL).Complete(context.Background(), req)
				if err == nil {
					t.Fatalf("status %d: expected error", status)
				}
				var he *llm.ProviderError
				if !errors.As(err, &he) || he.StatusCode != status {
					t.Fatalf("status %d: want HTTPError, got %v", status, err)
				}
			})
		}
	}
}

func TestCompleteMalformedAndEmpty(t *testing.T) {
	for _, c := range cases() {
		t.Run(c.name, func(t *testing.T) {
			for _, body := range []string{`not json`, c.emptyOK} {
				srv := serve(t, 200, body, nil)
				if _, err := c.build(srv.URL).Complete(context.Background(), req); err == nil {
					t.Fatalf("body %q: expected error", body)
				}
			}
		})
	}
}

func TestCompleteOversizedResponse(t *testing.T) {
	big := strings.Repeat("a", maxResponseBytes+10)
	for _, c := range cases() {
		t.Run(c.name, func(t *testing.T) {
			srv := serve(t, 200, big, nil)
			_, err := c.build(srv.URL).Complete(context.Background(), req)
			if err == nil || !strings.Contains(err.Error(), "exceeds") {
				t.Fatalf("want size error, got %v", err)
			}
		})
	}
}

func TestCompleteContextCancel(t *testing.T) {
	for _, c := range cases() {
		t.Run(c.name, func(t *testing.T) {
			release := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-release:
				case <-r.Context().Done():
				}
			}))
			defer srv.Close()
			defer close(release)

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			start := time.Now()
			if _, err := c.build(srv.URL).Complete(ctx, req); err == nil {
				t.Fatal("expected error on hung server")
			}
			if time.Since(start) > 5*time.Second {
				t.Fatal("hung request was not cut off by context")
			}
		})
	}
}

func TestClientHasTimeout(t *testing.T) {
	if newHTTPClient().Timeout != requestTimeout || requestTimeout <= 0 {
		t.Fatal("http client must have a bounded timeout")
	}
}

func TestStreamEmitsContentThenDone(t *testing.T) {
	for _, c := range cases() {
		t.Run(c.name, func(t *testing.T) {
			srv := serve(t, 200, c.okBody, nil)
			ch, err := c.build(srv.URL).Stream(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			first, second := <-ch, <-ch
			if first.Text != "hi" || !second.Done {
				t.Fatalf("got %+v %+v", first, second)
			}
		})
	}
}
