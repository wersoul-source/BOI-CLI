// Package dialect recovers tool calls that a model wrote into its message text
// in its own markup instead of the Chat Completions tool_calls field. It is
// the Dialect Parser of the Connector (Architecture IV, P1.3).
//
// A recovered call is a compensation, not the model's native protocol: every
// Result names the dialect so the capability is reported as ASSISTED, never
// as NATIVE (ADR-006). Parsing grants nothing. A recovered call is checked
// against the Tools the request offered here and then goes through the same
// Broker authorisation as any other call.
package dialect

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
)

// Dialect names a tool-call markup.
type Dialect string

const (
	// DSML is DeepSeek V4's markup: <｜DSML｜invoke name="…"> with
	// <｜DSML｜parameter name="…" string="true|false"> children.
	DSML Dialect = "dsml"
	// Hermes is a <tool_call>{"name": …, "arguments": …}</tool_call> block.
	Hermes Dialect = "hermes"
	// QwenXML is Qwen3-Coder's <tool_call><function=…><parameter=…> markup.
	QwenXML Dialect = "qwen-xml"
)

// Result is what Parse recovered from one reply.
type Result struct {
	Dialect Dialect
	Calls   []llm.ToolCallRequest
	// Text is the reply with the markup removed.
	Text string
}

// MaxCalls bounds how many calls one reply may yield, so a runaway reply
// cannot become an unbounded batch before the Runtime's own limit applies.
const MaxCalls = 16

// bar matches the full-width separator DeepSeek uses and the ASCII one some
// gateways substitute for it.
const bar = `(?:｜|\|)`

var (
	dsmlBlock     = regexp.MustCompile(`(?s)<` + bar + `DSML` + bar + `tool_calls>(.*?)</` + bar + `DSML` + bar + `tool_calls>`)
	dsmlInvoke    = regexp.MustCompile(`(?s)<` + bar + `DSML` + bar + `invoke\s+name="([^"]+)"\s*>(.*?)</` + bar + `DSML` + bar + `invoke>`)
	dsmlParameter = regexp.MustCompile(`(?s)<` + bar + `DSML` + bar + `parameter\s+name="([^"]+)"(?:\s+string="(true|false)")?\s*>(.*?)</` + bar + `DSML` + bar + `parameter>`)
	toolCallBlock = regexp.MustCompile(`(?s)<tool_call>(.*?)</tool_call>`)
	qwenFunction  = regexp.MustCompile(`(?s)^<function=([^>\s]+)>(.*?)</function>$`)
	qwenParameter = regexp.MustCompile(`(?s)<parameter=([^>\s]+)>(.*?)</parameter>`)
	// openers are where markup starts, for hiding it from streamed text.
	openers = regexp.MustCompile(`<` + bar + `DSML` + bar + `|<tool_call>`)
)

// Parse recovers tool calls from content. It succeeds only when the reply
// holds markup of one known dialect, every call in it is well formed, and
// every call names a Tool in offered. Anything else returns false and the
// reply stays text: a partial recovery would run some of what the model asked
// for and silently drop the rest.
func Parse(content string, offered []llm.ToolDefinition) (Result, bool) {
	if len(offered) == 0 {
		return Result{}, false
	}
	tools := make(map[string]llm.ToolDefinition, len(offered))
	for _, tool := range offered {
		tools[tool.Name] = tool
	}
	if blocks := dsmlBlock.FindAllStringSubmatchIndex(content, -1); len(blocks) > 0 {
		return parseDSML(content, blocks, tools)
	}
	if blocks := toolCallBlock.FindAllStringSubmatchIndex(content, -1); len(blocks) > 0 {
		return parseToolCallBlocks(content, blocks, tools)
	}
	return Result{}, false
}

// Visible returns the part of streamed text before any tool-call markup, so a
// live reply does not show markup that is about to become a tool call.
func Visible(text string) string {
	if loc := openers.FindStringIndex(text); loc != nil {
		return strings.TrimRight(text[:loc[0]], " \t\r\n")
	}
	return text
}

func parseDSML(content string, blocks [][]int, tools map[string]llm.ToolDefinition) (Result, bool) {
	result := Result{Dialect: DSML}
	for _, block := range blocks {
		inner := content[block[2]:block[3]]
		invokes := dsmlInvoke.FindAllStringSubmatch(inner, -1)
		if len(invokes) == 0 || strings.TrimSpace(dsmlInvoke.ReplaceAllString(inner, "")) != "" {
			return Result{}, false
		}
		for _, invoke := range invokes {
			tool, ok := tools[invoke[1]]
			if !ok {
				return Result{}, false
			}
			arguments := map[string]any{}
			parameters := dsmlParameter.FindAllStringSubmatch(invoke[2], -1)
			if strings.TrimSpace(dsmlParameter.ReplaceAllString(invoke[2], "")) != "" {
				return Result{}, false
			}
			for _, parameter := range parameters {
				value, err := convert(tool, parameter[1], parameter[3], parameter[2] == "false")
				if err != nil {
					return Result{}, false
				}
				arguments[parameter[1]] = value
			}
			if !result.add(invoke[1], arguments) {
				return Result{}, false
			}
		}
	}
	result.Text = strip(content, blocks)
	return result, true
}

func parseToolCallBlocks(content string, blocks [][]int, tools map[string]llm.ToolDefinition) (Result, bool) {
	var result Result
	for _, block := range blocks {
		inner := strings.TrimSpace(content[block[2]:block[3]])
		var dialect Dialect
		var name string
		arguments := map[string]any{}
		switch {
		case strings.HasPrefix(inner, "{"):
			dialect = Hermes
			var call struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if err := json.Unmarshal([]byte(inner), &call); err != nil || call.Name == "" {
				return Result{}, false
			}
			name = call.Name
			if len(call.Arguments) > 0 && string(call.Arguments) != "null" {
				raw := call.Arguments
				// Some templates encode the arguments object as a string.
				var encoded string
				if json.Unmarshal(raw, &encoded) == nil {
					raw = json.RawMessage(encoded)
				}
				if err := json.Unmarshal(raw, &arguments); err != nil {
					return Result{}, false
				}
			}
		case strings.HasPrefix(inner, "<function="):
			dialect = QwenXML
			match := qwenFunction.FindStringSubmatch(inner)
			if match == nil || strings.TrimSpace(qwenParameter.ReplaceAllString(match[2], "")) != "" {
				return Result{}, false
			}
			name = match[1]
			tool, ok := tools[name]
			if !ok {
				return Result{}, false
			}
			for _, parameter := range qwenParameter.FindAllStringSubmatch(match[2], -1) {
				value, err := convert(tool, parameter[1], trimNewlines(parameter[2]), false)
				if err != nil {
					return Result{}, false
				}
				arguments[parameter[1]] = value
			}
		default:
			return Result{}, false
		}
		if result.Dialect != "" && result.Dialect != dialect {
			return Result{}, false
		}
		result.Dialect = dialect
		if _, ok := tools[name]; !ok {
			return Result{}, false
		}
		if !result.add(name, arguments) {
			return Result{}, false
		}
	}
	result.Text = strip(content, blocks)
	return result, true
}

func (r *Result) add(name string, arguments map[string]any) bool {
	if len(r.Calls) >= MaxCalls {
		return false
	}
	encoded, err := json.Marshal(arguments)
	if err != nil {
		return false
	}
	r.Calls = append(r.Calls, llm.ToolCallRequest{ID: fmt.Sprintf("%s_%d", r.Dialect, len(r.Calls)), Name: name, Arguments: string(encoded)})
	return true
}

// convert turns a parameter's text into the value its schema asks for. DSML
// marks non-string values with string="false"; Qwen XML leaves the type to
// the schema, as its reference parser does. A value that does not decode as
// the declared type fails the whole parse instead of being guessed.
func convert(tool llm.ToolDefinition, name, raw string, isJSON bool) (any, error) {
	kind := schemaType(tool, name)
	if !isJSON && (kind == "" || kind == "string") {
		return raw, nil
	}
	var value any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &value); err != nil {
		return nil, fmt.Errorf("parameter %s is not valid %s", name, kind)
	}
	return value, nil
}

func schemaType(tool llm.ToolDefinition, name string) string {
	properties, _ := tool.Parameters["properties"].(map[string]any)
	property, _ := properties[name].(map[string]any)
	kind, _ := property["type"].(string)
	return kind
}

func trimNewlines(value string) string {
	value = strings.TrimPrefix(value, "\r\n")
	value = strings.TrimPrefix(value, "\n")
	value = strings.TrimSuffix(value, "\n")
	return strings.TrimSuffix(value, "\r")
}

func strip(content string, blocks [][]int) string {
	var b strings.Builder
	last := 0
	for _, block := range blocks {
		b.WriteString(content[last:block[0]])
		last = block[1]
	}
	b.WriteString(content[last:])
	return strings.TrimSpace(b.String())
}
