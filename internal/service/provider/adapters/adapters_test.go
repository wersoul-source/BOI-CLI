package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

// Every interactive turn asks to stream. An endpoint that ignores that and
// sends one ordinary completion used to produce an empty reply and no error.
func TestStreamFallsBackWhenTheServerDoesNotStream(t *testing.T) {
	server := serve(t, 200, `{"model":"m","choices":[{"message":{"content":"สวัสดี from a non-streaming gateway"},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":2}}`, nil)
	var seen []string
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
		OnText:   func(text string) { seen = append(seen, text) },
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "สวัสดี from a non-streaming gateway"
	if resp.Content != want || resp.FinishReason != "stop" || resp.InputTokens != 4 {
		t.Fatalf("resp = %+v", resp)
	}
	if len(seen) != 1 || seen[0] != want {
		t.Fatalf("the reply must be shown once, got %q", seen)
	}
}

// Parallel calls from a gateway that leaves out "index" must stay separate.
// They used to merge into one call whose arguments were two JSON objects.
func TestStreamSeparatesToolCallsWithoutIndex(t *testing.T) {
	server := sseServer(t, []string{
		`{"choices":[{"delta":{"tool_calls":[{"id":"a","function":{"name":"workspace__read","arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"th\":\"a\"}"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"id":"b","function":{"name":"workspace__list","arguments":"{\"path\":\"b\"}"}}]}}]}`,
		"[DONE]",
	}, nil)
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
		OnText:   func(string) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("want two calls, got %+v", resp.ToolCalls)
	}
	first, second := resp.ToolCalls[0], resp.ToolCalls[1]
	if first.ID != "a" || first.Name != "workspace__read" || first.Arguments != `{"path":"a"}` {
		t.Fatalf("first = %+v", first)
	}
	if second.ID != "b" || second.Name != "workspace__list" || second.Arguments != `{"path":"b"}` {
		t.Fatalf("second = %+v", second)
	}
}

// A model that writes its tool call as Hermes markup in the text gets the call
// recovered, marked as assisted, and the markup is never shown as reply text
// while it streams (Architecture IV, P1.3).
func TestStreamRecoversDialectToolCallsAndHidesTheMarkup(t *testing.T) {
	server := sseServer(t, []string{
		`{"model":"m","choices":[{"delta":{"content":"Reading it.\n<tool_"}}]}`,
		`{"choices":[{"delta":{"content":"call>{\"name\": \"workspace__read\", "}}]}`,
		`{"choices":[{"delta":{"content":"\"arguments\": {\"path\": \"README.md\"}}</tool_call>"},"finish_reason":"stop"}]}`,
		"[DONE]",
	}, nil)
	var seen []string
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "read"}},
		Tools:    []llm.ToolDefinition{{Name: "workspace__read"}},
		OnText:   func(text string) { seen = append(seen, text) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ToolCallDialect != "hermes" || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "workspace__read" || resp.ToolCalls[0].Arguments != `{"path":"README.md"}` {
		t.Fatalf("response = %+v", resp)
	}
	if resp.Content != "Reading it." {
		t.Fatalf("content = %q", resp.Content)
	}
	for _, text := range seen {
		if strings.Contains(text, "tool_call") || strings.Contains(text, "workspace__read") {
			t.Fatalf("streamed text showed the markup: %q", seen)
		}
	}
}

// Without Tools in the request nothing is recovered: markup is only a call
// when the host offered that Tool.
func TestDialectIsNotRecoveredWithoutOfferedTools(t *testing.T) {
	body := `{"choices":[{"message":{"role":"assistant","content":"<tool_call>{\"name\": \"workspace__read\", \"arguments\": {}}</tool_call>"},"finish_reason":"stop"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
	t.Cleanup(server.Close)
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{Messages: []llm.Message{{Role: "user", Content: "x"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 0 || resp.ToolCallDialect != "" {
		t.Fatalf("recovered a call no Tool was offered for: %+v", resp)
	}
}

// Reasoning an endpoint returns is carried on the response and replayed with
// the assistant turn that produced it (P1.4). An assistant message with no
// reasoning sends no reasoning_content field, so endpoints that never return
// it never see it.
func TestReasoningContentIsReturnedAndReplayed(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","reasoning_content":"read the file first","tool_calls":[{"id":"c1","type":"function","function":{"name":"workspace__read","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`))
	}))
	t.Cleanup(server.Close)
	provider := NewOpenAIProvider("openai", "k", server.URL, "m")
	first, err := provider.Complete(context.Background(), llm.CompletionRequest{Messages: []llm.Message{{Role: "user", Content: "go"}}, Tools: []llm.ToolDefinition{{Name: "workspace__read"}}})
	if err != nil {
		t.Fatal(err)
	}
	if first.ReasoningContent != "read the file first" {
		t.Fatalf("reasoning = %q", first.ReasoningContent)
	}
	_, err = provider.Complete(context.Background(), llm.CompletionRequest{Messages: []llm.Message{
		{Role: "user", Content: "go"},
		{Role: "assistant", ToolCalls: first.ToolCalls, ReasoningContent: first.ReasoningContent},
		{Role: "tool", ToolCallID: "c1", Content: "result"},
		{Role: "assistant", Content: "done"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var sent struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal([]byte(bodies[1]), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Messages[1]["reasoning_content"] != "read the file first" {
		t.Fatalf("assistant turn lost its reasoning: %v", sent.Messages[1])
	}
	for _, i := range []int{0, 2, 3} {
		if _, has := sent.Messages[i]["reasoning_content"]; has {
			t.Fatalf("message %d carries reasoning_content: %v", i, sent.Messages[i])
		}
	}
	if strings.Contains(bodies[0], "reasoning_content") {
		t.Fatal("the first request carried reasoning_content")
	}
}

func TestStreamCollectsReasoningWithoutShowingIt(t *testing.T) {
	server := sseServer(t, []string{
		`{"choices":[{"delta":{"reasoning_content":"think "}}]}`,
		`{"choices":[{"delta":{"reasoning_content":"hard"}}]}`,
		`{"choices":[{"delta":{"content":"answer"},"finish_reason":"stop"}]}`,
		"[DONE]",
	}, nil)
	var seen []string
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "x"}},
		OnText:   func(text string) { seen = append(seen, text) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ReasoningContent != "think hard" || resp.Content != "answer" {
		t.Fatalf("response = %q / %q", resp.ReasoningContent, resp.Content)
	}
	for _, text := range seen {
		if strings.Contains(text, "think") {
			t.Fatalf("reasoning was shown as reply text: %q", seen)
		}
	}
}

// An endpoint that ignores stream:true is a route change, and the response
// says so (P1.5).
func TestNonStreamingEndpointIsRecordedAsAFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(server.Close)
	resp, err := NewOpenAIProvider("openai", "k", server.URL, "m").Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "x"}},
		OnText:   func(string) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Fallbacks) != 1 || resp.Fallbacks[0].Kind != "stream→whole-response" || resp.Fallbacks[0].Reason == "" {
		t.Fatalf("fallbacks = %+v", resp.Fallbacks)
	}
}
