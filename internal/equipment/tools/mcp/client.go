package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
)

const defaultMaxResponseBytes = 1024 * 1024
const protocolVersion = "2025-06-18"

type ServerConfig struct {
	Name      string   `yaml:"name" json:"name"`
	Command   string   `yaml:"command" json:"command"`
	Args      []string `yaml:"args" json:"args"`
	Transport string   `yaml:"transport" json:"transport"`
	URL       string   `yaml:"url" json:"url,omitempty"`
}

type Client struct {
	mu               sync.RWMutex
	servers          map[string]*ServerConfig
	maxResponseBytes int64
}

func NewClient() *Client {
	return &Client{servers: make(map[string]*ServerConfig), maxResponseBytes: defaultMaxResponseBytes}
}
func (c *Client) Register(server ServerConfig) error {
	if strings.TrimSpace(server.Name) == "" || strings.TrimSpace(server.Command) == "" {
		return fmt.Errorf("MCP server name and command are required")
	}
	if server.Transport != "" && server.Transport != "stdio" {
		return fmt.Errorf("unsupported MCP transport %q", server.Transport)
	}
	copy := server
	c.mu.Lock()
	c.servers[server.Name] = &copy
	c.mu.Unlock()
	return nil
}
func (c *Client) List() []ServerConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]ServerConfig, 0, len(c.servers))
	for _, server := range c.servers {
		result = append(result, *server)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
func (c *Client) Get(name string) (*ServerConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	server, ok := c.servers[name]
	if !ok {
		return nil, fmt.Errorf("MCP server not found: %s", name)
	}
	copy := *server
	return &copy, nil
}

func (c *Client) DiscoverTools(server string) ([]ToolDef, error) {
	return c.DiscoverToolsContext(context.Background(), server)
}
func (c *Client) DiscoverToolsContext(ctx context.Context, server string) ([]ToolDef, error) {
	var response struct {
		Result struct {
			Tools []ToolDef `json:"tools"`
		} `json:"result"`
	}
	if err := c.request(ctx, server, "tools/list", nil, &response); err != nil {
		return nil, err
	}
	return response.Result.Tools, nil
}
func (c *Client) CallTool(ctx context.Context, server, tool string, arguments map[string]any) (string, error) {
	var response struct {
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
	}
	if err := c.request(ctx, server, "tools/call", map[string]any{"name": tool, "arguments": arguments}, &response); err != nil {
		return "", err
	}
	if response.Error != nil {
		return "", fmt.Errorf("MCP error %d: %s", response.Error.Code, response.Error.Message)
	}
	return string(response.Result), nil
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) request(ctx context.Context, serverName, method string, params any, target any) error {
	server, err := c.Get(serverName)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, server.Command, server.Args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open MCP stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open MCP stdout: %w", err)
	}
	stderr := &limitedBuffer{limit: c.maxResponseBytes}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start MCP server: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()
	encoder := json.NewEncoder(stdin)
	limited := &io.LimitedReader{R: stdout, N: c.maxResponseBytes + 1}
	decoder := json.NewDecoder(limited)
	initialize := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": protocolVersion, "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "boi-cli", "version": "0.3.0"}}}
	if err := encoder.Encode(initialize); err != nil {
		return fmt.Errorf("send MCP initialize: %w", err)
	}
	initResponse, err := readResponse(decoder, 1)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("MCP initialize: %w", err)
	}
	var initialized struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(initResponse.Result, &initialized); err != nil {
		return fmt.Errorf("parse MCP initialize result: %w", err)
	}
	if initialized.ProtocolVersion != protocolVersion {
		return fmt.Errorf("unsupported MCP protocol version %q", initialized.ProtocolVersion)
	}
	if err := encoder.Encode(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"}); err != nil {
		return fmt.Errorf("send MCP initialized notification: %w", err)
	}
	request := map[string]any{"jsonrpc": "2.0", "id": 2, "method": method}
	if params != nil {
		request["params"] = params
	}
	if err := encoder.Encode(request); err != nil {
		return fmt.Errorf("send MCP %s: %w", method, err)
	}
	response, err := readResponse(decoder, 2)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("MCP %s: %w", method, err)
	}
	if limited.N <= 0 || stderr.exceeded {
		return fmt.Errorf("MCP %s response exceeds %d bytes", method, c.maxResponseBytes)
	}
	encoded, _ := json.Marshal(response)
	if err := json.Unmarshal(encoded, target); err != nil {
		return fmt.Errorf("parse MCP %s response: %w", method, err)
	}
	return nil
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

func readResponse(decoder *json.Decoder, id int) (rpcResponse, error) {
	for messages := 0; messages < 128; messages++ {
		var response rpcResponse
		if err := decoder.Decode(&response); err != nil {
			return rpcResponse{}, err
		}
		if response.ID != id {
			continue
		}
		if response.Error != nil {
			return rpcResponse{}, fmt.Errorf("error %d: %s", response.Error.Code, response.Error.Message)
		}
		return response, nil
	}
	return rpcResponse{}, fmt.Errorf("too many MCP messages before response %d", id)
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int64
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - int64(b.Len())
	if remaining <= 0 {
		b.exceeded = true
		return len(p), nil
	}
	write := p
	if int64(len(write)) > remaining {
		write = write[:remaining]
		b.exceeded = true
	}
	_, _ = b.Buffer.Write(write)
	return len(p), nil
}

var _ io.Writer = (*limitedBuffer)(nil)

type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema struct {
		Type       string `json:"type"`
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
	} `json:"inputSchema"`
}
