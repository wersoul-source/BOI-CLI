package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/block/agentfolder"
	"github.com/boi-family/boi-cli/internal/workspace"
)

type runtimeContextKey struct{}

// Runtime contains process-wide paths and build information shared by all
// transports. Runtime services will be added here as the agent engine is
// consolidated in Phase 2.
type Runtime struct {
	Version         string
	WorkspaceRoot   string
	BoiDir          string
	ConfigPath      string
	EnvPath         string
	IdentityPath    string
	AgentFolderRoot string
	AgentFolder     *agentfolder.Store
	Sandbox         *workspace.Sandbox
}

// NewRuntime resolves workspace-scoped paths once at process startup.
func NewRuntime(version string) (*Runtime, error) {
	root, err := workspace.DetectRoot()
	if err != nil {
		return nil, err
	}

	boiDir := workspace.GetBoiDir(root)
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		return nil, fmt.Errorf("create workspace sandbox: %w", err)
	}
	agentFolderRoot := filepath.Join(root, "agent-folder")
	agentFolder, err := agentfolder.NewStore(agentFolderRoot)
	if err != nil {
		return nil, fmt.Errorf("create Agent Folder: %w", err)
	}
	return &Runtime{
		Version:         version,
		WorkspaceRoot:   root,
		BoiDir:          boiDir,
		ConfigPath:      filepath.Join(boiDir, "config.yaml"),
		EnvPath:         filepath.Join(root, ".env"),
		IdentityPath:    filepath.Join(boiDir, "agent.yaml"),
		AgentFolderRoot: agentFolderRoot,
		AgentFolder:     agentFolder,
		Sandbox:         sandbox,
	}, nil
}

// WithRuntime attaches the process runtime to a request context.
func WithRuntime(ctx context.Context, runtime *Runtime) context.Context {
	return context.WithValue(ctx, runtimeContextKey{}, runtime)
}

// RuntimeFromContext returns the process runtime attached by the composition
// root. Commands should use this instead of resolving workspace paths again.
func RuntimeFromContext(ctx context.Context) (*Runtime, bool) {
	runtime, ok := ctx.Value(runtimeContextKey{}).(*Runtime)
	return runtime, ok
}
