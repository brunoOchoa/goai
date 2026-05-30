// Package chat provides a high-level conversational session on top of any core.Client.
// It manages message history automatically and supports both blocking and streaming calls.
package chat

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"sync"

	"github.com/brunoochoa/goai/core"
)

// Session manages a multi-turn conversation with an LLM.
// It maintains message history either in memory or via an external core.Memory backend.
//
// Session is safe for concurrent use by multiple goroutines.
type Session struct {
	mu      sync.RWMutex
	client  core.ChatClient
	stream  core.StreamClient
	memory  core.Memory
	system  string
	model   string
	maxToks int
	tools   []core.AnyTool
	history []core.Message // used only when memory == nil
}

// Option configures a Session.
type Option func(*Session)

// WithMemory sets an external memory backend for history storage.
// When set, the in-memory slice is not used.
func WithMemory(m core.Memory) Option { return func(s *Session) { s.memory = m } }

// WithSystem sets the system prompt for all calls in this session.
func WithSystem(p string) Option { return func(s *Session) { s.system = p } }

// WithModel overrides the default model for this session.
func WithModel(m string) Option { return func(s *Session) { s.model = m } }

// WithMaxTokens sets the maximum output tokens per request.
func WithMaxTokens(n int) Option { return func(s *Session) { s.maxToks = n } }

// WithTools registers tools available to the model in this session.
func WithTools(tools ...core.AnyTool) Option { return func(s *Session) { s.tools = tools } }

// New creates a Session backed by the given client.
// The client must implement core.Client (which includes both ChatClient and StreamClient).
func New(client core.Client, opts ...Option) *Session {
	s := &Session{client: client, stream: client}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Send sends a user message and returns the assistant's response.
// History is loaded before the call and saved after.
func (s *Session) Send(ctx context.Context, text string) (core.Message, error) {
	if text == "" {
		return core.Message{}, fmt.Errorf("chat: mensagem vazia")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	msgs, err := s.buildMessages(ctx, text)
	if err != nil {
		return core.Message{}, err
	}

	req := core.Request{
		Model:     s.model,
		Messages:  msgs,
		Tools:     s.tools,
		MaxTokens: s.maxToks,
		System:    s.system,
	}

	resp, err := s.client.Chat(ctx, req)
	if err != nil {
		return core.Message{}, err
	}

	if err := s.saveHistory(ctx, core.UserMessage(text), resp.Message); err != nil {
		// Não falha a chamada por erro de persistência, mas loga.
		slog.WarnContext(ctx, "chat: falha ao salvar histórico", "err", err)
	}

	return resp.Message, nil
}

// Stream sends a user message and returns an iterator of streaming Events.
// History is saved after the stream completes (on EventDone).
func (s *Session) Stream(ctx context.Context, text string) iter.Seq2[core.Event, error] {
	return func(yield func(core.Event, error) bool) {
		if text == "" {
			err := fmt.Errorf("chat: mensagem vazia")
			yield(core.Event{Kind: core.EventError, Err: err}, err)
			return
		}

		s.mu.Lock()
		msgs, err := s.buildMessages(ctx, text)
		s.mu.Unlock()

		if err != nil {
			yield(core.Event{Kind: core.EventError, Err: err}, err)
			return
		}

		req := core.Request{
			Model:     s.model,
			Messages:  msgs,
			Tools:     s.tools,
			MaxTokens: s.maxToks,
			System:    s.system,
		}

		var fullText string
		for ev, err := range s.stream.Stream(ctx, req) {
			if err != nil {
				yield(core.Event{Kind: core.EventError, Err: err}, err)
				return
			}
			if ev.Kind == core.EventText {
				fullText = ev.Text
			}
			if !yield(ev, nil) {
				return
			}
		}

		// Salva histórico após stream completo.
		if fullText != "" {
			s.mu.Lock()
			if herr := s.saveHistory(ctx, core.UserMessage(text), core.AssistantMessage(fullText)); herr != nil {
				slog.WarnContext(ctx, "chat: falha ao salvar histórico do stream", "err", herr)
			}
			s.mu.Unlock()
		}
	}
}

// History returns a snapshot of the current conversation history.
func (s *Session) History(ctx context.Context) ([]core.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.memory != nil {
		return s.memory.Load(ctx)
	}
	out := make([]core.Message, len(s.history))
	copy(out, s.history)
	return out, nil
}

// Reset clears the conversation history.
func (s *Session) Reset(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.memory != nil {
		return s.memory.Clear(ctx)
	}
	s.history = nil
	return nil
}

// buildMessages monta a lista de mensagens para o request.
// Deve ser chamado com o lock adquirido.
func (s *Session) buildMessages(ctx context.Context, userText string) ([]core.Message, error) {
	var history []core.Message
	var err error

	if s.memory != nil {
		history, err = s.memory.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("chat: carregar histórico: %w", err)
		}
	} else {
		history = s.history
	}

	msgs := make([]core.Message, 0, len(history)+1)
	msgs = append(msgs, history...)
	msgs = append(msgs, core.UserMessage(userText))
	return msgs, nil
}

// saveHistory persiste uma troca usuário/assistente.
// Deve ser chamado com o lock adquirido (ou dentro da seção crítica de Stream).
func (s *Session) saveHistory(ctx context.Context, user, assistant core.Message) error {
	if s.memory != nil {
		if err := s.memory.Add(ctx, user); err != nil {
			return fmt.Errorf("salvar mensagem do usuário: %w", err)
		}
		if err := s.memory.Add(ctx, assistant); err != nil {
			return fmt.Errorf("salvar resposta do assistente: %w", err)
		}
		return nil
	}
	s.history = append(s.history, user, assistant)
	return nil
}
