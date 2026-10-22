package setup

import (
	"net/url"
	"strings"
)

// isLocalEndpoint reports whether baseURL points at this machine, where a
// server such as Ollama, llama.cpp or LM Studio needs no API key.
func isLocalEndpoint(baseURL string) bool {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}
