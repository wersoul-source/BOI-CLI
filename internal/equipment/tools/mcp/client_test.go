package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"testing"
	"time"
)

func TestClientPerformsLifecycleAndListsTools(t *testing.T) {
	client := NewClient()
	if err := client.Register(helperConfig()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := client.DiscoverToolsContext(ctx, "helper")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "search" {
		t.Fatalf("unexpected tools: %#v", tools)
	}
}

func TestClientPerformsLifecycleAndCallsTool(t *testing.T) {
	client := NewClient()
	if err := client.Register(helperConfig()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := client.CallTool(ctx, "helper", "search", map[string]any{"query": "boi"})
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"content":[{"text":"ok","type":"text"}]}` {
		t.Fatalf("unexpected result: %s", result)
	}
}

func TestClientCancellationStopsServer(t *testing.T) {
	client := NewClient()
	config := helperConfig()
	config.Args = append(config.Args, "mcp-hang")
	if err := client.Register(config); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := client.DiscoverToolsContext(ctx, "helper")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline, got %v", err)
	}
}

func helperConfig() ServerConfig {
	return ServerConfig{Name: "helper", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess", "--", "mcp-helper"}, Transport: "stdio"}
}

func TestMCPHelperProcess(t *testing.T) {
	if !slices.Contains(os.Args, "mcp-helper") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var request struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(2)
		}
		switch request.Method {
		case "initialize":
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"protocolVersion": protocolVersion, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "helper", "version": "1"}}})
		case "tools/list":
			if slices.Contains(os.Args, "mcp-hang") {
				time.Sleep(30 * time.Second)
			}
			tool := map[string]any{"name": "search", "description": "search", "inputSchema": map[string]any{"type": "object"}}
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"tools": []map[string]any{tool}}})
			return
		case "tools/call":
			content := map[string]any{"type": "text", "text": "ok"}
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"content": []map[string]any{content}}})
			return
		}
	}
}
