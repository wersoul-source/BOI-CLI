package process

import (
	"fmt"
	"regexp"
	"strings"
)

// Sandbox is a best-effort deny-list over the command text. It is NOT an
// isolation boundary: obfuscated commands (variable expansion, encoded
// payloads, nested interpreters) evade it by design. Authorization is
// enforced by the Capability Broker approval step and the workspace path
// boundary; this list only stops obviously destructive commands early.
type Sandbox struct {
	denyPatterns []*regexp.Regexp
}

// NewSandbox creates a sandbox with default safety rules
func NewSandbox() *Sandbox {
	patterns := []string{
		// recursive rm aimed at root, home, $HOME or a bare glob
		`\brm\s+(?:-\S+\s+)*(?:-\S*[rR]\S*|--recursive)\s+(?:-\S+\s+)*(?:/|~|\$HOME|\*)`,
		`(?:^|[;&|(\s])(?:sudo|doas)\s`,
		`\bmkfs(?:\.|\s)`,
		`\bdd\s+.*\bof=/dev/`,
		`\bdd\s+if=`,
		// any producer piped into a shell
		`\|\s*(?:sudo\s+)?(?:ba|z|da|k)?sh\b`,
		`>\s*/dev/(?:sd|nvme|hd|disk|mmcblk)`,
		`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`,
	}
	compiled := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		compiled[i] = regexp.MustCompile(`(?i)` + p)
	}
	return &Sandbox{denyPatterns: compiled}
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
