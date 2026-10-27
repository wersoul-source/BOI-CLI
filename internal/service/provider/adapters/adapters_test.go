package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
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
	t.Setenv(EnvRequestTimeout, "")
	if newHTTPClient().Timeout != defaultRequestTimeout || defaultRequestTimeout <= 0 {
		t.Fatal("http client must have a bounded timeout")
	}
}

func TestRequestTimeoutCanBeRaisedForSlowLocalModels(t *testing.T) {
	cases := map[string]time.Duration{
		"10m": 10 * time.Minute, " 90s ": 90 * time.Second,
		"": defaultRequestTimeout, "soon": defaultRequestTimeout, "1s": defaultRequestTimeout, "48h": defaultRequestTimeout, "-5m": defaultRequestTimeout,
	}
	for value, want := range cases {
		t.Setenv(EnvRequestTimeout, value)
		if got := newHTTPClient().Timeout; got != want {
			t.Errorf("%s=%q: timeout %v, want %v", EnvRequestTimeout, value, got, want)
		}
	}
}

func TestAnEmptyAPIKeySendsNoAuthorizationHeader(t *testing.T) {
	var header string
	var seen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header, seen = r.Header.Get("Authorization"), true
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer server.Close()
	provider := NewOpenAIProvider("ollama", "", server.URL, "m")
	if _, err := provider.Complete(context.Background(), llm.CompletionRequest{Messages: []llm.Message{{Role: "user", Content: "hi"}}}); err != nil {
		t.Fatal(err)
	}
	if !seen || header != "" {
		t.Fatalf("a keyless local server must not receive an Authorization header: seen=%v header=%q", seen, header)
	}
}

// sseServer replies with a server-sent event stream built from the given
// lines, the way a Chat Completions endpoint does.
func sseServer(t *testing.T, events []string, inspect func(*http.Request)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inspect != nil {
			inspect(r)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, event := range events {
			_, _ = w.Write([]byte("data: " + event + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// A streamed reply must reach the caller while it is being produced, and the
// whole text must still come back in the response. OnText is cumulative, so a
// restarted reply replaces what was shown instead of doubling it.
func TestStreamReportsTextAsItArrives(t *testing.T) {
	var sent map[string]any
	server := sseServer(t, []string{
		`{"model":"m","choices":[{"delta":{"content":"สวัส"}}]}`,
		`{"choices":[{"delta":{"content":"ดี"}}]}`,
		`{"choices":[{"delta":{"content":" ok"},"finish_reason":"stop"}]}`,
		`{"usage":{"prompt_tokens":7,"completion_tokens":3}}`,
		"[DONE]",
	}, func(r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Error(err)
		}
	})
	var seen []string
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
		OnText:   func(text string) { seen = append(seen, text) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if sent["stream"] != true {
		t.Fatalf("a streaming request must ask the endpoint to stream: %v", sent["stream"])
	}
	want := []string{"สวัส", "สวัสดี", "สวัสดี ok"}
	if len(seen) != len(want) {
		t.Fatalf("got %d updates, want %d: %q", len(seen), len(want), seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("update %d = %q, want %q", i, seen[i], want[i])
		}
	}
	if resp.Content != "สวัสดี ok" || resp.FinishReason != "stop" {
		t.Fatalf("response = %q / %q", resp.Content, resp.FinishReason)
	}
	if resp.InputTokens != 7 || resp.OutputTokens != 3 {
		t.Fatalf("usage = %d/%d", resp.InputTokens, resp.OutputTokens)
	}
}

// A tool call arrives in pieces that share an index, with its arguments split
// across chunks. It must be reassembled into the same call a non-streamed
// reply returns.
func TestStreamReassemblesASplitToolCall(t *testing.T) {
	server := sseServer(t, []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"workspace__read","arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\":\"a.txt\"}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		"[DONE]",
	}, nil)
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
		OnText:   func(string) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("want one tool call, got %d", len(resp.ToolCalls))
	}
	call := resp.ToolCalls[0]
	if call.ID != "c1" || call.Name != "workspace__read" || call.Arguments != `{"path":"a.txt"}` {
		t.Fatalf("call = %+v", call)
	}
}

// Keep-alives, comments and events this adapter does not model must not end
// the stream.
func TestStreamIgnoresUnknownEvents(t *testing.T) {
	server := sseServer(t, []string{
		": keep-alive",
		`{"choices":[{"delta":{"role":"assistant"}}]}`,
		`{"choices":[{"delta":{"content":"hi"}}]}`,
		"[DONE]",
	}, nil)
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
		OnText:   func(string) {},
	})
	if err != nil || resp.Content != "hi" {
		t.Fatalf("resp = %#v err = %v", resp, err)
	}
}

// A rejected streaming request must classify like any other provider error,
// so the Router can fail over or retry.
func TestStreamSurfacesAnHTTPError(t *testing.T) {
	server := serve(t, 401, `{"error":"bad key"}`, nil)
	_, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
		OnText:   func(string) {},
	})
	if llm.ClassifyError(err) != llm.ErrorAuth {
		t.Fatalf("want an auth error, got %v", err)
	}
}

// Without OnText nothing changes: the request does not ask to stream.
func TestCompleteDoesNotStreamWithoutASink(t *testing.T) {
	var sent map[string]any
	server := serve(t, 200, `{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`, func(r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Error(err)
		}
	})
	if _, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if sent["stream"] != false {
		t.Fatalf("stream = %v, want false", sent["stream"])
	}
}

func TestOpenAINativeToolRoundTrip(t *testing.T) {
	var sent map[string]any
	srv := serve(t, 200, `{"choices":[{"message":{"content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"workspace__read","arguments":"{\"path\":\"a.txt\"}"}}]}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`, func(r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Error(err)
		}
	})
	provider := NewOpenAIProvider("openai", "k", srv.URL, "m")
	if !llm.SupportsNativeTools(provider) {
		t.Fatal("OpenAI adapter must declare native tool support")
	}
	resp, err := provider.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "read"},
			{Role: "assistant", ToolCalls: []llm.ToolCallRequest{{ID: "h1", Name: "workspace__read", Arguments: `{"path":"x"}`}}},
			{Role: "tool", ToolCallID: "h1", Content: "data"},
		},
		Tools: []llm.ToolDefinition{{Name: "workspace__read", Description: "read", Parameters: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "workspace__read" || resp.ToolCalls[0].Arguments != `{"path":"a.txt"}` {
		t.Fatalf("tool calls not parsed: %+v", resp.ToolCalls)
	}
	tools, _ := sent["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools not sent: %v", sent["tools"])
	}
	messages, _ := sent["messages"].([]any)
	assistant, _ := messages[1].(map[string]any)
	if assistant["content"] != nil || assistant["tool_calls"] == nil {
		t.Fatalf("assistant tool-call message must have null content and tool_calls: %v", assistant)
	}
	toolReply, _ := messages[2].(map[string]any)
	if toolReply["tool_call_id"] != "h1" {
		t.Fatalf("tool reply must link its call: %v", toolReply)
	}
}

func TestOpenAIOmitsToolsWhenNoneGiven(t *testing.T) {
	var sent map[string]any
	srv := serve(t, 200, `{"choices":[{"message":{"content":"hi"}}]}`, func(r *http.Request) { _ = json.NewDecoder(r.Body).Decode(&sent) })
	if _, err := NewOpenAIProvider("openai", "k", srv.URL, "m").Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if _, present := sent["tools"]; present {
		t.Fatal("a request without tools must not send an empty tools field")
	}
}

func TestOpenAISendsTemperatureZero(t *testing.T) {
	var sent map[string]any
	srv := serve(t, 200, `{"choices":[{"message":{"content":"hi"}}]}`, func(r *http.Request) { _ = json.NewDecoder(r.Body).Decode(&sent) })
	if _, err := NewOpenAIProvider("openai", "k", srv.URL, "m").Complete(context.Background(), llm.CompletionRequest{Messages: req.Messages, Temperature: 0}); err != nil {
		t.Fatal(err)
	}
	if value, present := sent["temperature"]; !present || value != 0.0 {
		t.Fatalf("temperature 0 must be sent, got %v (present=%v)", value, present)
	}
}
