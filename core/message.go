package core

import "encoding/json"

// Role identifies the participant in a conversation turn.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	// RoleTool carries tool execution results back to the model.
	RoleTool Role = "tool"
)

// ContentKind discriminates the type of a ContentPart.
type ContentKind uint8

const (
	ContentText  ContentKind = iota // plain text
	ContentImage                    // image (base64 encoded or URL)
	ContentAudio                    // audio (base64 encoded)
	ContentFile                     // file reference (URL or inline bytes)
)

// ContentPart is a single element within a message.
// A message can hold multiple parts for multimodal input (e.g., text + image).
// Either Data or URL should be set for binary content, not both.
type ContentPart struct {
	Kind     ContentKind
	Text     string // set when Kind == ContentText
	MIMEType string // e.g., "image/png", "audio/wav"
	Data     []byte // inline binary content (nil when using URL)
	URL      string // remote URL (empty when using inline Data)
}

// ToolCall represents a function invocation requested by the model.
type ToolCall struct {
	// ID is an opaque identifier used to correlate this call with its ToolResult.
	ID string

	// Name is the function name as registered in the ToolRegistry.
	Name string

	// Arguments is the raw JSON payload the model produced for this call.
	// Validate and unmarshal before passing to user code.
	Arguments json.RawMessage
}

// ToolResult carries the output of an executed ToolCall back to the model.
type ToolResult struct {
	// CallID matches the ToolCall.ID that triggered this result.
	CallID string

	// Name is the function name (mirrors ToolCall.Name).
	Name string

	// Content is the serialized result — JSON or plain text.
	Content string

	// IsError signals that the tool failed; the model will receive the error description.
	IsError bool
}

// Message is the canonical unit of conversation.
//
// Exactly one of Parts, ToolCalls, or ToolResults should be populated per message:
//   - Parts: natural-language content (text, images, audio).
//   - ToolCalls: the model requests tool execution (Role == RoleAssistant).
//   - ToolResults: tool outputs sent back to the model (Role == RoleTool).
type Message struct {
	Role        Role
	Parts       []ContentPart
	ToolCalls   []ToolCall
	ToolResults []ToolResult
	// Metadata carries provider-specific pass-through data (e.g., cache hints).
	Metadata map[string]any
}

// Text returns the concatenation of all text parts in the message.
func (m Message) Text() string {
	var out string
	for _, p := range m.Parts {
		if p.Kind == ContentText {
			out += p.Text
		}
	}
	return out
}

// TextMessage creates a message with a single text part.
func TextMessage(role Role, text string) Message {
	return Message{Role: role, Parts: []ContentPart{{Kind: ContentText, Text: text}}}
}

// UserMessage creates a user text message.
func UserMessage(text string) Message { return TextMessage(RoleUser, text) }

// AssistantMessage creates an assistant text message.
func AssistantMessage(text string) Message { return TextMessage(RoleAssistant, text) }

// SystemMessage creates a system prompt message.
func SystemMessage(text string) Message { return TextMessage(RoleSystem, text) }
