package agent

import "strings"

// hasCodeFence reports whether text offers code in a fenced block, the usual
// shape of a fix that is shown instead of applied.
func hasCodeFence(text string) bool { return strings.Contains(text, "```") }

// maxProbedNames bounds how many names in one text are checked.
const maxProbedNames = 20

// namesWorkspaceFile reports whether text names an existing workspace file,
// such as calc.go or docs/guide.md. It looks at tokens that carry a file
// extension or a directory separator, so it works in any human language.
func namesWorkspaceFile(text string, exists func(string) bool) bool {
	if exists == nil {
		return false
	}
	probed := 0
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return !(r == '.' || r == '_' || r == '-' || r == '/' || r == '\\' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r > 127)
	}) {
		token = strings.Trim(token, ".-_/\\")
		if !strings.ContainsAny(token, "./\\") || len(token) < 3 {
			continue
		}
		if probed++; probed > maxProbedNames {
			return false
		}
		if exists(token) {
			return true
		}
	}
	return false
}
