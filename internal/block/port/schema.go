package port

import (
	"fmt"
	"math"
	"reflect"
	"sort"
)

// schemaKeywords is the JSON Schema subset BOI understands, enforces and
// sends to Providers. It is the common ground of the Chat Completions
// family; anything else (such as $ref, oneOf or format) is dropped before a
// definition leaves the host, so a Provider never receives a keyword it may
// reject and the host never advertises a rule it does not enforce.
var schemaKeywords = map[string]bool{
	"type": true, "description": true, "properties": true, "required": true,
	"items": true, "enum": true, "additionalProperties": true,
}

// ProviderSchema returns the Tool's argument schema reduced to the supported
// subset. It is the only schema adapters may send.
func (s ToolSpec) ProviderSchema() map[string]any {
	normalized, _ := normalizeSchema(s.Schema()).(map[string]any)
	if normalized == nil || normalized["type"] != "object" {
		return map[string]any{"type": "object"}
	}
	return normalized
}

func normalizeSchema(node any) any {
	object, ok := node.(map[string]any)
	if !ok {
		return node
	}
	out := map[string]any{}
	for key, value := range object {
		if !schemaKeywords[key] {
			continue
		}
		switch key {
		case "properties":
			properties, _ := value.(map[string]any)
			normalized := map[string]any{}
			for name, child := range properties {
				normalized[name] = normalizeSchema(child)
			}
			out[key] = normalized
		case "items":
			out[key] = normalizeSchema(value)
		case "additionalProperties":
			if flag, isBool := value.(bool); isBool {
				out[key] = flag
			}
		default:
			out[key] = value
		}
	}
	return out
}

// ValidateArguments checks model-supplied arguments against the Tool's
// schema before any authorization: a Provider's own strict mode is never
// trusted, so native calls and text envelopes pass the same check.
func (s ToolSpec) ValidateArguments(arguments map[string]any) error {
	if arguments == nil {
		arguments = map[string]any{}
	}
	if err := validateValue(s.ProviderSchema(), arguments, "arguments"); err != nil {
		return fmt.Errorf("invalid arguments for %s: %w", s.Name, err)
	}
	return nil
}

func validateValue(schema map[string]any, value any, path string) error {
	if kind, ok := schema["type"].(string); ok {
		if err := checkType(kind, value, path); err != nil {
			return err
		}
	}
	if allowed, ok := schema["enum"].([]any); ok && !containsValue(allowed, value) {
		return fmt.Errorf("%s must be one of %v", path, allowed)
	}
	if allowed, ok := schema["enum"].([]string); ok {
		text, _ := value.(string)
		if !containsString(allowed, text) {
			return fmt.Errorf("%s must be one of %v", path, allowed)
		}
	}
	switch typed := value.(type) {
	case map[string]any:
		properties, _ := schema["properties"].(map[string]any)
		for _, name := range requiredNames(schema["required"]) {
			if _, present := typed[name]; !present {
				return fmt.Errorf("%s.%s is required", path, name)
			}
		}
		names := make([]string, 0, len(typed))
		for name := range typed {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			child, known := properties[name].(map[string]any)
			if !known {
				if open, isBool := schema["additionalProperties"].(bool); isBool && !open {
					return fmt.Errorf("%s.%s is not a known argument", path, name)
				}
				continue
			}
			if err := validateValue(child, typed[name], path+"."+name); err != nil {
				return err
			}
		}
	case []any:
		if items, ok := schema["items"].(map[string]any); ok {
			for index, item := range typed {
				if err := validateValue(items, item, fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkType(kind string, value any, path string) error {
	ok := false
	switch kind {
	case "object":
		_, ok = value.(map[string]any)
	case "array":
		_, ok = value.([]any)
	case "string":
		_, ok = value.(string)
	case "boolean":
		_, ok = value.(bool)
	case "number":
		_, ok = value.(float64)
	case "integer":
		number, isNumber := value.(float64)
		ok = isNumber && number == math.Trunc(number)
	default:
		ok = true
	}
	if !ok {
		return fmt.Errorf("%s must be %s", path, kind)
	}
	return nil
}

func requiredNames(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		var names []string
		for _, item := range typed {
			if name, ok := item.(string); ok {
				names = append(names, name)
			}
		}
		return names
	}
	return nil
}

func containsValue(values []any, want any) bool {
	for _, value := range values {
		if reflect.DeepEqual(value, want) {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
