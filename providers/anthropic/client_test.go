package anthropic_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/brunoochoa/goai/core"
	"github.com/brunoochoa/goai/providers/anthropic"
)

// fixtureHandler retorna um http.Handler que serve um arquivo de testdata com o status dado.
func fixtureHandler(t *testing.T, file string, status int) http.Handler {
	t.Helper()
	data, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatalf("fixture %s: %v", file, err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(data)
	})
}

func TestNewClient_MissingKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := anthropic.NewClient()
	if err == nil {
		t.Fatal("esperado erro com API key vazia")
	}
}

func TestChat_Success(t *testing.T) {
	srv := httptest.NewServer(fixtureHandler(t, "success.json", http.StatusOK))
	defer srv.Close()

	client, err := anthropic.NewClient(
		anthropic.WithAPIKey("test-key"),
		anthropic.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.Chat(t.Context(), core.Request{
		Messages: []core.Message{core.UserMessage("Olá!")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if got := resp.Message.Text(); got != "Olá! Como posso ajudar?" {
		t.Errorf("texto inesperado: %q", got)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens: want 10, got %d", resp.Usage.InputTokens)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("StopReason: want end_turn, got %q", resp.StopReason)
	}
}

func TestChat_RateLimit(t *testing.T) {
	srv := httptest.NewServer(fixtureHandler(t, "rate_limit.json", http.StatusTooManyRequests))
	defer srv.Close()

	client, err := anthropic.NewClient(
		anthropic.WithAPIKey("test-key"),
		anthropic.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Chat(t.Context(), core.Request{
		Messages: []core.Message{core.UserMessage("Olá!")},
	})
	if !errors.Is(err, core.ErrRateLimit) {
		t.Errorf("esperado ErrRateLimit, got: %v", err)
	}
}

func TestChat_AuthError(t *testing.T) {
	srv := httptest.NewServer(fixtureHandler(t, "auth_error.json", http.StatusUnauthorized))
	defer srv.Close()

	client, err := anthropic.NewClient(
		anthropic.WithAPIKey("key-invalida"),
		anthropic.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Chat(t.Context(), core.Request{
		Messages: []core.Message{core.UserMessage("Olá!")},
	})
	if !errors.Is(err, core.ErrAuthFailed) {
		t.Errorf("esperado ErrAuthFailed, got: %v", err)
	}
}

func TestChat_EmptyMessages(t *testing.T) {
	srv := httptest.NewServer(fixtureHandler(t, "success.json", http.StatusOK))
	defer srv.Close()

	client, err := anthropic.NewClient(
		anthropic.WithAPIKey("test-key"),
		anthropic.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Request sem mensagens — o provider vai reclamar, mas o client não deve panic.
	_, err = client.Chat(context.Background(), core.Request{})
	// Não verificamos o erro específico — só que não paniquou.
	_ = err
}

func TestStream_Success(t *testing.T) {
	// Resposta SSE mínima compatível com o Anthropic SDK.
	sseBody := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_01\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-sonnet-4-5\",\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Olá!\"}}\n\n" +
		"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sseBody))
	}))
	defer srv.Close()

	client, err := anthropic.NewClient(
		anthropic.WithAPIKey("test-key"),
		anthropic.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var text string
	var gotDone bool
	for ev, err := range client.Stream(t.Context(), core.Request{
		Messages: []core.Message{core.UserMessage("Olá!")},
	}) {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		switch ev.Kind {
		case core.EventText:
			text = ev.Text
		case core.EventDone:
			gotDone = true
		}
	}

	if text != "Olá!" {
		t.Errorf("texto acumulado: want %q, got %q", "Olá!", text)
	}
	if !gotDone {
		t.Error("EventDone não recebido")
	}
}
