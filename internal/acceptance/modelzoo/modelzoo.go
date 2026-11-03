// Package modelzoo serves simulated model endpoints for tests. Each endpoint
// speaks the OpenAI Chat Completions protocol and reproduces one trait seen in
// a real model or gateway, or expected of a class of them, so the Connector,
// the probes and the Runtime can be tested against all of them in CI without a
// real Provider (Architecture IV, Phase 0, P0.6).
//
// An endpoint behaves like a model doing one scripted task: asked to read
// README.md, it requests the read in its own tool-call format; once a tool
// result carrying Marker is in the conversation, it answers with the marker.
// It also answers the BOI qualification probes the way its trait would.
package modelzoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Marker is the first line of the README.md the scripted task reads. A reply
// that carries it proves the model saw the real tool result.
const Marker = "ZOO-MARKER-42"

// Trait is the behaviour one simulated endpoint reproduces.
type Trait string

const (
	// Native is a well-behaved model: native tool calls, streaming, correct
	// answers. The control case.
	Native Trait = "native"
	// NoStream ignores "stream": true and always sends one JSON completion,
	// as some gateways do. Seen-class: fixed in v2.7.1.
	NoStream Trait = "no-stream"
	// DSML writes DeepSeek's DSML tool-call markup into the message text
	// instead of tool_calls. Seen with maxplus/deepseek-v4.1-flash on
	// 2026-10-24.
	DSML Trait = "dsml"
	// Hermes writes a <tool_call>{json}</tool_call> block into the text, the
	// format of Hermes-style chat templates.
	Hermes Trait = "hermes"
	// QwenXML writes Qwen3-Coder's <function=…><parameter=…> markup into the
	// text, as served without a tool parser.
	QwenXML Trait = "qwen-xml"
	// ReasoningReplay returns reasoning_content and rejects, with HTTP 400, a
	// later request whose assistant tool-call message does not carry it back,
	// as DeepSeek's thinking mode does.
	ReasoningReplay Trait = "reasoning-replay"
	// RejectsReplay accepts tool definitions and returns native tool calls,
	// then answers HTTP 400 to any request that replays a tool result, as
	// some gateways do. The Runtime must finish the task another way and say so.
	RejectsReplay Trait = "rejects-replay"
	// MissingIndex streams two parallel tool calls without "index".
	MissingIndex Trait = "missing-index"
	// CorruptToken answers correctly in shape but corrupts one token, as
	// maxplus did in its tool_observation probe (OBSERVATION_public).
	CorruptToken Trait = "corrupt-token"
	// SlowFirstLoad delays its first answer, like a local model loading.
	SlowFirstLoad Trait = "slow-first-load"
	// Down fails every request with HTTP 503, like a dead cluster replica.
	Down Trait = "down"
)

// Traits lists every trait the zoo serves, in a stable order.
var Traits = []Trait{Native, NoStream, DSML, Hermes, QwenXML, ReasoningReplay, RejectsReplay, MissingIndex, CorruptToken, SlowFirstLoad, Down}

// SlowDelay is how long a SlowFirstLoad endpoint waits before its first answer.
const SlowDelay = 300 * time.Millisecond

// Endpoint is one running simulated model.
type Endpoint struct {
	Trait Trait
	// URL is the base URL to give an OpenAI-compatible adapter.
	URL string

	mu        sync.Mutex
	requests  int
	rejected  int
	streamed  int
	firstDone bool
}

// Start serves an endpoint with the given trait until the test ends.
func Start(t testing.TB, trait Trait) *Endpoint {
	t.Helper()
	endpoint := &Endpoint{Trait: trait}
	server := httptest.NewServer(http.HandlerFunc(endpoint.serve))
	t.Cleanup(server.Close)
	endpoint.URL = server.URL
	return endpoint
}

// Requests is how many requests the endpoint received.
func (e *Endpoint) Requests() int { e.mu.Lock(); defer e.mu.Unlock(); return e.requests }

// Rejected is how many requests it refused with HTTP 400 (ReasoningReplay,
// RejectsReplay).
func (e *Endpoint) Rejected() int { e.mu.Lock(); defer e.mu.Unlock(); return e.rejected }

// Streamed is how many answers it sent as server-sent events.
func (e *Endpoint) Streamed() int { e.mu.Lock(); defer e.mu.Unlock(); return e.streamed }

type message struct {
	Role             string     `json:"role"`
	Content          *string    `json:"content"`
	ToolCalls        []toolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ReasoningContent *string    `json:"reasoning_content,omitempty"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type request struct {
	Messages []message `json:"messages"`
	Tools    []struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	} `json:"tools"`
	Stream bool `json:"stream"`
}

type reply struct {
	content   string
	calls     []toolCall
	reasoning string
}

func (e *Endpoint) serve(w http.ResponseWriter, r *http.Request) {
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"message":"bad json"}}`, http.StatusBadRequest)
		return
	}
	e.mu.Lock()
	e.requests++
	first := !e.firstDone
	e.firstDone = true
	e.mu.Unlock()

	switch e.Trait {
	case Down:
		writeError(w, http.StatusServiceUnavailable, "replica unavailable")
		return
	case SlowFirstLoad:
		if first {
			time.Sleep(SlowDelay)
		}
	case RejectsReplay:
		for _, m := range req.Messages {
			if m.Role == "tool" {
				e.mu.Lock()
				e.rejected++
				e.mu.Unlock()
				writeError(w, http.StatusBadRequest, "messages with role tool are not supported by this gateway")
				return
			}
		}
	case ReasoningReplay:
		for _, m := range req.Messages {
			if m.Role == "assistant" && len(m.ToolCalls) > 0 && m.ReasoningContent == nil {
				e.mu.Lock()
				e.rejected++
				e.mu.Unlock()
				writeError(w, http.StatusBadRequest, "The `reasoning_content` in the thinking mode must be passed back to the API.")
				return
			}
		}
	}

	answer := e.decide(req)
	if req.Stream && e.Trait != NoStream {
		e.mu.Lock()
		e.streamed++
		e.mu.Unlock()
		writeStream(w, answer, e.Trait != MissingIndex)
		return
	}
	writeJSON(w, answer)
}

func (e *Endpoint) decide(req request) reply {
	last := ""
	if n := len(req.Messages); n > 0 && req.Messages[n-1].Content != nil {
		last = *req.Messages[n-1].Content
	}
	if strings.HasPrefix(last, "BOI_PROBE ") {
		return e.probe(last, len(req.Tools) > 0)
	}
	for _, m := range req.Messages {
		if m.Content != nil && strings.Contains(*m.Content, Marker) && m.Role != "system" {
			if e.Trait == CorruptToken {
				return reply{content: "The first line is ZOO-MARKER-public"}
			}
			return reply{content: "The first line is " + Marker}
		}
	}
	return e.requestRead(len(req.Tools) > 0)
}

// requestRead asks for README.md in the trait's own tool-call format.
func (e *Endpoint) requestRead(nativeOffered bool) reply {
	switch e.Trait {
	case DSML:
		return reply{content: dsmlCall("workspace__read", "path", "README.md")}
	case Hermes:
		return reply{content: "<tool_call>\n{\"name\": \"workspace__read\", \"arguments\": {\"path\": \"README.md\"}}\n</tool_call>"}
	case QwenXML:
		return reply{content: "<tool_call>\n<function=workspace__read>\n<parameter=path>\nREADME.md\n</parameter>\n</function>\n</tool_call>"}
	}
	if !nativeOffered {
		return reply{content: `<boi-action>{"id":"zoo","tool":"workspace.read","purpose":"read the README","arguments":{"path":"README.md"}}</boi-action>`}
	}
	answer := reply{calls: []toolCall{call("call_read", "workspace__read", `{"path":"README.md"}`)}}
	if e.Trait == MissingIndex {
		answer.calls = append(answer.calls, call("call_list", "workspace__list", `{"path":"."}`))
	}
	if e.Trait == ReasoningReplay {
		answer.reasoning = "The task needs the file, so read it first."
	}
	return answer
}

// probe answers a BOI qualification probe the way the trait would.
func (e *Endpoint) probe(prompt string, nativeOffered bool) reply {
	capability := strings.TrimSuffix(strings.Fields(strings.TrimPrefix(prompt, "BOI_PROBE "))[0], ":")
	switch capability {
	case "completion":
		return reply{content: "BOI_OK"}
	case "reasoning":
		return reply{content: "1>2>3"}
	case "skill_calling":
		return reply{content: "SKILL:beta"}
	case "authority":
		return reply{content: "DENY"}
	case "context":
		return reply{content: "BOTH_NEEDLES"}
	case "tool_observation":
		if e.Trait == CorruptToken {
			return reply{content: "OBSERVATION_public"}
		}
		return reply{content: "OBSERVATION_IGNORED"}
	case "tool_calling", "native_tool_schema":
		switch e.Trait {
		case DSML, Hermes, QwenXML:
			return e.requestRead(nativeOffered)
		}
		if capability == "tool_calling" {
			return reply{content: `<boi-action>{"id":"probe","tool":"workspace.read","purpose":"probe","arguments":{"path":"README.md"}}</boi-action>`}
		}
		return reply{calls: []toolCall{call("call_probe", "workspace__read", `{"path":"README.md"}`)}}
	}
	return reply{content: "BOI_OK"}
}

func call(id, name, arguments string) toolCall {
	var c toolCall
	c.ID, c.Type = id, "function"
	c.Function.Name, c.Function.Arguments = name, arguments
	return c
}

// dsmlCall renders one call in DeepSeek V4's DSML grammar as the vLLM parser
// documents it: invoke and parameter elements inside a tool_calls block.
func dsmlCall(name, parameter, value string) string {
	const bar = "｜"
	return fmt.Sprintf("<%[1]sDSML%[1]stool_calls>\n<%[1]sDSML%[1]sinvoke name=%[2]q>\n<%[1]sDSML%[1]sparameter name=%[3]q string=\"true\">%[4]s</%[1]sDSML%[1]sparameter>\n</%[1]sDSML%[1]sinvoke>\n</%[1]sDSML%[1]stool_calls>", bar, name, parameter, value)
}

func writeError(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": text}})
}

func writeJSON(w http.ResponseWriter, answer reply) {
	msg := map[string]any{"role": "assistant", "content": answer.content}
	finish := "stop"
	if len(answer.calls) > 0 {
		msg["tool_calls"] = answer.calls
		finish = "tool_calls"
	}
	if answer.reasoning != "" {
		msg["reasoning_content"] = answer.reasoning
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model":   "zoo",
		"choices": []any{map[string]any{"message": msg, "finish_reason": finish}},
		"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5},
	})
}

// writeStream sends the answer as server-sent events, the text in two pieces.
// withIndex false leaves "index" out of every tool-call piece.
func writeStream(w http.ResponseWriter, answer reply, withIndex bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	flusher, _ := w.(http.Flusher)
	send := func(chunk any) {
		data, _ := json.Marshal(chunk)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		if flusher != nil {
			flusher.Flush()
		}
	}
	delta := func(d map[string]any, finish any) map[string]any {
		return map[string]any{"model": "zoo", "choices": []any{map[string]any{"delta": d, "finish_reason": finish}}}
	}
	if answer.reasoning != "" {
		send(delta(map[string]any{"reasoning_content": answer.reasoning}, nil))
	}
	if answer.content != "" {
		half := len([]rune(answer.content)) / 2
		runes := []rune(answer.content)
		send(delta(map[string]any{"content": string(runes[:half])}, nil))
		send(delta(map[string]any{"content": string(runes[half:])}, nil))
	}
	for i, c := range answer.calls {
		piece := map[string]any{"id": c.ID, "type": "function", "function": map[string]any{"name": c.Function.Name, "arguments": c.Function.Arguments}}
		if withIndex {
			piece["index"] = i
		}
		send(delta(map[string]any{"tool_calls": []any{piece}}, nil))
	}
	finish := "stop"
	if len(answer.calls) > 0 {
		finish = "tool_calls"
	}
	send(delta(map[string]any{}, finish))
	send(map[string]any{"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5}, "choices": []any{}})
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
}
