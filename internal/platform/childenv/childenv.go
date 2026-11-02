// Package childenv builds the environment BOI hands to a child process.
package childenv

import "strings"

// Scrub returns parent without BOI's Provider settings (every PSC_* entry).
// BOI loads .env into its own environment, so passing that environment on
// whole would give a command or an MCP server the user's model API keys,
// which the file Tools refuse to read for exactly that reason. Every place
// that starts a child process uses this one filter, so none can forget it.
func Scrub(parent []string) []string {
	env := make([]string, 0, len(parent))
	for _, item := range parent {
		if strings.HasPrefix(strings.ToUpper(item), "PSC_") {
			continue
		}
		env = append(env, item)
	}
	return env
}
