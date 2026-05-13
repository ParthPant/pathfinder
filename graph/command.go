package graph

import (
	"fmt"
	"log/slog"
)

type ICommand[T any] interface {
	ApplyTo(IGraph[T]) error
}

type Command[T any] struct {
	Goto     string
	NewState T
}

func (c *Command[T]) ApplyTo(g IGraph[T]) error {
	slog.Debug("Apply Command", "goto", c.Goto)
	if !g.HasNodeName(c.Goto) {
		return fmt.Errorf("Node not present in Graph %s=%s", "goto", c.Goto)
	}
	g.SetState(c.NewState)
	g.SetNode(c.Goto)
	return nil
}

func NewCommand[T any](goTo string, newState T) *Command[T] {
	return &Command[T]{
		goTo,
		newState,
	}
}

type ExitCommand[T any] struct{}

func (c *ExitCommand[T]) ApplyTo(g IGraph[T]) error {
	g.SetCompleted()
	return nil
}

func NewExitCommand[T any]() *ExitCommand[T] {
	return &ExitCommand[T]{}
}
