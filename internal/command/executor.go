package command

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/boi-family/boi-cli/internal/logger"
)

// Executor executes shell commands with sandboxing
type Executor struct {
	sandbox *Sandbox
	logger  *logger.Logger
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
	if err := e.sandbox.Allow(command); err != nil {
		return "", fmt.Errorf("sandbox blocked: %w", err)
	}

	cmd := e.buildCommand(command, "")
	return e.execute(cmd)
}

// RunWithDir executes a command in a specific directory
func (e *Executor) RunWithDir(command, dir string) (string, error) {
	if err := e.sandbox.Allow(command); err != nil {
		return "", fmt.Errorf("sandbox blocked: %w", err)
	}

	cmd := e.buildCommand(command, dir)
	return e.execute(cmd)
}

// buildCommand creates the appropriate exec.Cmd for the OS
func (e *Executor) buildCommand(command, dir string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = dir
	cmd.Env = os.Environ()
	return cmd
}

// execute runs the command and captures output
func (e *Executor) execute(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
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

	return stdout.String(), nil
}
