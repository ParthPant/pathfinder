package main

import (
	"fmt"
	"log/slog"

	"github.com/ParthPant/pathfinder/messages"
	"github.com/google/uuid"
)

type ISessionRepository interface {
	NewSession() (string, error)
	GetById(id string) ([]messages.Message, error)
	SaveMessage(sessionId string, message messages.Message) error
}

type InMemorySessionRepo struct {
	sessions map[string][]messages.Message
}

func NewInMemorySessionRepo() *InMemorySessionRepo {
	return &InMemorySessionRepo{
		sessions: make(map[string][]messages.Message),
	}
}

func (repo *InMemorySessionRepo) NewSession() (string, error) {
	sid, err := uuid.NewV7()

	if err != nil {
		slog.Error("Error while generating session id", "error", err)
		return "", err
	}

	key := sid.String()
	repo.sessions[key] = []messages.Message{}
	return key, nil
}

func (repo *InMemorySessionRepo) GetById(id string) ([]messages.Message, error) {
	messages, ok := repo.sessions[id]
	if !ok {
		slog.Warn("Session Not Found", "id", id)
		return nil, fmt.Errorf("Session Not Found id=%s", id)
	}
	return messages, nil
}

// TODO: Make this function Atomic
func (repo *InMemorySessionRepo) SaveMessage(sessionId string, message messages.Message) error {
	if _, ok := repo.sessions[sessionId]; ok {
		repo.sessions[sessionId] = append(repo.sessions[sessionId], message)
		return nil
	}
	slog.Warn("Session Not Found", "id", sessionId)
	return fmt.Errorf("Session Not Found id=%s", sessionId)
}
