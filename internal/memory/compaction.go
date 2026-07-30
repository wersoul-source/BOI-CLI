package memory

// CompactLevel defines how aggressive compaction should be
type CompactLevel int

const (
	CompactNone  CompactLevel = iota
	CompactMicro
	CompactAuto
	CompactFull
)

// Compactor handles context compaction
type Compactor struct {
	maxMessages  int
	maxOutputLen int
}

// NewCompactor creates a compactor with default settings
func NewCompactor() *Compactor {
	return &Compactor{
		maxMessages:  50,
		maxOutputLen: 2000,
	}
}

// Compact reduces context based on the level
func (c *Compactor) Compact(cm *ContextManager, level CompactLevel) string {
	switch level {
	case CompactMicro:
		return c.microCompact(cm)
	case CompactAuto:
		return c.autoCompact(cm)
	case CompactFull:
		return c.fullCompact(cm)
	default:
		return ""
	}
}

func (c *Compactor) microCompact(cm *ContextManager) string {
	for i := range cm.messages {
		msg := &cm.messages[i]
		if msg.Role == "tool" && len([]rune(msg.Content)) > c.maxOutputLen {
			msg.Content = string([]rune(msg.Content)[:c.maxOutputLen]) + "\n... [output trimmed]"
			msg.Tokens = EstimateTokens(msg.Content)
		}
	}
	return "Micro compaction applied"
}

func (c *Compactor) autoCompact(cm *ContextManager) string {
	msgs := cm.messages
	if len(msgs) <= c.maxMessages {
		return ""
	}
	keep := c.maxMessages / 2
	cm.messages = msgs[len(msgs)-keep:]
	cm.tokenUsed = 0
	for _, m := range cm.messages {
		cm.tokenUsed += m.Tokens
	}
	return "Auto compaction: kept last " + itoa(keep) + " messages"
}

func (c *Compactor) fullCompact(cm *ContextManager) string {
	summary := cm.Summarize()
	cm.messages = nil
	cm.tokenUsed = 0
	cm.AddMessage("system", cm.systemPrompt)
	cm.AddMessage("assistant", "[Previous conversation summarized]\n"+summary)
	return "Full compaction applied"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
