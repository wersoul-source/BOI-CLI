package childenv

import (
	"slices"
	"testing"
)

func TestScrubDropsProviderSettingsOnly(t *testing.T) {
	got := Scrub([]string{"PATH=/bin", "PSC_1_API_KEY=secret", "psc_2_api_key=lower", "PSC_1_BASE_URL=https://x", "HOME=/root", "MY_PSC_VAR=kept"})
	want := []string{"PATH=/bin", "HOME=/root", "MY_PSC_VAR=kept"}
	if !slices.Equal(got, want) {
		t.Fatalf("Scrub = %q, want %q", got, want)
	}
}
