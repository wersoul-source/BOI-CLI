package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/block/core"
)

// EnsureAgentIdentity loads an existing identity or asks the interactive user
// to create one. Existing invalid identity files are never overwritten.
func EnsureAgentIdentity(runtime *app.Runtime, in io.Reader, out io.Writer) (*coreblock.Identity, bool, error) {
	if runtime == nil || strings.TrimSpace(runtime.IdentityPath) == "" {
		return nil, false, fmt.Errorf("Agent identity path is not configured")
	}
	identity, err := coreblock.LoadIdentity(runtime.IdentityPath)
	if err == nil {
		return identity, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}

	reader := bufio.NewReader(in)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "🤖 Name your BOI Agent")
	fmt.Fprintln(out, "   Core Persona: boi (fixed)")
	fmt.Fprintf(out, "   Agent name [%s]: ", coreblock.DefaultAgentName)
	name, readErr := reader.ReadString('\n')
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return nil, false, fmt.Errorf("read Agent name: %w", readErr)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = coreblock.DefaultAgentName
	}
	identity, err = coreblock.NewIdentity(name, time.Now())
	if err != nil {
		return nil, false, err
	}
	if err := coreblock.SaveIdentity(runtime.IdentityPath, identity); err != nil {
		return nil, false, err
	}
	fmt.Fprintf(out, "   ✅ Agent named: %s\n", identity.Name)
	return identity, true, nil
}
