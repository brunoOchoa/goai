// Package memory provides concrete implementations of the core.Memory interface.
//
// Available implementations:
//   - [WindowMemory] — keeps the last N messages in RAM (no persistence)
//   - [FileMemory]   — persists the full history to a JSON file on disk
package memory

import (
	"context"
	"sync"

	"github.com/brunoochoa/goai/core"
)

const defaultWindowSize = 20

// WindowMemory keeps the last N messages in RAM.
// Older messages are discarded when the limit is reached.
// It is safe for concurrent use by multiple goroutines.
//
// Example:
//
//	mem := memory.NewWindow(10) // keep last 10 messages
//	session := chat.New(client, chat.WithMemory(mem))
type WindowMemory struct {
	mu       sync.RWMutex
	messages []core.Message
	maxSize  int // número máximo de mensagens a manter
}

// NewWindow creates a WindowMemory that retains the last maxSize messages.
// If maxSize <= 0, defaults to 20.
func NewWindow(maxSize int) *WindowMemory {
	if maxSize <= 0 {
		maxSize = defaultWindowSize
	}
	return &WindowMemory{
		maxSize:  maxSize,
		messages: make([]core.Message, 0, maxSize),
	}
}

// Add appends a message to the window.
// If the window is full, the oldest message is discarded.
func (m *WindowMemory) Add(_ context.Context, msg core.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, msg)

	// Descarta as mensagens mais antigas quando ultrapassa o limite.
	if len(m.messages) > m.maxSize {
		// Move o slice para o início sem realocar — mais eficiente.
		overflow := len(m.messages) - m.maxSize
		copy(m.messages, m.messages[overflow:])
		m.messages = m.messages[:m.maxSize]
	}

	return nil
}

// Load returns a snapshot of the current window.
// The returned slice is a copy — mutations do not affect the internal state.
func (m *WindowMemory) Load(_ context.Context) ([]core.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]core.Message, len(m.messages))
	copy(out, m.messages)
	return out, nil
}

// Clear removes all messages from the window.
func (m *WindowMemory) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = m.messages[:0]
	return nil
}

// Len returns the current number of messages in the window.
func (m *WindowMemory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}
