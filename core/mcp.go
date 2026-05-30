package core

import (
	"context"
	"encoding/json"
)

// MCPTool é uma tool descoberta de um servidor MCP.
// Também satisfaz AnyTool para poder ser adicionada a qualquer ToolRegistry.
type MCPTool struct {
	ToolName        string
	ToolDescription string
	InputSchema     json.RawMessage
	Client          MCPClient // referência para Execute
}

func (t *MCPTool) Name() string              { return t.ToolName }
func (t *MCPTool) Description() string       { return t.ToolDescription }
func (t *MCPTool) Schema() json.RawMessage   { return t.InputSchema }
func (t *MCPTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.Client.CallTool(ctx, t.ToolName, args)
}

// MCPResource é um recurso legível exposto por um servidor MCP.
type MCPResource struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
}

// MCPClient conecta a um servidor MCP e expõe suas capacidades.
type MCPClient interface {
	Connect(ctx context.Context) error
	Close() error
	ListTools(ctx context.Context) ([]MCPTool, error)
	ListResources(ctx context.Context) ([]MCPResource, error)
	CallTool(ctx context.Context, name string, args json.RawMessage) (string, error)
	ReadResource(ctx context.Context, uri string) ([]byte, string, error)
}
