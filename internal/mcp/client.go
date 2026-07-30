package mcp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
)

// ServerConfig defines an MCP server connection
type ServerConfig struct {
	Name      string   `yaml:"name" json:"name"`
	Command   string   `yaml:"command" json:"command"`
	Args      []string `yaml:"args" json:"args"`
	Transport string   `yaml:"transport" json:"transport"`
	URL       string   `yaml:"url" json:"url,omitempty"`
}

// Client manages MCP server connections
type Client struct {
	mu      sync.RWMutex
	servers map[string]*ServerConfig
}

// NewClient creates a new MCP client
func NewClient() *Client {
	return &Client{
		servers: make(map[string]*ServerConfig),
	}
}

// Register adds an MCP server config
func (c *Client) Register(s ServerConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.servers[s.Name] = &s
}

// List returns all registered MCP servers
func (c *Client) List() []ServerConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]ServerConfig, 0, len(c.servers))
	for _, s := range c.servers {
		result = append(result, *s)
	}
	return result
}

// Get retrieves a server by name
func (c *Client) Get(name string) (*ServerConfig, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.servers[name]
	if !ok {
		return nil, fmt.Errorf("MCP server not found: %s", name)
	}
	return s, nil
}

// DiscoverTools lists tools from an MCP server
func (c *Client) DiscoverTools(serverName string) ([]ToolDef, error) {
	s, err := c.Get(serverName)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(s.Command, s.Args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	data, _ := json.Marshal(req)

	go func() {
		stdin.Write(data)
		stdin.Close()
	}()

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("MCP tools/list failed: %w", err)
	}

	var resp struct {
		Result struct {
			Tools []ToolDef `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("parse MCP response: %w", err)
	}

	return resp.Result.Tools, nil
}

// ToolDef represents a tool from an MCP server
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
