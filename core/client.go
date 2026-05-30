// Package core defines the provider-agnostic interfaces and types for the goai framework.
// It has zero external dependencies — only the Go standard library.
//
// All AI provider implementations (anthropic, openai, gemini, compat) implement the
// interfaces defined here, so application code can switch providers without changes.
package core

import (
	"context"
	"iter"
)

// Request is the input to any LLM call.
// All fields are optional except Messages (or System for single-turn calls).
type Request struct {
	// Model overrides the client's default model for this specific call.
	Model string

	// Messages is the conversation history, in chronological order.
	// Must contain at least one message.
	Messages []Message

	// Tools lists the functions the model may call during this request.
	Tools []AnyTool

	// MaxTokens caps the number of output tokens. Zero uses the client default.
	MaxTokens int

	// Temperature controls output randomness (0.0–2.0). Nil uses the model default.
	Temperature *float64

	// TopP enables nucleus sampling. Nil uses the model default.
	// Do not set both Temperature and TopP simultaneously.
	TopP *float64

	// Stop lists strings that cause the model to stop generating.
	Stop []string

	// System is a convenience system prompt, merged with any RoleSystem messages.
	System string

	// Metadata passes provider-specific options through the request.
	Metadata map[string]any
}

// Response is the complete output of a non-streaming LLM call.
type Response struct {
	// Message is the assistant's reply, including any tool calls.
	Message Message

	// Usage reports token consumption for this call.
	Usage Usage

	// Model is the model that produced this response (may differ from the request).
	Model string

	// StopReason indicates why the model stopped generating.
	// Common values: "end_turn", "tool_use", "max_tokens", "stop_sequence".
	// The exact values are provider-specific.
	StopReason string
}

// ChatClient performs a single-turn, non-streaming LLM call.
type ChatClient interface {
	// Chat sends a request and blocks until the full response is available.
	Chat(ctx context.Context, req Request) (Response, error)
}

// StreamClient performs a streaming LLM call.
type StreamClient interface {
	// Stream returns an iterator that yields Events incrementally.
	// The iterator emits events until EventDone or EventError.
	// Breaking out of the iterator early (e.g., via return) is safe — no goroutine leaks.
	Stream(ctx context.Context, req Request) iter.Seq2[Event, error]
}

// Client combines ChatClient and StreamClient.
// All provider implementations satisfy this interface.
type Client interface {
	ChatClient
	StreamClient

	// ModelID returns the default model identifier configured for this client.
	ModelID() string
}
