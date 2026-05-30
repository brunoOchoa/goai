package chat_test

import (
	"context"
	"iter"
	"sync"
	"testing"

	"github.com/brunoochoa/goai/chat"
	"github.com/brunoochoa/goai/core"
)

// mockClient é um core.Client que retorna respostas fixas para testes.
type mockClient struct {
	response string
	err      error
}

func (m *mockClient) ModelID() string { return "mock-model" }

func (m *mockClient) Chat(_ context.Context, _ core.Request) (core.Response, error) {
	if m.err != nil {
		return core.Response{}, m.err
	}
	return core.Response{
		Message:    core.AssistantMessage(m.response),
		StopReason: "end_turn",
	}, nil
}

func (m *mockClient) Stream(_ context.Context, _ core.Request) iter.Seq2[core.Event, error] {
	return func(yield func(core.Event, error) bool) {
		if m.err != nil {
			yield(core.Event{Kind: core.EventError, Err: m.err}, m.err)
			return
		}
		yield(core.Event{Kind: core.EventText, Delta: m.response, Text: m.response}, nil)
		yield(core.Event{Kind: core.EventDone}, nil)
	}
}

func TestSession_Send(t *testing.T) {
	mc := &mockClient{response: "Olá! Como posso ajudar?"}
	s := chat.New(mc)

	msg, err := s.Send(t.Context(), "Oi")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if msg.Text() != "Olá! Como posso ajudar?" {
		t.Errorf("resposta inesperada: %q", msg.Text())
	}
}

func TestSession_EmptyMessage(t *testing.T) {
	mc := &mockClient{response: "x"}
	s := chat.New(mc)

	_, err := s.Send(t.Context(), "")
	if err == nil {
		t.Fatal("esperado erro para mensagem vazia")
	}
}

func TestSession_History(t *testing.T) {
	mc := &mockClient{response: "resp1"}
	s := chat.New(mc)

	if _, err := s.Send(t.Context(), "msg1"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	mc.response = "resp2"
	if _, err := s.Send(t.Context(), "msg2"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	history, err := s.History(t.Context())
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	// 2 trocas = 4 mensagens (user+assistant por troca)
	if len(history) != 4 {
		t.Errorf("esperado 4 mensagens no histórico, got %d", len(history))
	}
}

func TestSession_Reset(t *testing.T) {
	mc := &mockClient{response: "resp"}
	s := chat.New(mc)

	if _, err := s.Send(t.Context(), "msg"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if err := s.Reset(t.Context()); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	history, _ := s.History(t.Context())
	if len(history) != 0 {
		t.Errorf("esperado histórico vazio após Reset, got %d mensagens", len(history))
	}
}

func TestSession_WithSystem(t *testing.T) {
	mc := &mockClient{response: "ok"}
	s := chat.New(mc, chat.WithSystem("Você é um assistente."))

	if _, err := s.Send(t.Context(), "Olá"); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

// TestSession_ConcurrentSend verifica que a Session não tem race conditions.
// Execute com: go test -race ./chat/...
func TestSession_ConcurrentSend(t *testing.T) {
	mc := &mockClient{response: "ok"}
	s := chat.New(mc)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = s.Send(t.Context(), "mensagem concorrente")
		}()
	}

	wg.Wait()

	history, err := s.History(t.Context())
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	// Cada goroutine adiciona 2 mensagens (user + assistant)
	if len(history) != goroutines*2 {
		t.Errorf("esperado %d mensagens, got %d", goroutines*2, len(history))
	}
}

func TestSession_Stream(t *testing.T) {
	mc := &mockClient{response: "streaming!"}
	s := chat.New(mc)

	var collected string
	var gotDone bool
	for ev, err := range s.Stream(t.Context(), "Olá") {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		switch ev.Kind {
		case core.EventText:
			collected = ev.Text
		case core.EventDone:
			gotDone = true
		}
	}

	if collected != "streaming!" {
		t.Errorf("texto coletado: want %q, got %q", "streaming!", collected)
	}
	if !gotDone {
		t.Error("EventDone não recebido")
	}
}
