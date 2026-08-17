package update

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func FetchLatestVersion() (string, error) {
	url := "https://api.github.com/repos/boi-family/boi-cli/releases/latest"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "boi-cli")

	resp, err := http.DefaultClient.Do(req)
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

	url := fmt.Sprintf("https://github.com/boi-family/boi-cli/releases/download/v%s/%s", version, assetName)

	resp, err := http.Get(url)
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

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	f.Close()

	extractedDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return err
	}

	if err := extractTarGz(tmpFile, extractedDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	newBinary := filepath.Join(extractedDir, "boi")
	if goos == "windows" {
		newBinary += ".exe"
	}

	if _, err := os.Stat(newBinary); err != nil {
		return fmt.Errorf("binary not found in archive")
	}

	if fi, _ := os.Stat(newBinary); fi.Size() == 0 {
		return fmt.Errorf("downloaded binary is empty")
	}

	binPath, _ := os.Executable()
	if err := os.Rename(binPath, binPath+".old"); err != nil {
		return fmt.Errorf("cannot replace running binary: %w", err)
	}

	src, err := os.Open(newBinary)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(binPath)
	if err != nil {
		os.Rename(binPath+".old", binPath)
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Rename(binPath+".old", binPath)
		return err
	}

	os.Remove(binPath + ".old")
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

func RestartSelf() {
	binPath, _ := os.Executable()
	cmd := exec.Command(binPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err == nil {
		os.Exit(0)
	}
	os.Exit(1)
}
