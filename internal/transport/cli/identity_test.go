package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/block/core"
)

func TestEnsureAgentIdentityCreatesThenLoads(t *testing.T) {
	runtime := &app.Runtime{IdentityPath: filepath.Join(t.TempDir(), coreblock.IdentityFilename)}
	var output bytes.Buffer
	identity, created, err := EnsureAgentIdentity(runtime, strings.NewReader("แก้ว\n"), &output)
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	if !created || identity.Name != "แก้ว" {
		t.Fatalf("identity = %#v, created = %v", identity, created)
	}

	output.Reset()
	loaded, created, err := EnsureAgentIdentity(runtime, strings.NewReader("ignored\n"), &output)
	if err != nil {
		t.Fatalf("load identity: %v", err)
	}
	if created || loaded.ID != identity.ID || loaded.Name != identity.Name {
		t.Fatalf("loaded = %#v, created = %v", loaded, created)
	}
}

func TestEnsureAgentIdentityUsesDefaultOnEOF(t *testing.T) {
	runtime := &app.Runtime{IdentityPath: filepath.Join(t.TempDir(), coreblock.IdentityFilename)}
	identity, created, err := EnsureAgentIdentity(runtime, strings.NewReader(""), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !created || identity.Name != coreblock.DefaultAgentName {
		t.Fatalf("identity = %#v, created = %v", identity, created)
	}
}
