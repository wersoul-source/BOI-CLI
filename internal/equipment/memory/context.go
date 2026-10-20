package memory

import (
	"strings"
)

// ContextManager tracks conversation context and token budget
type ContextManager struct {
	systemPrompt string
	messages     []ContextMessage
	tokenBudget  int
	tokenUsed    int
}

// ContextMessage is a message in the context window
type ContextMessage struct {
	Role    string
	Content string
	Tokens  int
}

// NewContextManager creates a context manager with a budget
func NewContextManager(systemPrompt string, budget int) *ContextManager {
	cm := &ContextManager{
		systemPrompt: systemPrompt,
		tokenBudget:  budget,
	}
	cm.AddMessage("system", systemPrompt)
	return cm
}

// EstimateTokens estimates token count (rough: ~4 chars per token)
func EstimateTokens(text string) int {
	return len([]rune(text)) / 4
}

// AddMessage adds a message and tracks tokens
func (cm *ContextManager) AddMessage(role, content string) {
	tokens := EstimateTokens(content)
	cm.messages = append(cm.messages, ContextMessage{
		Role:    role,
		Content: content,
		Tokens:  tokens,
	})
	cm.tokenUsed += tokens
}

// IsOverBudget checks if we've exceeded the token budget
func (cm *ContextManager) IsOverBudget() bool {
	return cm.tokenUsed > cm.tokenBudget
}

// RemainingTokens returns available budget
func (cm *ContextManager) RemainingTokens() int {
	remaining := cm.tokenBudget - cm.tokenUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// TotalTokens returns tokens used so far
func (cm *ContextManager) TotalTokens() int {
	return cm.tokenUsed
}

// Budget returns the total budget
func (cm *ContextManager) Budget() int {
	return cm.tokenBudget
}

// GetMessages returns all messages for LLM context
func (cm *ContextManager) GetMessages() []ContextMessage {
	return cm.messages
}

// Summarize generates a compact version of the context
func (cm *ContextManager) Summarize() string {
	var sb strings.Builder
	sb.WriteString("Previous conversation summary:\n")
	for _, m := range cm.messages {
		if m.Role == "system" {
			continue
		}
		prefix := "Assistant: "
		if m.Role == "user" {
			prefix = "User: "
		}
		runes := []rune(m.Content)
		if len(runes) > 200 {
			sb.WriteString(prefix + string(runes[:200]) + "...\n")
		} else {
			sb.WriteString(prefix + m.Content + "\n")
		}
	}
	return sb.String()
}
