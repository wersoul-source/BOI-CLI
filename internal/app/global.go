package app

import (
	"os"
	"path/filepath"
	"strings"

	coreblock "github.com/boi-family/boi-cli/internal/core"
)

// GlobalDir is the per-user BOI directory shared by every workspace, so a
// Provider configured and qualified once works in each new project.
// BOI_CONFIG_DIR overrides it (tests use it for isolation).
func GlobalDir() string {
	if dir := strings.TrimSpace(os.Getenv("BOI_CONFIG_DIR")); dir != "" {
		return dir
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "boi")
}

// GlobalEnvPath is the shared .env, read after the workspace .env so a
// workspace value always wins.
func GlobalEnvPath() string {
	if dir := GlobalDir(); dir != "" {
		return filepath.Join(dir, ".env")
	}
	return ""
}

// profileDirs lists where qualification profiles are looked up, workspace first.
func profileDirs(boiDir string) []string {
	dirs := []string{boiDir}
	if global := GlobalDir(); global != "" && filepath.Clean(global) != filepath.Clean(boiDir) {
		dirs = append(dirs, global)
	}
	return dirs
}

// LoadProfile returns the first valid profile for target and the path it came from.
func LoadProfile(boiDir string, target coreblock.ProviderTarget) (*coreblock.CapabilityProfile, string, error) {
	var lastErr error
	for _, dir := range profileDirs(boiDir) {
		path := coreblock.CapabilityProfilePath(dir, target)
		profile, err := coreblock.LoadCapabilityProfile(path)
		if err == nil {
			return profile, path, nil
		}
		lastErr = err
	}
	return nil, "", lastErr
}

// SaveProfile stores a profile in the workspace and in the global cache.
// The workspace copy is authoritative; a global write failure is ignored.
func SaveProfile(boiDir string, target coreblock.ProviderTarget, profile *coreblock.CapabilityProfile) (string, error) {
	path := coreblock.CapabilityProfilePath(boiDir, target)
	if err := coreblock.SaveCapabilityProfile(path, profile); err != nil {
		return "", err
	}
	for _, dir := range profileDirs(boiDir)[1:] {
		_ = coreblock.SaveCapabilityProfile(coreblock.CapabilityProfilePath(dir, target), profile)
	}
	return path, nil
}
