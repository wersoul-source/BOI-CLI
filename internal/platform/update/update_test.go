package update

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal", a: "0.3.0", b: "0.3.0", want: 0},
		{name: "newer patch", a: "0.3.1", b: "0.3.0", want: 1},
		{name: "older minor", a: "0.2.9", b: "0.3.0", want: -1},
		{name: "missing patch", a: "1.0", b: "1.0.0", want: 0},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := CompareVersions(testCase.a, testCase.b); got != testCase.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d", testCase.a, testCase.b, got, testCase.want)
			}
		})
	}
}

func TestVerifyChecksumFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "boi_0.3.1_windows_amd64.tar.gz")
	content := []byte("verified archive")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	checksums := []byte(fmt.Sprintf("%x  %s\n", sum, filepath.Base(path)))
	if err := verifyChecksumFile(path, filepath.Base(path), checksums); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksumFile(path, filepath.Base(path), []byte("deadbeef  "+filepath.Base(path))); err == nil {
		t.Fatal("expected invalid checksum to fail")
	}
	wrong := sha256.Sum256([]byte("different"))
	if err := verifyChecksumFile(path, filepath.Base(path), []byte(fmt.Sprintf("%x  %s", wrong, filepath.Base(path)))); err == nil {
		t.Fatal("expected checksum mismatch to fail")
	}
}

func TestFindExtractedBinarySupportsWrappedArchive(t *testing.T) {
	root := t.TempDir()
	wrapped := filepath.Join(root, "boi_0.3.1_windows_amd64")
	if err := os.MkdirAll(wrapped, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(wrapped, "boi.exe")
	if err := os.WriteFile(want, []byte("binary"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := findExtractedBinary(root, "windows")
	if err != nil || got != want {
		t.Fatalf("findExtractedBinary()=(%q,%v), want %q", got, err, want)
	}
}
