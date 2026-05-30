package core

import "context"

// Memory armazena e recupera histórico de conversa.
type Memory interface {
	Add(ctx context.Context, msg Message) error
	Load(ctx context.Context) ([]Message, error)
	Clear(ctx context.Context) error
}
