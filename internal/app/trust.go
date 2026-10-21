package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// trustFile records, per workspace, the exact environment.json content the
// user approved. It lives in the per-user directory, outside any repository,
// so a repository cannot trust itself.
const trustFile = "trusted-environments.json"

func environmentKey(boiDir string) string {
	abs, err := filepath.Abs(boiDir)
	if err != nil {
		return boiDir
	}
	return abs
}

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func loadTrust() map[string]string {
	trusted := map[string]string{}
	data, err := os.ReadFile(filepath.Join(GlobalDir(), trustFile))
	if err == nil {
		_ = json.Unmarshal(data, &trusted)
	}
	return trusted
}

// environmentTrusted reports whether the user trusted exactly this content
// for this workspace. Any edit to the file withdraws the trust.
func environmentTrusted(boiDir string, data []byte) bool {
	return loadTrust()[environmentKey(boiDir)] == digestOf(data)
}

// TrustEnvironment records the workspace's current environment.json as
// trusted and returns the MCP commands it allows to start.
func TrustEnvironment(boiDir string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(boiDir, EnvironmentFile))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", EnvironmentFile, err)
	}
	environment, err := LoadEnvironment(boiDir)
	if err != nil {
		return nil, err
	}
	var commands []string
	for _, item := range environment.Blocks {
		if item.MCP != nil {
			line := item.ID + ": " + item.MCP.Command
			for _, arg := range item.MCP.Args {
				line += " " + arg
			}
			commands = append(commands, line)
		}
	}
	trusted := loadTrust()
	trusted[environmentKey(boiDir)] = digestOf(data)
	encoded, err := json.MarshalIndent(trusted, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(GlobalDir(), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(GlobalDir(), trustFile), encoded, 0o600); err != nil {
		return nil, err
	}
	return commands, nil
}
