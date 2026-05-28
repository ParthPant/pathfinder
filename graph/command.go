package graph

import (
	"fmt"
	"log/slog"
)

type ICommand[T any, E any, I any] interface {
	ApplyTo(IGraph[T, E, I]) error
}

type Command[T any, E any, I any] struct {
	Goto     string
	NewState T
}

func (c *Command[T, E, I]) ApplyTo(g IGraph[T, E, I]) error {
	slog.Debug("Apply Command", "goto", c.Goto)
	if !g.HasNodeName(c.Goto) {
		return fmt.Errorf("Node not present in Graph %s=%s", "goto", c.Goto)
	}
	g.SetState(c.NewState)
	g.SetNode(c.Goto)
	return nil
}

func NewCommand[T any, E any, I any](goTo string, newState T) *Command[T, E, I] {
	return &Command[T, E, I]{
		goTo,
		newState,
	}
}

type ExitCommand[T any, E any, I any] struct{}

func (c *ExitCommand[T, E, I]) ApplyTo(g IGraph[T, E, I]) error {
	g.SetCompleted()
	return nil
}

func NewExitCommand[T any, E any, I any]() *ExitCommand[T, E, I] {
	return &ExitCommand[T, E, I]{}
}

type noOpCommand[T any, E any, I any] struct{}

func (c *noOpCommand[T, E, I]) ApplyTo(g IGraph[T, E, I]) error {
	return nil
}

func NoOpCommand[T any, E any, I any]() *noOpCommand[T, E, I] {
	return &noOpCommand[T, E, I]{}
}

type ResumeCommand[T any, E any, I any] struct {
	Resume T
}

func (c *ResumeCommand[T, E, I]) ApplyTo(g IGraph[T, E, I]) error {
	g.SetState(c.Resume)
	return nil
}
