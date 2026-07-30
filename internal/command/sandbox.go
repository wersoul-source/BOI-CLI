package command

import (
	"fmt"
	"regexp"
	"strings"
)

// Sandbox checks commands against deny patterns
type Sandbox struct {
	denyPatterns []*regexp.Regexp
}

// NewSandbox creates a sandbox with default safety rules
func NewSandbox() *Sandbox {
	return &Sandbox{
		denyPatterns: []*regexp.Regexp{
			regexp.MustCompile(`rm\s+-rf\s+/`),
			regexp.MustCompile(`sudo\s+`),
			regexp.MustCompile(`mkfs\.`),
			regexp.MustCompile(`dd\s+if=`),
			regexp.MustCompile(`curl\s+.*\|\s*(ba)?sh`),
			regexp.MustCompile(`wget\s+.*\|\s*(ba)?sh`),
			regexp.MustCompile(`>\s*/dev/`),
			regexp.MustCompile(`:\(\)\{ :\|:& \};:`),
		},
	}
}

// Allow checks if a command is safe to execute
func (s *Sandbox) Allow(command string) error {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return fmt.Errorf("empty command")
	}

	for _, pattern := range s.denyPatterns {
		if pattern.MatchString(trimmed) {
			return fmt.Errorf("dangerous command blocked: %q", trimmed)
		}
	}

	return nil
}

// IsBlocked returns true if the command matches a deny pattern
func (s *Sandbox) IsBlocked(command string) bool {
	return s.Allow(command) != nil
}
