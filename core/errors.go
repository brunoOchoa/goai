package core

import (
	"errors"
	"fmt"
)

// Sentinel errors — use [errors.Is] to match these across provider implementations.
var (
	// ErrRateLimit is returned when the provider throttles the request (HTTP 429).
	ErrRateLimit = errors.New("goai: rate limit exceeded")

	// ErrAuthFailed is returned for invalid or missing API keys (HTTP 401/403).
	ErrAuthFailed = errors.New("goai: authentication failed")

	// ErrModelNotFound is returned when the requested model does not exist (HTTP 404).
	ErrModelNotFound = errors.New("goai: model not found")

	// ErrContextLength is returned when the input exceeds the model's context window.
	ErrContextLength = errors.New("goai: context length exceeded")

	// ErrToolNotFound is returned when a tool call references an unregistered tool.
	ErrToolNotFound = errors.New("goai: tool not found")

	// ErrToolExec is returned when a tool's Execute method returns an error.
	ErrToolExec = errors.New("goai: tool execution failed")

	// ErrProviderError is the catch-all for provider API errors not covered above.
	ErrProviderError = errors.New("goai: provider error")

	// ErrStreamClosed is returned when the streaming connection closes unexpectedly.
	ErrStreamClosed = errors.New("goai: stream closed unexpectedly")

	// ErrInvalidResponse is returned when the provider returns a malformed response.
	ErrInvalidResponse = errors.New("goai: invalid response from provider")

	// ErrMCPTransport is returned for MCP protocol communication errors.
	ErrMCPTransport = errors.New("goai: MCP transport error")

	// ErrSchemaGen is returned when JSON Schema generation fails.
	ErrSchemaGen = errors.New("goai: schema generation failed")

	// ErrMaxIterations is returned when an agent exceeds its iteration limit.
	ErrMaxIterations = errors.New("goai: max iterations reached")
)

// APIError wraps an HTTP error from a provider with contextual metadata.
// Unwrap returns a sentinel error suitable for use with [errors.Is].
type APIError struct {
	// StatusCode is the HTTP status code (e.g., 429, 401, 404).
	StatusCode int

	// Provider identifies the source (e.g., "anthropic", "openai", "gemini").
	Provider string

	// Message is the error description returned by the provider API.
	Message string

	// RequestID is the provider's trace ID, useful for support tickets.
	// May be empty if the provider did not return one.
	RequestID string

	// Err wraps a sentinel error (e.g., ErrRateLimit) for use with errors.Is.
	Err error
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("goai/%s HTTP %d: %s (request_id=%s)",
			e.Provider, e.StatusCode, e.Message, e.RequestID)
	}
	return fmt.Sprintf("goai/%s HTTP %d: %s", e.Provider, e.StatusCode, e.Message)
}

// Unwrap enables [errors.Is] and [errors.As] to inspect the sentinel.
func (e *APIError) Unwrap() error { return e.Err }
