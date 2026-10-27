package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePolicy(t *testing.T, boiDir, body string) string {
	t.Helper()
	if err := os.MkdirAll(boiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := AccessPolicyPath(boiDir)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// No policy means the primary root alone, which is the default and the safest
// answer.
func TestMissingPolicyAllowsOnlyThePrimaryRoot(t *testing.T) {
	policy, err := LoadAccessPolicy(AccessPolicyPath(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.Folders) != 0 {
		t.Fatalf("folders = %q", policy.Folders)
	}
}

// A folder named by the policy is usable; everything else is still refused,
// so widening the policy does not turn the boundary off.
func TestPolicyAddsAFolderWithoutOpeningTheRest(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "project")
	shared := filepath.Join(base, "shared")
	outside := filepath.Join(base, "secret")
	for _, dir := range []string{root, shared, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(shared, "data.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "key.txt"), []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}

	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sandbox.ResolveExisting(filepath.Join(shared, "data.txt")); err == nil {
		t.Fatal("the shared folder was reachable before the policy allowed it")
	}

	writePolicy(t, filepath.Join(root, ".boi"), `{"schema_version":1,"folders":["../shared"]}`)
	policy, err := LoadAccessPolicy(AccessPolicyPath(filepath.Join(root, ".boi")))
	if err != nil {
		t.Fatal(err)
	}
	if err := policy.Apply(sandbox, root); err != nil {
		t.Fatal(err)
	}
	if _, err := sandbox.ResolveExisting(filepath.Join(shared, "data.txt")); err != nil {
		t.Fatalf("the policy folder is not reachable: %v", err)
	}
	if _, err := sandbox.ResolveExisting(filepath.Join(outside, "key.txt")); err == nil {
		t.Fatal("a folder the policy never named was reachable")
	}
}

// A grant covers one task and is withdrawn when it ends.
func TestATaskGrantDoesNotOutliveTheTask(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "project")
	scratch := filepath.Join(base, "scratch")
	for _, dir := range []string{root, scratch} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(scratch, "note.txt")
	if err := os.WriteFile(file, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sandbox.GrantForTask(scratch); err != nil {
		t.Fatal(err)
	}
	if _, err := sandbox.ResolveExisting(file); err != nil {
		t.Fatalf("the granted folder is not reachable: %v", err)
	}
	sandbox.ReleaseTaskGrants()
	if _, err := sandbox.ResolveExisting(file); err == nil {
		t.Fatal("the grant outlived the task")
	}
}

// Allowing the whole filesystem is not widening the boundary, it is removing
// it, which the policy exists to prevent.
func TestPolicyRefusesTheWholeFilesystem(t *testing.T) {
	root := t.TempDir()
	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	policy := &AccessPolicy{SchemaVersion: AccessPolicySchemaVersion, Folders: []string{string(filepath.Separator)}}
	if err := policy.Apply(sandbox, root); err == nil || !strings.Contains(err.Error(), "whole filesystem") {
		t.Fatalf("err = %v", err)
	}
}

// A policy that half applied is one nobody can reason about, so a folder that
// cannot be used is named instead of skipped.
func TestPolicyFailsLoudlyOnAMissingFolder(t *testing.T) {
	root := t.TempDir()
	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	policy := &AccessPolicy{SchemaVersion: AccessPolicySchemaVersion, Folders: []string{"does-not-exist"}}
	if err := policy.Apply(sandbox, root); err == nil {
		t.Fatal("a missing policy folder was accepted")
	}
}

func TestPolicyRejectsAnUnknownSchemaAndTooManyFolders(t *testing.T) {
	boiDir := filepath.Join(t.TempDir(), ".boi")
	writePolicy(t, boiDir, `{"schema_version":99,"folders":[]}`)
	if _, err := LoadAccessPolicy(AccessPolicyPath(boiDir)); err == nil {
		t.Fatal("an unknown schema was accepted")
	}
	many := make([]string, MaxPolicyFolders+1)
	for i := range many {
		many[i] = `"x"`
	}
	writePolicy(t, boiDir, `{"schema_version":1,"folders":[`+strings.Join(many, ",")+`]}`)
	if _, err := LoadAccessPolicy(AccessPolicyPath(boiDir)); err == nil {
		t.Fatal("an oversized policy was accepted")
	}
}

// A symlink inside an allowed folder must not become a way out of it.
func TestAnAllowedFolderCannotBeEscapedBySymlink(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "project")
	shared := filepath.Join(base, "shared")
	secret := filepath.Join(base, "secret")
	for _, dir := range []string{root, shared, secret} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(secret, "key.txt"), []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(shared, "way-out")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sandbox.AddRoot(shared); err != nil {
		t.Fatal(err)
	}
	if _, err := sandbox.ResolveExisting(filepath.Join(shared, "way-out", "key.txt")); err == nil {
		t.Fatal("a symlink led out of an allowed folder")
	}
}

// A relative path always means the workspace the user is working in, never a
// folder that happens to be allowed.
func TestARelativePathAlwaysMeansThePrimaryRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "project")
	shared := filepath.Join(base, "shared")
	for _, dir := range []string{root, shared} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(shared, "note.txt"), []byte("shared"), 0o600); err != nil {
		t.Fatal(err)
	}
	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := sandbox.AddRoot(shared); err != nil {
		t.Fatal(err)
	}
	if _, err := sandbox.ResolveExisting("note.txt"); err == nil {
		t.Fatal("a relative path reached into another allowed folder")
	}
}
