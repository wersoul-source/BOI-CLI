package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func writeTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractTarGzSkipsTraversalEntries(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	archive := filepath.Join(root, "a.tar.gz")
	writeTarGz(t, archive, map[string]string{"boi/boi": "bin", "../escaped": "evil"})
	if err := extractTarGz(archive, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "escaped")); err == nil {
		t.Fatal("traversal entry escaped the destination")
	}
	if b, err := os.ReadFile(filepath.Join(dest, "boi", "boi")); err != nil || string(b) != "bin" {
		t.Fatalf("regular entry not extracted: %v", err)
	}
}

func TestExtractTarGzRejectsCorruptArchive(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.tar.gz")
	_ = os.WriteFile(bad, []byte("not gzip"), 0o600)
	if err := extractTarGz(bad, t.TempDir()); err == nil {
		t.Fatal("corrupt archive must error")
	}
}

func TestVerifyChecksumFileRejectsMismatchAndMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset")
	_ = os.WriteFile(path, []byte("data"), 0o600)
	wrong := fmt.Sprintf("%x  asset\n", sha256.Sum256([]byte("other")))
	if err := verifyChecksumFile(path, "asset", []byte(wrong)); err == nil {
		t.Fatal("checksum mismatch must fail")
	}
	if err := verifyChecksumFile(path, "asset", []byte("deadbeef  asset\n")); err == nil {
		t.Fatal("short checksum must fail")
	}
	if err := verifyChecksumFile(path, "asset", []byte(wrong[:0])); err == nil {
		t.Fatal("missing checksum must fail")
	}
}

func TestDownloadChecksumsBounds(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("abc")) })
	mux.HandleFunc("/missing", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) })
	mux.HandleFunc("/huge", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, maxChecksumsBytes+10))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if b, err := downloadChecksums(srv.URL + "/ok"); err != nil || string(b) != "abc" {
		t.Fatalf("ok: %q %v", b, err)
	}
	if _, err := downloadChecksums(srv.URL + "/missing"); err == nil {
		t.Fatal("404 must error")
	}
	if _, err := downloadChecksums(srv.URL + "/huge"); err == nil {
		t.Fatal("oversized checksums must error")
	}
}
