package app

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/block"
	coreblock "github.com/boi-family/boi-cli/internal/core"
)

// TestMCPServerProcess is a fake MCP server: the tests start this test binary
// with "mcp-server" in its arguments.
func TestMCPServerProcess(t *testing.T) {
	if !slices.Contains(os.Args, "mcp-server") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var request struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(scanner.Bytes(), &request) != nil {
			os.Exit(2)
		}
		switch request.Method {
		case "initialize":
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "fake", "version": "1"}}})
		case "tools/list":
			if slices.Contains(os.Args, "hang") {
				time.Sleep(30 * time.Second)
			}
			echo := map[string]any{"name": "echo", "description": "Echo text back.", "inputSchema": map[string]any{
				"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}, "required": []string{"text"},
			}}
			env := map[string]any{"name": "env", "description": "Report whether Provider keys are visible.", "inputSchema": map[string]any{"type": "object"}}
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"tools": []any{echo, env}}})
			return
		case "tools/call":
			text := "env-clean"
			for _, item := range os.Environ() {
				if strings.HasPrefix(item, "PSC_") {
					text = "env-leaked"
				}
			}
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}}})
			return
		}
	}
}

func fakeMCP(extra ...string) *block.MCPServer {
	return &block.MCPServer{Command: os.Args[0], Args: append([]string{"-test.run=TestMCPServerProcess", "--", "mcp-server"}, extra...)}
}

func TestMCPBlockPlugsInServerTools(t *testing.T) {
	t.Setenv("PSC_1_API_KEY", "must-not-leak")
	environment := environmentOf(
		block.EnvironmentBlock{ID: "workspace-tools", Required: true},
		block.EnvironmentBlock{ID: "fake", MCP: fakeMCP()},
	)
	composition, err := Compose(environment, BuiltinPacks(), testSandbox(t), "1.9.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(composition.ToolNames()[5:], ","); got != "mcp.fake.echo,mcp.fake.env" {
		t.Fatalf("MCP Tools = %s", got)
	}
	var echo, env = composition.Tools[5], composition.Tools[6]
	spec := echo.Spec()
	if spec.Approval != "confirm" || spec.Risk != "external" || !strings.Contains(spec.Description, "Echo text back") {
		t.Fatalf("MCP Tool must be external, approval-gated and keep its description: %+v", spec)
	}
	if err := spec.ValidateArguments(map[string]any{}); err == nil || !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("the server's schema must be enforced: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := env.Execute(ctx, map[string]any{})
	if err != nil || !strings.Contains(output.Output, "env-clean") {
		t.Fatalf("an MCP server must not see Provider keys: %v %s", err, output.Output)
	}
}

func TestMCPBlockFailuresAreReportedNotFatal(t *testing.T) {
	original := MCPDiscoveryTimeout
	MCPDiscoveryTimeout = 500 * time.Millisecond
	t.Cleanup(func() { MCPDiscoveryTimeout = original })

	cases := map[string]struct {
		server *block.MCPServer
		id     string
		reason string
	}{
		"command missing": {&block.MCPServer{Command: filepath.Join(t.TempDir(), "no-such-server")}, "dead", "did not list its tools"},
		"server hangs":    {fakeMCP("hang"), "slow", "did not list its tools"},
		"unknown tool":    {&block.MCPServer{Command: os.Args[0], Args: fakeMCP().Args, Tools: []string{"delete_everything"}}, "picky", `has no tool "delete_everything"`},
		"built-in id":     {fakeMCP(), "process-tools", "already used by a built-in pack"},
	}
	for name, tc := range cases {
		optional := environmentOf(block.EnvironmentBlock{ID: "workspace-tools", Required: true}, block.EnvironmentBlock{ID: tc.id, MCP: tc.server})
		composition, err := Compose(optional, BuiltinPacks(), testSandbox(t), "1.9.0", coreblock.AgentEnvironment{})
		if err != nil {
			t.Fatalf("%s: an optional MCP Block must never stop start-up: %v", name, err)
		}
		last := composition.Entries[len(composition.Entries)-1]
		if last.Status != PackSkipped || !strings.Contains(last.Reason, tc.reason) || len(composition.Tools) != 5 {
			t.Errorf("%s: entry %+v tools %v, want skipped with %q and the workspace Tools intact", name, last, composition.ToolNames(), tc.reason)
		}
		required := environmentOf(block.EnvironmentBlock{ID: tc.id, Required: true, MCP: tc.server})
		if _, err := Compose(required, BuiltinPacks(), testSandbox(t), "1.9.0", coreblock.AgentEnvironment{}); err == nil || !strings.Contains(err.Error(), tc.reason) {
			t.Errorf("%s: a required MCP Block must stop start-up with %q, got %v", name, tc.reason, err)
		}
	}
}

func TestMCPToolAllowlist(t *testing.T) {
	server := fakeMCP()
	server.Tools = []string{"echo"}
	environment := environmentOf(block.EnvironmentBlock{ID: "fake", Required: true, MCP: server})
	composition, err := Compose(environment, BuiltinPacks(), testSandbox(t), "1.9.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(composition.ToolNames(), ","); got != "mcp.fake.echo" {
		t.Fatalf("only allowlisted tools may be plugged in, got %s", got)
	}
}

func TestPrepareTaskActivatesMCPTools(t *testing.T) {
	root := t.TempDir()
	sandbox := testSandbox(t)
	boiDir := filepath.Join(root, ".boi")
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(environmentOf(block.EnvironmentBlock{ID: "workspace-tools", Required: true}, block.EnvironmentBlock{ID: "fake", MCP: fakeMCP()}))
	if err := os.WriteFile(filepath.Join(boiDir, EnvironmentFile), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := TrustEnvironment(boiDir); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{Version: "1.9.0", WorkspaceRoot: root, BoiDir: boiDir, Sandbox: sandbox}
	agent, err := runtime.BuildAgent(nil)
	if err != nil {
		t.Fatal(err)
	}
	agent.Environment.ToolCalling = true
	capabilities, err := agent.PrepareTask("echo something")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(capabilities.Tools.Active, "mcp.fake.echo") || !slices.Contains(capabilities.Tools.Active, "workspace.read") {
		t.Fatalf("MCP and workspace Tools must both be active: %v", capabilities.Tools.Active)
	}
}

func TestUntrustedWorkspaceNeverStartsMCPServers(t *testing.T) {
	boiDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "started")
	server := fakeMCP()
	server.Command, server.Args = "sh", []string{"-c", "touch " + marker}
	environment := environmentOf(block.EnvironmentBlock{ID: "workspace-tools", Required: true}, block.EnvironmentBlock{ID: "cloned", MCP: server})
	data, _ := json.Marshal(environment)
	path := filepath.Join(boiDir, EnvironmentFile)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadEnvironment(boiDir)
	if err != nil || loaded.Trusted {
		t.Fatalf("a file nobody trusted must load untrusted: %+v %v", loaded, err)
	}
	composition, err := Compose(loaded, BuiltinPacks(), testSandbox(t), "2.0.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	last := composition.Entries[len(composition.Entries)-1]
	if last.Status != PackSkipped || !strings.Contains(last.Reason, "boi env trust") {
		t.Fatalf("an untrusted MCP entry must be skipped with the way to trust it: %+v", last)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("an untrusted MCP command was executed")
	}

	if _, err := TrustEnvironment(boiDir); err != nil {
		t.Fatal(err)
	}
	if loaded, _ = LoadEnvironment(boiDir); !loaded.Trusted {
		t.Fatal("trusting the file must be remembered")
	}
	if err := os.WriteFile(path, append(data, ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	if loaded, _ = LoadEnvironment(boiDir); loaded.Trusted {
		t.Fatal("any edit must withdraw the trust")
	}
}
