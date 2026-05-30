package core

import (
	"context"
	"encoding/json"
)

// AnyTool is the runtime interface stored in registries and passed to providers.
// Type-safe generic implementations live in the tools package.
//
// # Security
//
// Tool implementations receive raw JSON from the model and must validate inputs
// before executing side effects. Never trust model-generated arguments blindly.
type AnyTool interface {
	// Name returns the identifier the model uses to call this tool.
	// Must be unique within a ToolRegistry. Use snake_case (e.g., "web_search").
	Name() string

	// Description explains what the tool does. Shown to the model.
	// Be precise — the model uses this to decide when and how to call the tool.
	Description() string

	// Schema returns the JSON Schema (draft 2020-12) for the input arguments.
	// The returned bytes must be valid JSON representing an object schema.
	Schema() json.RawMessage

	// Execute runs the tool with the raw JSON arguments produced by the model.
	// Returns the serialized result as a string (JSON or plain text).
	// Return a non-nil error to signal failure; the model will receive the error text.
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// ToolRegistry is a named collection of tools available to an agent or chat session.
type ToolRegistry interface {
	// Register adds a tool. Returns an error if a tool with the same name already exists.
	Register(tool AnyTool) error

	// Get retrieves a tool by name. Returns false if not found.
	Get(name string) (AnyTool, bool)

	// All returns all registered tools in an unspecified order.
	All() []AnyTool
}
