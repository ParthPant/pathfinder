package main

import (
	"context"
	"errors"
	"log/slog"
)

type INode interface {
	Next(context.Context) error
}

type IGraph[T any] interface {
	Run(context.Context) error
	SetState(T) error
	SetNode(INode)
}

type BaseGraph[T any] struct {
	State         T
	node          INode
	completed     bool
	maxIterations int

	entryNode INode
}

func (g *BaseGraph[T]) SetNode(n INode) {
	g.node = n
}

func (g *BaseGraph[T]) SetState(newState T) error {
	g.State = newState
	return nil
}

func (g *BaseGraph[T]) SetCompleted() {
	g.completed = true
}

func (g *BaseGraph[T]) Reset() {
	g.completed = false
	g.SetNode(g.entryNode)
}

func (g *BaseGraph[T]) Run(ctx context.Context) error {
	// TODO: Detect cycles.

	for i := 0; i <= g.maxIterations; i++ {
		if err := g.node.Next(ctx); err != nil {
			return err
		}
		if g.completed {
			slog.Debug("Resetting graph to entry point.")
			g.Reset()
			return nil
		}
	}
	return errors.New("Max graph iterations reached.")
}
