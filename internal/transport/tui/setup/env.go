package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/workspace"
)

const (
	providerEnvBegin = "# BEGIN BOI CLI PROVIDERS"
	providerEnvEnd   = "# END BOI CLI PROVIDERS"
)

func writeProviderEnv(providers []ConfiguredProvider) (string, string, error) {
	root, err := workspace.DetectRoot()
	if err != nil {
		root, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("resolve environment directory: %w", err)
		}
	}
	envPath := filepath.Join(root, ".env")
	if err := workspace.EnsureLocalGitExcludes(root, ".env", ".env.boi-backup-*"); err != nil {
		return envPath, "", fmt.Errorf("protect Provider secrets from Git: %w", err)
	}
	backup, err := writeProviderEnvFile(envPath, providers)
	return envPath, backup, err
}

func writeProviderEnvFile(envPath string, providers []ConfiguredProvider) (string, error) {
	for _, provider := range providers {
		for label, value := range map[string]string{"name": provider.Name, "API key": provider.APIKey, "endpoint": provider.Endpoint, "model": provider.Model} {
			if strings.ContainsAny(value, "\r\n\x00") {
				return "", fmt.Errorf("Provider %s contains an invalid %s value", provider.Name, label)
			}
		}
	}

	existing, err := os.ReadFile(envPath)
	hasExisting := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read existing environment: %w", err)
	}
	content := mergeProviderEnv(existing, providers)
	dir := filepath.Dir(envPath)
	tmp, err := os.CreateTemp(dir, ".boi-env-*")
	if err != nil {
		return "", fmt.Errorf("create secure environment file: %w", err)
	}
	tmpPath := tmp.Name()
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("secure environment permissions: %w", err)
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("write environment: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("sync environment: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close environment: %w", err)
	}

	backupPath := ""
	if hasExisting {
		backupPath = envPath + ".boi-backup-" + time.Now().UTC().Format("20060102T150405.000000000")
		if err := os.Rename(envPath, backupPath); err != nil {
			return "", fmt.Errorf("back up existing environment: %w", err)
		}
		if err := os.Chmod(backupPath, 0o600); err != nil {
			_ = os.Rename(backupPath, envPath)
			return "", fmt.Errorf("secure environment backup permissions: %w", err)
		}
	}
	if err := os.Rename(tmpPath, envPath); err != nil {
		if backupPath != "" {
			_ = os.Rename(backupPath, envPath)
		}
		return "", fmt.Errorf("replace environment: %w", err)
	}
	keepTemp = false
	if err := os.Chmod(envPath, 0o600); err != nil {
		return backupPath, fmt.Errorf("enforce environment permissions: %w", err)
	}
	return backupPath, nil
}

func mergeProviderEnv(existing []byte, providers []ConfiguredProvider) []byte {
	var kept []string
	existingText := string(existing)
	beginIndex := strings.Index(existingText, providerEnvBegin)
	endIndex := strings.Index(existingText, providerEnvEnd)
	hasCompleteManagedBlock := beginIndex >= 0 && endIndex > beginIndex
	inManagedBlock := false
	for _, line := range strings.Split(strings.ReplaceAll(existingText, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if hasCompleteManagedBlock && trimmed == providerEnvBegin {
			inManagedBlock = true
			continue
		}
		if hasCompleteManagedBlock && trimmed == providerEnvEnd {
			inManagedBlock = false
			continue
		}
		if inManagedBlock || isProviderEnvAssignment(trimmed) {
			continue
		}
		kept = append(kept, line)
	}
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}
	if len(kept) > 0 {
		kept = append(kept, "")
	}
	kept = append(kept, providerEnvBegin, "# Managed by 'boi setup'; unrelated variables are preserved.")
	for i, provider := range providers {
		n := i + 1
		kept = append(kept,
			fmt.Sprintf("PSC_%d_NAME=%s", n, provider.Name),
			fmt.Sprintf("PSC_%d_API_KEY=%s", n, provider.APIKey),
			fmt.Sprintf("PSC_%d_BASE_URL=%s", n, provider.Endpoint),
			fmt.Sprintf("PSC_%d_MODEL=%s", n, provider.Model),
		)
	}
	kept = append(kept, providerEnvEnd)
	return []byte(strings.Join(kept, "\n") + "\n")
}

func isProviderEnvAssignment(line string) bool {
	if !strings.HasPrefix(line, "PSC_") {
		return false
	}
	key, _, ok := strings.Cut(line, "=")
	if !ok {
		return false
	}
	parts := strings.Split(key, "_")
	if len(parts) < 3 || strings.Trim(parts[1], "0123456789") != "" || parts[1] == "" {
		return false
	}
	suffix := strings.Join(parts[2:], "_")
	switch suffix {
	case "NAME", "API_KEY", "BASE_URL", "MODEL":
		return true
	default:
		return false
	}
}
