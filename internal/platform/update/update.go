package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	releaseOwner      = "wersoul-source"
	releaseRepository = "BOI-CLI"
	maxArchiveBytes   = 512 << 20
	maxChecksumsBytes = 1 << 20
)

var releaseHTTPClient = &http.Client{Timeout: 2 * time.Minute}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func FetchLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", releaseOwner, releaseRepository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "boi-cli")

	resp, err := releaseHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}

	return strings.TrimPrefix(rel.TagName, "v"), nil
}

func CompareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		va, vb := 0, 0
		if i < len(partsA) {
			fmt.Sscanf(partsA[i], "%d", &va)
		}
		if i < len(partsB) {
			fmt.Sscanf(partsB[i], "%d", &vb)
		}
		if va > vb {
			return 1
		}
		if va < vb {
			return -1
		}
	}
	return 0
}

func DownloadAndReplace(version string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	assetName := fmt.Sprintf("boi_%s_%s_%s.tar.gz", version, goos, goarch)

	baseURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s", releaseOwner, releaseRepository, version)
	url := baseURL + "/" + assetName

	resp, err := releaseHTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return fmt.Errorf("no binary found for %s/%s at v%s", goos, goarch, version)
		}
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "boi-upgrade-*")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, assetName)
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	written, err := io.Copy(f, io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		f.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if written > maxArchiveBytes {
		return fmt.Errorf("downloaded archive exceeds %d bytes", maxArchiveBytes)
	}

	checksums, err := downloadChecksums(baseURL + "/checksums.txt")
	if err != nil {
		return err
	}
	if err := verifyChecksumFile(tmpFile, assetName, checksums); err != nil {
		return err
	}

	extractedDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return err
	}

	if err := extractTarGz(tmpFile, extractedDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	newBinary, err := findExtractedBinary(extractedDir, goos)
	if err != nil {
		return err
	}

	if fi, _ := os.Stat(newBinary); fi.Size() == 0 {
		return fmt.Errorf("downloaded binary is empty")
	}

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve running binary: %w", err)
	}
	src, err := os.Open(newBinary)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.CreateTemp(filepath.Dir(binPath), ".boi-upgrade-*")
	if err != nil {
		return fmt.Errorf("create replacement binary: %w", err)
	}
	replacementPath := dst.Name()
	defer os.Remove(replacementPath)
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return fmt.Errorf("copy replacement binary: %w", err)
	}
	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		return fmt.Errorf("sync replacement binary: %w", err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("close replacement binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(replacementPath, 0o755); err != nil {
			return fmt.Errorf("make replacement executable: %w", err)
		}
	}

	backupPath := binPath + ".old"
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale binary backup: %w", err)
	}
	if err := os.Rename(binPath, backupPath); err != nil {
		return fmt.Errorf("back up running binary: %w", err)
	}
	if err := os.Rename(replacementPath, binPath); err != nil {
		_ = os.Rename(backupPath, binPath)
		return fmt.Errorf("install replacement binary: %w", err)
	}
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove binary backup after successful install: %w", err)
	}
	return nil
}

func findExtractedBinary(root, goos string) (string, error) {
	want := "boi"
	if goos == "windows" {
		want = "boi.exe"
	}
	var matches []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() && entry.Name() == want {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("inspect extracted archive: %w", err)
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("archive must contain exactly one %s binary; found %d", want, len(matches))
	}
	return matches[0], nil
}

func downloadChecksums(url string) ([]byte, error) {
	resp, err := releaseHTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download checksums returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumsBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read checksums: %w", err)
	}
	if len(data) > maxChecksumsBytes {
		return nil, fmt.Errorf("checksums file exceeds %d bytes", maxChecksumsBytes)
	}
	return data, nil
}

func verifyChecksumFile(path, assetName string, checksums []byte) error {
	want := ""
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimPrefix(fields[len(fields)-1], "*") == assetName {
			want = strings.ToLower(fields[0])
			break
		}
	}
	if len(want) != sha256.Size*2 {
		return fmt.Errorf("trusted checksum for %s is missing or invalid", assetName)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open archive for checksum: %w", err)
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return fmt.Errorf("hash downloaded archive: %w", err)
	}
	got := fmt.Sprintf("%x", hash.Sum(nil))
	if got != want {
		return fmt.Errorf("checksum mismatch for %s", assetName)
	}
	return nil
}

func extractTarGz(tarGzPath, dest string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			os.Chmod(target, os.FileMode(header.Mode))
		}
	}
	return nil
}
