package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	logger "github.com/boi-family/boi-cli/internal/platform/logging"
	"github.com/boi-family/boi-cli/internal/workspace"
)

const maxProcessOutputBytes = 1024 * 1024

type limitedOutput struct {
	data      []byte
	truncated bool
}

func (o *limitedOutput) Write(p []byte) (int, error) {
	remaining := maxProcessOutputBytes - len(o.data)
	if remaining > 0 {
		take := len(p)
		if take > remaining {
			take = remaining
		}
		o.data = append(o.data, p[:take]...)
	}
	if len(p) > remaining {
		o.truncated = true
	}
	return len(p), nil
}
func (o *limitedOutput) String() string { return string(o.data) }
func (o *limitedOutput) Len() int       { return len(o.data) }

// Executor executes shell commands with sandboxing
type Executor struct {
	sandbox          *Sandbox
	logger           *logger.Logger
	workspaceSandbox *workspace.Sandbox
}

// ExecutorOption configures an Executor
type ExecutorOption func(*Executor)

// WithVerbose enables verbose logging
func WithVerbose(v bool) ExecutorOption {
	return func(e *Executor) {
		if v {
			e.logger = logger.NewWithLevel("debug")
		}
	}
}

// WithWorkspace restricts working directories to the supplied workspace path
// boundary. It does not turn the spawned shell into an OS-level sandbox.
func WithWorkspace(sandbox *workspace.Sandbox) ExecutorOption {
	return func(e *Executor) {
		e.workspaceSandbox = sandbox
	}
}

// NewExecutor creates a new Executor with default options
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		sandbox: NewSandbox(),
		logger:  logger.NewSilent(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Run executes a command and returns its stdout
func (e *Executor) Run(command string) (string, error) {
	return e.RunContext(context.Background(), command)
}

func (e *Executor) RunContext(ctx context.Context, command string) (string, error) {
	if err := e.sandbox.Allow(command); err != nil {
		return "", fmt.Errorf("sandbox blocked: %w", err)
	}

	dir := ""
	if e.workspaceSandbox != nil {
		dir = e.workspaceSandbox.Root()
	}
	cmd := e.buildCommandContext(ctx, command, dir)
	output, err := e.execute(cmd)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return output, err
}

// RunWithDir executes a command in a specific directory
func (e *Executor) RunWithDir(command, dir string) (string, error) {
	return e.RunWithDirContext(context.Background(), command, dir)
}

func (e *Executor) RunWithDirContext(ctx context.Context, command, dir string) (string, error) {
	if err := e.sandbox.Allow(command); err != nil {
		return "", fmt.Errorf("sandbox blocked: %w", err)
	}

	if e.workspaceSandbox != nil {
		resolved, err := e.workspaceSandbox.ResolveExisting(dir)
		if err != nil {
			return "", fmt.Errorf("workspace sandbox blocked working directory: %w", err)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return "", fmt.Errorf("stat working directory: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("working directory is not a directory: %s", dir)
		}
		dir = resolved
	}

	cmd := e.buildCommandContext(ctx, command, dir)
	output, err := e.execute(cmd)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return output, err
}

// buildCommand creates the appropriate exec.Cmd for the OS
func (e *Executor) buildCommand(command, dir string) *exec.Cmd {
	return e.buildCommandContext(context.Background(), command, dir)
}

func (e *Executor) buildCommandContext(ctx context.Context, command, dir string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	cmd.Dir = dir
	cmd.Env = os.Environ()
	return cmd
}

// execute runs the command and captures output
func (e *Executor) execute(cmd *exec.Cmd) (string, error) {
	var stdout, stderr limitedOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	e.logger.Debug("executing",
		"command", cmd.String(),
		"dir", cmd.Dir,
		"os", runtime.GOOS,
	)

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w\n%s", err, stderr.String())
		}
		return "", err
	}
	if stdout.truncated || stderr.truncated {
		return "", fmt.Errorf("process output exceeded %d bytes", maxProcessOutputBytes)
	}

	return stdout.String(), nil
}
