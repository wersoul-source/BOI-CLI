package agentfolder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
)

const ManifestSchemaVersion = 1

type Store struct {
	root string
	now  func() time.Time
}

type TaskManifest struct {
	SchemaVersion      int                       `json:"schema_version"`
	TaskID             string                    `json:"task_id"`
	PlanID             string                    `json:"plan_id"`
	Outcome            string                    `json:"outcome"`
	StopReason         agent.StopReason          `json:"stop_reason"`
	StartedAt          time.Time                 `json:"started_at"`
	FinishedAt         time.Time                 `json:"finished_at"`
	Provider           string                    `json:"provider,omitempty"`
	Model              string                    `json:"model,omitempty"`
	ProviderProfileRef string                    `json:"provider_profile_ref,omitempty"`
	IdempotencyKeyHash string                    `json:"idempotency_key_hash,omitempty"`
	Usage              ManifestUsage             `json:"usage"`
	Artifacts          []agent.ArtifactReference `json:"artifacts"`
	Evidence           []ManifestEvidence        `json:"evidence"`
	Retention          RetentionPolicy           `json:"retention"`
}

type ManifestUsage struct {
	Steps         int `json:"steps"`
	InputTokens   int `json:"input_tokens"`
	OutputTokens  int `json:"output_tokens"`
	ProviderCalls int `json:"provider_calls"`
	ToolCalls     int `json:"tool_calls"`
}

type ManifestEvidence struct {
	Kind   string `json:"kind"`
	Tool   string `json:"tool,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Path   string `json:"path,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

type RetentionPolicy struct {
	Class   string `json:"class"`
	Cleanup string `json:"cleanup"`
}

type checkpoint struct {
	SchemaVersion int              `json:"schema_version"`
	TaskID        string           `json:"task_id"`
	PlanID        string           `json:"plan_id"`
	PlanRevision  int              `json:"plan_revision"`
	PlanStatus    agent.PlanStatus `json:"plan_status"`
	Phase         agent.AgentPhase `json:"phase"`
	Step          int              `json:"step"`
	StopReason    agent.StopReason `json:"stop_reason,omitempty"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

func NewStore(root string) (*Store, error) {
	store, err := OpenStore(root)
	if err != nil {
		return nil, err
	}
	if err := store.Ensure(); err != nil {
		return nil, err
	}
	return store, nil
}

// OpenStore resolves an Agent Folder without creating it. This keeps read-only
// CLI commands side-effect free; Begin and explicit initialization still call
// Ensure before writing task state.
func OpenStore(root string) (*Store, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("Agent Folder root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Agent Folder root: %w", err)
	}
	return &Store{root: filepath.Clean(abs), now: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *Store) Root() string       { return s.root }
func (s *Store) BinRoot() string    { return filepath.Join(s.root, "bin") }
func (s *Store) OutputRoot() string { return filepath.Join(s.root, "output") }

func (s *Store) Ensure() error {
	for _, path := range []string{s.root, s.BinRoot(), s.OutputRoot()} {
		if err := ensureRealDirectory(path); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Begin(task string, plan *agent.TaskPlan) (*agent.TaskSession, error) {
	if plan == nil || strings.TrimSpace(plan.ID) == "" {
		return nil, fmt.Errorf("Task Plan is required before creating an Agent Folder scope")
	}
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	started := s.now()
	baseID := taskID(started, task, plan.ID)
	id := baseID
	for suffix := 2; ; suffix++ {
		binDir := filepath.Join(s.BinRoot(), id)
		outputDir := filepath.Join(s.OutputRoot(), id)
		if _, binErr := os.Lstat(binDir); os.IsNotExist(binErr) {
			if _, outputErr := os.Lstat(outputDir); os.IsNotExist(outputErr) {
				if err := os.Mkdir(binDir, 0o755); err != nil {
					return nil, fmt.Errorf("create task bin directory: %w", err)
				}
				if err := os.Mkdir(outputDir, 0o755); err != nil {
					return nil, fmt.Errorf("create task output directory: %w", err)
				}
				session := &agent.TaskSession{ID: id, BinDir: binDir, OutputDir: outputDir, StartedAt: started}
				if err := s.writeCheckpoint(session, checkpoint{SchemaVersion: 1, TaskID: id, PlanID: plan.ID, PlanRevision: plan.Revision, PlanStatus: plan.Status, Phase: agent.PhaseObserve, UpdatedAt: started}); err != nil {
					return nil, err
				}
				return session, nil
			}
		}
		id = fmt.Sprintf("%s-%02d", baseID, suffix)
	}
}

func (s *Store) RecordEvent(session *agent.TaskSession, event agent.EngineEvent) error {
	if err := s.validateSession(session); err != nil {
		return err
	}
	value := checkpoint{SchemaVersion: 1, TaskID: session.ID, Phase: event.Phase, Step: event.Step, StopReason: event.StopReason, UpdatedAt: s.now()}
	if event.Plan != nil {
		value.PlanID = event.Plan.ID
		value.PlanRevision = event.Plan.Revision
		value.PlanStatus = event.Plan.Status
	}
	return s.writeCheckpoint(session, value)
}

func (s *Store) Finalize(session *agent.TaskSession, result *agent.AgentResult) error {
	if err := s.validateSession(session); err != nil {
		return err
	}
	if result == nil || result.Plan == nil {
		return fmt.Errorf("Agent Result and Task Plan are required")
	}
	artifacts, err := s.collectArtifacts(session)
	if err != nil {
		return err
	}
	outcome := "failed"
	manifestDir := session.BinDir
	retention := RetentionPolicy{Class: "diagnostic", Cleanup: "explicit_bin_cleanup_only"}
	if result.StopReason == agent.StopCompleted {
		outcome = "completed"
		manifestDir = session.OutputDir
		retention = RetentionPolicy{Class: "deliverable", Cleanup: "manual_only_never_automatic"}
	}
	manifest := TaskManifest{
		SchemaVersion: ManifestSchemaVersion, TaskID: session.ID, PlanID: result.Plan.ID,
		Outcome: outcome, StopReason: result.StopReason, StartedAt: session.StartedAt, FinishedAt: s.now(),
		Provider: result.Provider, Model: result.Model, ProviderProfileRef: result.ProviderProfileRef, IdempotencyKeyHash: result.IdempotencyKeyHash,
		Usage:     ManifestUsage{Steps: result.Steps, InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, ProviderCalls: result.Usage.ProviderCalls, ToolCalls: result.Usage.ToolCalls},
		Artifacts: artifacts, Evidence: manifestEvidence(result, artifacts), Retention: retention,
	}
	manifestPath := filepath.Join(manifestDir, "manifest.json")
	if err := writeJSONAtomic(manifestPath, manifest); err != nil {
		return fmt.Errorf("write task manifest: %w", err)
	}
	result.TaskID = session.ID
	result.Manifest = s.relativePath(manifestPath)
	result.Artifacts = artifacts
	return s.RecordEvent(session, agent.EngineEvent{Kind: "stop", Phase: agent.PhaseStopped, Step: result.Steps, StopReason: result.StopReason, Plan: result.Plan})
}

// CleanupBinBefore removes only direct task directories under bin when apply
// is true. Output is deliberately outside this cleanup API.
func (s *Store) CleanupBinBefore(before time.Time, apply bool) ([]string, error) {
	entries, err := os.ReadDir(s.BinRoot())
	if err != nil {
		return nil, err
	}
	var candidates []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasPrefix(entry.Name(), "task-") {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(before) {
			continue
		}
		path := filepath.Join(s.BinRoot(), entry.Name())
		if err := ensureWithin(s.BinRoot(), path); err != nil {
			return nil, err
		}
		candidates = append(candidates, path)
	}
	sort.Strings(candidates)
	if apply {
		for _, path := range candidates {
			if err := os.RemoveAll(path); err != nil {
				return candidates, fmt.Errorf("remove bin task %s: %w", filepath.Base(path), err)
			}
		}
	}
	return candidates, nil
}

func (s *Store) collectArtifacts(session *agent.TaskSession) ([]agent.ArtifactReference, error) {
	var artifacts []agent.ArtifactReference
	err := filepath.WalkDir(session.OutputDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == session.OutputDir || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact symlink is not allowed: %s", entry.Name())
		}
		if entry.Name() == "manifest.json" {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		bytes, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		artifacts = append(artifacts, agent.ArtifactReference{Path: s.relativePath(path), SHA256: hex.EncodeToString(hash.Sum(nil)), Bytes: bytes})
		return nil
	})
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	return artifacts, err
}

func manifestEvidence(result *agent.AgentResult, artifacts []agent.ArtifactReference) []ManifestEvidence {
	var evidence []ManifestEvidence
	for _, step := range result.Trace {
		if step.ToolCall != nil && step.ToolResult != nil && step.ToolResult.Status == agent.ToolSucceeded {
			evidence = append(evidence, ManifestEvidence{Kind: "tool_result", Tool: step.ToolCall.Tool, Ref: step.ToolResult.CallID})
		}
	}
	for _, artifact := range artifacts {
		evidence = append(evidence, ManifestEvidence{Kind: "artifact_sha256", Path: artifact.Path, SHA256: artifact.SHA256})
	}
	return evidence
}

func (s *Store) validateSession(session *agent.TaskSession) error {
	if session == nil || strings.TrimSpace(session.ID) == "" {
		return fmt.Errorf("task session is required")
	}
	for root, path := range map[string]string{s.BinRoot(): session.BinDir, s.OutputRoot(): session.OutputDir} {
		if err := ensureWithin(root, path); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) writeCheckpoint(session *agent.TaskSession, value checkpoint) error {
	if err := writeJSONAtomic(filepath.Join(session.BinDir, "checkpoint.json"), value); err != nil {
		return fmt.Errorf("write task checkpoint: %w", err)
	}
	return nil
}

func (s *Store) relativePath(path string) string {
	rel, err := filepath.Rel(filepath.Dir(s.root), path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func taskID(now time.Time, task, planID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(task) + "\x00" + planID + "\x00" + now.Format(time.RFC3339Nano)))
	return "task-" + now.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(sum[:4])
}

func ensureRealDirectory(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", path, err)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("Agent Folder path is not a real directory: %s", path)
	}
	return nil
}

func ensureWithin(root, path string) error {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("task path escapes Agent Folder tray: %s", path)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(filepath.Dir(path), ".boi-agent-folder-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	renameErr := os.Rename(tempPath, path)
	if renameErr == nil {
		return nil
	}
	// Windows does not replace an existing destination with os.Rename. The
	// checkpoint contains no user deliverable, so replace that exact file only.
	if runtime.GOOS != "windows" {
		return renameErr
	}
	if _, statErr := os.Lstat(path); statErr != nil {
		return renameErr
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, path)
}
