package memory_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/brunoochoa/goai/core"
	"github.com/brunoochoa/goai/memory"
)

// ── WindowMemory ──────────────────────────────────────────────────────────────

func TestWindow_AddAndLoad(t *testing.T) {
	m := memory.NewWindow(5)

	if err := m.Add(t.Context(), core.UserMessage("olá")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := m.Add(t.Context(), core.AssistantMessage("oi!")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	msgs, err := m.Load(t.Context())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("esperado 2 mensagens, got %d", len(msgs))
	}
	if msgs[0].Text() != "olá" {
		t.Errorf("msgs[0]: want %q, got %q", "olá", msgs[0].Text())
	}
}

func TestWindow_LimitEnforced(t *testing.T) {
	m := memory.NewWindow(3)

	// Adiciona 5 mensagens — só as últimas 3 devem sobrar.
	for i := range 5 {
		_ = m.Add(t.Context(), core.UserMessage(string(rune('A'+i))))
	}

	msgs, _ := m.Load(t.Context())
	if len(msgs) != 3 {
		t.Errorf("esperado 3 mensagens após overflow, got %d", len(msgs))
	}
	// Deve ter C, D, E (as últimas 3)
	if msgs[0].Text() != "C" {
		t.Errorf("primeira mensagem: want C, got %q", msgs[0].Text())
	}
	if msgs[2].Text() != "E" {
		t.Errorf("última mensagem: want E, got %q", msgs[2].Text())
	}
}

func TestWindow_Clear(t *testing.T) {
	m := memory.NewWindow(10)
	_ = m.Add(t.Context(), core.UserMessage("msg"))

	if err := m.Clear(t.Context()); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	msgs, _ := m.Load(t.Context())
	if len(msgs) != 0 {
		t.Errorf("esperado 0 mensagens após Clear, got %d", len(msgs))
	}
}

func TestWindow_DefaultSize(t *testing.T) {
	// maxSize <= 0 deve usar o padrão de 20.
	m := memory.NewWindow(0)
	for range 25 {
		_ = m.Add(t.Context(), core.UserMessage("x"))
	}
	if m.Len() != 20 {
		t.Errorf("esperado 20 (default), got %d", m.Len())
	}
}

func TestWindow_LoadReturnsCopy(t *testing.T) {
	m := memory.NewWindow(5)
	_ = m.Add(t.Context(), core.UserMessage("original"))

	msgs, _ := m.Load(t.Context())
	msgs[0] = core.UserMessage("modificado") // muta a cópia

	// O estado interno não deve ter mudado.
	msgs2, _ := m.Load(t.Context())
	if msgs2[0].Text() != "original" {
		t.Error("Load não retornou cópia — estado interno foi mutado")
	}
}

func TestWindow_Concurrent(t *testing.T) {
	m := memory.NewWindow(100)
	var wg sync.WaitGroup

	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.Add(t.Context(), core.UserMessage("concurrent"))
		}()
	}
	wg.Wait()

	if m.Len() != 50 {
		t.Errorf("esperado 50 mensagens, got %d", m.Len())
	}
}

// ── FileMemory ────────────────────────────────────────────────────────────────

func TestFile_AddLoadPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	m, err := memory.NewFile(path)
	if err != nil {
		t.Fatalf("NewFile: %v", err)
	}

	_ = m.Add(t.Context(), core.UserMessage("persistido"))
	_ = m.Add(t.Context(), core.AssistantMessage("ok!"))

	// Cria uma nova instância apontando para o mesmo arquivo.
	m2, _ := memory.NewFile(path)
	msgs, err := m2.Load(t.Context())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("esperado 2 mensagens após reload, got %d", len(msgs))
	}
	if msgs[0].Text() != "persistido" {
		t.Errorf("msgs[0]: want %q, got %q", "persistido", msgs[0].Text())
	}
}

func TestFile_LoadEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novo.json")

	m, _ := memory.NewFile(path)
	msgs, err := m.Load(t.Context())
	if err != nil {
		t.Fatalf("Load em arquivo inexistente: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("esperado slice vazio, got %d mensagens", len(msgs))
	}
}

func TestFile_Clear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	m, _ := memory.NewFile(path)

	_ = m.Add(t.Context(), core.UserMessage("dados"))
	_ = m.Clear(t.Context())

	msgs, _ := m.Load(t.Context())
	if len(msgs) != 0 {
		t.Errorf("esperado 0 após Clear, got %d", len(msgs))
	}
}

func TestFile_InvalidDirectory(t *testing.T) {
	_, err := memory.NewFile("/diretorio/que/nao/existe/history.json")
	if err == nil {
		t.Fatal("esperado erro para diretório inexistente")
	}
}

func TestFile_AtomicWrite(t *testing.T) {
	// Verifica que o arquivo .tmp não fica para trás após Add bem-sucedido.
	path := filepath.Join(t.TempDir(), "history.json")
	m, _ := memory.NewFile(path)

	_ = m.Add(t.Context(), core.UserMessage("msg"))

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("arquivo .tmp não deveria existir após Add bem-sucedido")
	}
}
