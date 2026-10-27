package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const AccessPolicySchemaVersion = 1

// MaxPolicyFolders bounds how many folders a policy may add. A policy a person
// cannot read through in one sitting is not a policy.
const MaxPolicyFolders = 8

// AccessPolicy names folders outside the primary workspace root that the Agent
// may use. The file is written by the user, and that is the permission: there
// is no way for a task to add a folder to it.
type AccessPolicy struct {
	SchemaVersion int `json:"schema_version"`
	// Folders are absolute, or relative to the workspace root.
	Folders []string `json:"folders"`
}

// AccessPolicyPath is where the policy lives inside .boi.
func AccessPolicyPath(boiDir string) string {
	return filepath.Join(boiDir, "workspace-policy.json")
}

// LoadAccessPolicy reads the policy. A missing file is an empty policy: the
// primary root alone, which is the default and the safest answer.
func LoadAccessPolicy(path string) (*AccessPolicy, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &AccessPolicy{SchemaVersion: AccessPolicySchemaVersion}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read workspace access policy: %w", err)
	}
	var policy AccessPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("parse workspace access policy: %w", err)
	}
	if policy.SchemaVersion != AccessPolicySchemaVersion {
		return nil, fmt.Errorf("unsupported workspace access policy schema %d", policy.SchemaVersion)
	}
	if len(policy.Folders) > MaxPolicyFolders {
		return nil, fmt.Errorf("workspace access policy lists %d folders; at most %d are allowed", len(policy.Folders), MaxPolicyFolders)
	}
	return &policy, nil
}

// Apply adds every folder of the policy to the sandbox. A relative folder is
// read against base, the workspace root. A folder that cannot be used stops
// start-up by name rather than being skipped in silence: a policy that half
// applied is one nobody can reason about.
func (p *AccessPolicy) Apply(sandbox *Sandbox, base string) error {
	if p == nil || sandbox == nil {
		return nil
	}
	for _, folder := range p.Folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			return fmt.Errorf("workspace access policy has an empty folder")
		}
		if !filepath.IsAbs(folder) {
			folder = filepath.Join(base, folder)
		}
		if err := guardPolicyFolder(folder); err != nil {
			return err
		}
		if err := sandbox.AddRoot(folder); err != nil {
			return fmt.Errorf("workspace access policy folder %q: %w", folder, err)
		}
	}
	return nil
}

// guardPolicyFolder refuses a folder that would end the boundary instead of
// widening it. Allowing the filesystem root, or a whole drive, is the same as
// having no sandbox, which the policy exists to avoid.
func guardPolicyFolder(folder string) error {
	cleaned := filepath.Clean(folder)
	if cleaned == string(filepath.Separator) || cleaned == filepath.VolumeName(cleaned)+string(filepath.Separator) {
		return fmt.Errorf("workspace access policy cannot allow the whole filesystem: %q", folder)
	}
	return nil
}
