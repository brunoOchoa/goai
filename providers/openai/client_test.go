package openai_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/brunoochoa/goai/core"
	"github.com/brunoochoa/goai/providers/openai"
)

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
	t.Setenv("OPENAI_API_KEY", "")
	_, err := openai.NewClient()
	if err == nil {
		t.Fatal("esperado erro com API key vazia")
	}
}

func TestNewClient_WithBaseURL_NoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	// BaseURL definida → provider compatível, não exige key real.
	_, err := openai.NewClient(openai.WithBaseURL("http://localhost:11434/v1"))
	if err != nil {
		t.Fatalf("não esperado erro com baseURL definida: %v", err)
	}
}

func TestChat_Success(t *testing.T) {
	srv := httptest.NewServer(fixtureHandler(t, "success.json", http.StatusOK))
	defer srv.Close()

	client, err := openai.NewClient(
		openai.WithAPIKey("test-key"),
		openai.WithBaseURL(srv.URL),
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
	if resp.StopReason != "stop" {
		t.Errorf("StopReason: want stop, got %q", resp.StopReason)
	}
}

func TestChat_RateLimit(t *testing.T) {
	srv := httptest.NewServer(fixtureHandler(t, "rate_limit.json", http.StatusTooManyRequests))
	defer srv.Close()

	client, err := openai.NewClient(
		openai.WithAPIKey("test-key"),
		openai.WithBaseURL(srv.URL),
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

func TestModelID(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test")
	client, err := openai.NewClient(openai.WithModel("gpt-4o-mini"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.ModelID() != "gpt-4o-mini" {
		t.Errorf("ModelID: want gpt-4o-mini, got %s", client.ModelID())
	}
}
