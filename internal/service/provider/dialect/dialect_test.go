package dialect

import (
	"encoding/json"
	"strings"
	"testing"

	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
)

var offered = []llm.ToolDefinition{
	{Name: "workspace__read", Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}}},
	{Name: "process__run", Parameters: map[string]any{"type": "object", "properties": map[string]any{
		"command": map[string]any{"type": "string"},
		"timeout": map[string]any{"type": "integer"},
		"env":     map[string]any{"type": "object"},
	}}},
}

func arguments(t *testing.T, call llm.ToolCallRequest) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &out); err != nil {
		t.Fatalf("arguments are not JSON: %v", err)
	}
	return out
}

func TestParseEachDialect(t *testing.T) {
	cases := []struct {
		name    string
		content string
		dialect Dialect
	}{
		{"dsml", "Let me look.\n<｜DSML｜tool_calls>\n<｜DSML｜invoke name=\"workspace__read\">\n<｜DSML｜parameter name=\"path\" string=\"true\">README.md</｜DSML｜parameter>\n</｜DSML｜invoke>\n</｜DSML｜tool_calls>", DSML},
		{"dsml ascii bar", "<|DSML|tool_calls><|DSML|invoke name=\"workspace__read\"><|DSML|parameter name=\"path\" string=\"true\">README.md</|DSML|parameter></|DSML|invoke></|DSML|tool_calls>", DSML},
		{"hermes", "Let me look.\n<tool_call>\n{\"name\": \"workspace__read\", \"arguments\": {\"path\": \"README.md\"}}\n</tool_call>", Hermes},
		{"hermes string arguments", "<tool_call>{\"name\": \"workspace__read\", \"arguments\": \"{\\\"path\\\": \\\"README.md\\\"}\"}</tool_call>", Hermes},
		{"qwen xml", "Let me look.\n<tool_call>\n<function=workspace__read>\n<parameter=path>\nREADME.md\n</parameter>\n</function>\n</tool_call>", QwenXML},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := Parse(tc.content, offered)
			if !ok {
				t.Fatal("markup was not recovered")
			}
			if result.Dialect != tc.dialect || len(result.Calls) != 1 || result.Calls[0].Name != "workspace__read" {
				t.Fatalf("got %+v", result)
			}
			if got := arguments(t, result.Calls[0]); got["path"] != "README.md" || len(got) != 1 {
				t.Fatalf("arguments = %v", got)
			}
			if strings.Contains(result.Text, "tool_call") || strings.Contains(result.Text, "DSML") {
				t.Fatalf("markup left in text: %q", result.Text)
			}
		})
	}
}

// The reply seen on 2026-10-24 asked for a shell command in DSML. Its typed
// parameters follow the schema: string="false" is JSON.
func TestParseDSMLTypedParameters(t *testing.T) {
	content := "<｜DSML｜tool_calls><｜DSML｜invoke name=\"process__run\">" +
		"<｜DSML｜parameter name=\"command\" string=\"true\">go test ./...</｜DSML｜parameter>" +
		"<｜DSML｜parameter name=\"timeout\" string=\"false\">30</｜DSML｜parameter>" +
		"</｜DSML｜invoke></｜DSML｜tool_calls>"
	result, ok := Parse(content, offered)
	if !ok {
		t.Fatal("not recovered")
	}
	got := arguments(t, result.Calls[0])
	if got["command"] != "go test ./..." || got["timeout"] != float64(30) {
		t.Fatalf("arguments = %v", got)
	}
}

func TestParseQwenConvertsBySchema(t *testing.T) {
	content := "<tool_call><function=process__run><parameter=command>\nls -la\n</parameter><parameter=timeout>\n15\n</parameter><parameter=env>{\"A\":\"1\"}</parameter></function></tool_call>"
	result, ok := Parse(content, offered)
	if !ok {
		t.Fatal("not recovered")
	}
	got := arguments(t, result.Calls[0])
	if got["command"] != "ls -la" || got["timeout"] != float64(15) || got["env"].(map[string]any)["A"] != "1" {
		t.Fatalf("arguments = %v", got)
	}
}

func TestParseSeveralCalls(t *testing.T) {
	content := "<tool_call>{\"name\":\"workspace__read\",\"arguments\":{\"path\":\"a\"}}</tool_call>\n<tool_call>{\"name\":\"workspace__read\",\"arguments\":{\"path\":\"b\"}}</tool_call>"
	result, ok := Parse(content, offered)
	if !ok || len(result.Calls) != 2 || arguments(t, result.Calls[1])["path"] != "b" {
		t.Fatalf("got %+v ok=%v", result, ok)
	}
	if result.Calls[0].ID == result.Calls[1].ID {
		t.Fatal("recovered calls share an ID")
	}
}

// Anything not fully understood stays text: a partial recovery would run part
// of what the model asked for and drop the rest.
func TestParseRefuses(t *testing.T) {
	cases := map[string]string{
		"no markup":            "README.md says hello",
		"tool not offered":     "<tool_call>{\"name\":\"secrets__dump\",\"arguments\":{}}</tool_call>",
		"one of two unknown":   "<tool_call>{\"name\":\"workspace__read\",\"arguments\":{\"path\":\"a\"}}</tool_call><tool_call>{\"name\":\"shell\",\"arguments\":{}}</tool_call>",
		"broken json":          "<tool_call>{\"name\":\"workspace__read\",\"arguments\":{\"path\":</tool_call>",
		"mixed dialects":       "<tool_call>{\"name\":\"workspace__read\",\"arguments\":{\"path\":\"a\"}}</tool_call><tool_call><function=workspace__read><parameter=path>b</parameter></function></tool_call>",
		"junk inside dsml":     "<｜DSML｜tool_calls>rm -rf /<｜DSML｜invoke name=\"workspace__read\"></｜DSML｜invoke></｜DSML｜tool_calls>",
		"wrong type":           "<tool_call><function=process__run><parameter=timeout>soon</parameter></function></tool_call>",
		"unknown inner format": "<tool_call>workspace__read README.md</tool_call>",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if result, ok := Parse(content, offered); ok {
				t.Fatalf("recovered %+v", result)
			}
		})
	}
	if _, ok := Parse("<tool_call>{\"name\":\"workspace__read\",\"arguments\":{}}</tool_call>", nil); ok {
		t.Fatal("recovered a call although no Tools were offered")
	}
}

func TestParseBoundsTheBatch(t *testing.T) {
	block := "<tool_call>{\"name\":\"workspace__read\",\"arguments\":{\"path\":\"a\"}}</tool_call>"
	if _, ok := Parse(strings.Repeat(block, MaxCalls+1), offered); ok {
		t.Fatalf("recovered more than %d calls", MaxCalls)
	}
}

func TestVisibleHidesMarkup(t *testing.T) {
	cases := map[string]string{
		"Let me look.\n<tool_call>{\"name\"":    "Let me look.",
		"Checking <｜DSML｜tool_calls><｜DSML｜inv": "Checking",
		"plain answer": "plain answer",
		"<tool_call>":  "",
	}
	for in, want := range cases {
		if got := Visible(in); got != want {
			t.Errorf("Visible(%q) = %q, want %q", in, got, want)
		}
	}
}
