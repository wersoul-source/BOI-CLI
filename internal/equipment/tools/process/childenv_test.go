package process

import (
	"runtime"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

// .env is loaded into BOI's own environment. A command the model proposes —
// or repository code it runs, such as a test suite — must not be able to
// print the Provider API key back into the conversation.
func TestCommandsDoNotSeeProviderKeys(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell")
	}
	t.Setenv("PSC_1_API_KEY", "audit-secret-value")
	t.Setenv("BOI_AUDIT_KEPT", "visible")
	sandbox, err := workspace.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewExecutor(WithWorkspace(sandbox)).Run("env")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "audit-secret-value") {
		t.Fatal("a command saw the Provider API key")
	}
	if !strings.Contains(out, "BOI_AUDIT_KEPT=visible") {
		t.Fatal("the rest of the environment must still reach the command")
	}
}
