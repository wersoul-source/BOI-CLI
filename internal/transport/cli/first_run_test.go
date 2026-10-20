package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/app"
)

func TestOfferQualificationCanBeDeclined(t *testing.T) {
	t.Setenv("PSC_1_NAME", "openai")
	t.Setenv("PSC_1_API_KEY", "test-key")
	t.Setenv("PSC_1_MODEL", "m")
	runtime := &app.Runtime{BoiDir: filepath.Join(t.TempDir(), ".boi")}
	var out bytes.Buffer
	offerQualification(runtime, strings.NewReader("n\n"), &out)
	if !strings.Contains(out.String(), "not qualified yet") || !strings.Contains(out.String(), "boi provider qualify openai") {
		t.Fatalf("unexpected prompt output: %q", out.String())
	}
}

func TestOfferQualificationIsSilentWithoutProviders(t *testing.T) {
	for _, key := range []string{"PSC_1_NAME", "PSC_1_API_KEY", "PSC_1_MODEL"} {
		t.Setenv(key, "")
	}
	var out bytes.Buffer
	offerQualification(&app.Runtime{BoiDir: t.TempDir()}, strings.NewReader(""), &out)
	if out.Len() != 0 {
		t.Fatalf("prompted without a configured Provider: %q", out.String())
	}
}
