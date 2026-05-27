package graph

import (
	"fmt"
	"log/slog"
)

type ICommand[T any, E any] interface {
	ApplyTo(IGraph[T, E]) error
}

type Command[T any, E any] struct {
	Goto     string
	NewState T
}

func (c *Command[T, E]) ApplyTo(g IGraph[T, E]) error {
	slog.Debug("Apply Command", "goto", c.Goto)
	if !g.HasNodeName(c.Goto) {
		return fmt.Errorf("Node not present in Graph %s=%s", "goto", c.Goto)
	}
	g.SetState(c.NewState)
	g.SetNode(c.Goto)
	return nil
}

func NewCommand[T any, E any](goTo string, newState T) *Command[T, E] {
	return &Command[T, E]{
		goTo,
		newState,
	}
}

type ExitCommand[T any, E any] struct{}

func (c *ExitCommand[T, E]) ApplyTo(g IGraph[T, E]) error {
	g.SetCompleted()
	return nil
}

func NewExitCommand[T any, E any]() *ExitCommand[T, E] {
	return &ExitCommand[T, E]{}
}

type noOpCommand[T any, E any] struct{}

func (c *noOpCommand[T, E]) ApplyTo(g IGraph[T, E]) error {
	return nil
}

func NoOpCommand[T any, E any]() *noOpCommand[T, E] {
	return &noOpCommand[T, E]{}
}
