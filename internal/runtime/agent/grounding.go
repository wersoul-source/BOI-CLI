package agent

import "strings"

// hasCodeFence reports whether text offers code in a fenced block, the usual
// shape of a fix that is shown instead of applied.
func hasCodeFence(text string) bool { return strings.Contains(text, "```") }

// maxProbedNames bounds how many names in one text are checked.
const maxProbedNames = 20

// pathTokens splits text into the tokens that could name a file, in any human
// language: anything a path is made of, cut at everything else. ':' is kept so
// a web address arrives as one token and can be recognised and rejected,
// rather than arriving as a bare "//example.com/docs" that reads like a path.
func pathTokens(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !(r == '.' || r == '_' || r == '-' || r == '/' || r == '\\' || r == ':' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r > 127)
	})
}

// namesWorkspaceFile reports whether text names an existing workspace file,
// such as calc.go or docs/guide.md. It looks at tokens that carry a file
// extension or a directory separator, so it works in any human language.
func namesWorkspaceFile(text string, exists func(string) bool) bool {
	if exists == nil {
		return false
	}
	probed := 0
	for _, token := range pathTokens(text) {
		token = strings.Trim(token, ".-_/\\:")
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

// namesNewWorkspaceFile reports whether text names a file or folder that does
// not exist yet: the shape of a task asking for something to be produced.
// namesWorkspaceFile cannot see these, because it matches only files already
// on disk, so "write report.json" was never challenged however little the
// model did. Only the task text is checked, never the answer, so a model
// mentioning a path in prose does not trigger a challenge by itself.
func namesNewWorkspaceFile(text string, exists func(string) bool) bool {
	if exists == nil {
		return false
	}
	probed := 0
	for _, token := range pathTokens(text) {
		if !looksLikeFileTarget(token) {
			continue
		}
		if probed++; probed > maxProbedNames {
			return false
		}
		if !exists(strings.Trim(token, "./\\")) {
			return true
		}
	}
	return false
}

// looksLikeFileTarget reports whether token has the shape of a path a task
// asks for. A web address is not a target, and neither is a version such as
// 2.6.1, so an extension must be short and alphabetic and the name must carry
// a letter somewhere.
func looksLikeFileTarget(token string) bool {
	lower := strings.ToLower(token)
	if strings.Contains(lower, "://") || strings.HasPrefix(lower, "http") || strings.Contains(token, ":") {
		return false
	}
	token = strings.Trim(token, "./\\")
	if len(token) < 3 || !hasLetter(token) {
		return false
	}
	if strings.ContainsAny(token, "/\\") {
		return true
	}
	dot := strings.LastIndex(token, ".")
	if dot <= 0 || dot == len(token)-1 {
		return false
	}
	extension := token[dot+1:]
	if len(extension) > 5 {
		return false
	}
	for _, r := range extension {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

// hasLetter keeps version numbers such as 1.2.3 from looking like file names.
func hasLetter(token string) bool {
	for _, r := range token {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r > 127 {
			return true
		}
	}
	return false
}
