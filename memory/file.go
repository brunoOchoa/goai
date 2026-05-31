package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/brunoochoa/goai/core"
)

// FileMemory persists the conversation history to a JSON file on disk.
// The history survives program restarts — useful for long-running assistants.
// It is safe for concurrent use by multiple goroutines.
//
// Example:
//
//	mem, err := memory.NewFile("./history.json")
//	session := chat.New(client, chat.WithMemory(mem))
type FileMemory struct {
	mu   sync.RWMutex
	path string
}

// fileData é o formato serializado no arquivo JSON.
type fileData struct {
	Messages []core.Message `json:"messages"`
}

// NewFile creates a FileMemory backed by the given file path.
// The file is created automatically on the first Add call if it does not exist.
// Parent directories must already exist.
func NewFile(path string) (*FileMemory, error) {
	// Valida que o diretório pai existe.
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("memory: diretório %q não existe: %w", dir, err)
	}
	return &FileMemory{path: path}, nil
}

// Add appends a message and immediately persists the updated history to disk.
func (m *FileMemory) Add(ctx context.Context, msg core.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	msgs, err := m.load()
	if err != nil {
		return err
	}

	msgs = append(msgs, msg)
	return m.save(msgs)
}

// Load reads the full history from disk.
// Returns an empty slice if the file does not exist yet.
func (m *FileMemory) Load(_ context.Context) ([]core.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.load()
}

// Clear deletes all messages — overwrites the file with an empty history.
func (m *FileMemory) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.save(nil)
}

// Path returns the file path used by this memory.
func (m *FileMemory) Path() string { return m.path }

// load lê o arquivo e retorna as mensagens. Deve ser chamado com lock.
func (m *FileMemory) load() ([]core.Message, error) {
	data, err := os.ReadFile(m.path)
	if os.IsNotExist(err) {
		// Arquivo ainda não existe — histórico vazio.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("memory: lendo %q: %w", m.path, err)
	}

	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return nil, fmt.Errorf("memory: JSON inválido em %q: %w", m.path, err)
	}

	return fd.Messages, nil
}

// save serializa as mensagens e grava no arquivo de forma atômica.
// Usa arquivo temporário + rename para evitar corrupção em caso de falha.
func (m *FileMemory) save(msgs []core.Message) error {
	fd := fileData{Messages: msgs}
	data, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return fmt.Errorf("memory: serializando histórico: %w", err)
	}

	// Gravação atômica: escreve em temp e renomeia.
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("memory: gravando arquivo temporário: %w", err)
	}
	if err := os.Rename(tmp, m.path); err != nil {
		os.Remove(tmp) // limpa o temp em caso de falha no rename
		return fmt.Errorf("memory: renomeando para %q: %w", m.path, err)
	}

	return nil
}
