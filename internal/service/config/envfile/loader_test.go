package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHandlesQuotesExportAndPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\nexport BOI_T_A=\"quoted value\"\nBOI_T_B='single'\nBOI_T_C=plain\nBOI_T_D=kept\nnot a pair\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BOI_T_D", "from-shell")
	for _, key := range []string{"BOI_T_A", "BOI_T_B", "BOI_T_C"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"BOI_T_A": "quoted value", "BOI_T_B": "single", "BOI_T_C": "plain", "BOI_T_D": "from-shell"}
	for key, value := range want {
		if got := os.Getenv(key); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}
}
