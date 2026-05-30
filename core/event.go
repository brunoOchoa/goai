package core

// EventKind classifies a streaming event emitted by a StreamClient.
type EventKind uint8

const (
	// EventText carries an incremental text delta from the model.
	// Both Delta (this chunk only) and Text (accumulated so far) are set.
	EventText EventKind = iota

	// EventToolCall carries a (partial or complete) tool call from the model.
	EventToolCall

	// EventToolResult carries a tool result injected mid-stream.
	EventToolResult

	// EventUsage carries token accounting, typically as the last event before EventDone.
	EventUsage

	// EventDone signals that the stream completed successfully.
	// No further events will be emitted after this.
	EventDone

	// EventError signals a terminal error. The Err field will be non-nil.
	// No further events will be emitted after this.
	EventError
)

// Usage reports token consumption for a request.
type Usage struct {
	InputTokens  int
	OutputTokens int
	// CachedTokens counts tokens served from the provider's prompt cache.
	// Provider-specific: non-zero only for Anthropic (cache read tokens).
	CachedTokens int
}

// Event is a single chunk yielded by a streaming call.
//
// Consumers should switch on Kind to determine which fields are meaningful:
//
//	for ev, err := range client.Stream(ctx, req) {
//	    if err != nil { /* handle */ }
//	    switch ev.Kind {
//	    case core.EventText:
//	        fmt.Print(ev.Delta)
//	    case core.EventDone:
//	        return
//	    }
//	}
type Event struct {
	Kind EventKind

	// Delta is the text added in this specific chunk (EventText only).
	Delta string

	// Text is the full accumulated text up to and including this chunk (EventText only).
	Text string

	// ToolCall carries the function call data (EventToolCall only).
	ToolCall *ToolCall

	// Usage carries token accounting (EventUsage only).
	Usage *Usage

	// Err is the terminal error (EventError only). Always non-nil when Kind == EventError.
	Err error
}
