package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteProviderEnvPreservesUnrelatedValuesAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	original := "DATABASE_URL=postgres://local\nPSC_1_NAME=old\nPSC_1_API_KEY=old-secret\nFEATURE_FLAG=true\n"
	if err := os.WriteFile(envPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	backupPath, err := writeProviderEnvFile(envPath, []ConfiguredProvider{{Name: "new", APIKey: "secret", Endpoint: "https://example.invalid/v1", Model: "model"}})
	if err != nil {
		t.Fatal(err)
	}
	if backupPath == "" {
		t.Fatal("existing environment was not backed up")
	}
	backup, err := os.ReadFile(backupPath)
	if err != nil || string(backup) != original {
		t.Fatalf("backup=%q err=%v", backup, err)
	}
	updated, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	for _, want := range []string{"DATABASE_URL=postgres://local", "FEATURE_FLAG=true", providerEnvBegin, "PSC_1_NAME=new", "PSC_1_API_KEY=secret", providerEnvEnd} {
		if !strings.Contains(text, want) {
			t.Fatalf("updated environment missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "old-secret") {
		t.Fatal("old managed Provider value remained active")
	}
	assertPrivateFile(t, envPath)
	assertPrivateFile(t, backupPath)
}

func assertPrivateFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("%s permissions=%v, want no group/other access", path, info.Mode().Perm())
	}
}

func TestWriteProviderEnvRejectsLineInjection(t *testing.T) {
	_, err := writeProviderEnvFile(filepath.Join(t.TempDir(), ".env"), []ConfiguredProvider{{Name: "bad\nINJECTED=true", APIKey: "secret"}})
	if err == nil {
		t.Fatal("expected newline injection to be rejected")
	}
}

func TestMergeProviderEnvPreservesLargeUnrelatedLine(t *testing.T) {
	large := "LARGE_VALUE=" + strings.Repeat("x", 128<<10)
	merged := string(mergeProviderEnv([]byte(large+"\n"), []ConfiguredProvider{{Name: "fixture", APIKey: "secret"}}))
	if !strings.Contains(merged, large) {
		t.Fatal("large unrelated environment value was truncated")
	}
}
