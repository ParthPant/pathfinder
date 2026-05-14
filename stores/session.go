package stores

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type IStore[T any] interface {
	NewSession() (string, error)
	GetById(id string) (T, error)
	SaveState(sessionId string, state T) error
}

type InMemoryStore[T any] struct {
	sessions map[string]*T
}

func NewInMemoryStore[T any]() *InMemoryStore[T] {
	return &InMemoryStore[T]{
		sessions: make(map[string]*T),
	}
}

func (repo *InMemoryStore[T]) NewSession() (string, error) {
	sid, err := uuid.NewV7()

	if err != nil {
		slog.Error("Error while generating session id", "error", err)
		return "", err
	}

	key := sid.String()
	repo.sessions[key] = new(T)
	return key, nil
}

// This returns a copy of the state.
func (repo *InMemoryStore[T]) GetById(id string) (T, error) {
	state, ok := repo.sessions[id]
	if !ok {
		slog.Warn("Session Not Found", "id", id)
		return *new(T), fmt.Errorf("Session Not Found id=%s", id)
	}
	return *state, nil
}

// TODO: Make this function Atomic
func (repo *InMemoryStore[T]) SaveState(sessionId string, state T) error {
	if _, ok := repo.sessions[sessionId]; ok {
		repo.sessions[sessionId] = &state
		return nil
	}
	slog.Warn("Session Not Found", "id", sessionId)
	return fmt.Errorf("Session Not Found id=%s", sessionId)
}
