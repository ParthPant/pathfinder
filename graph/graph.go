package graph

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ParthPant/pathfinder/stores"
)

type IGraph[T any] interface {
	Run(context.Context) error
	SetState(T) error
	SetEntryNode(string)
	SetNode(string)
	CurrentNode() string
	SetCompleted()
	HasNodeName(string) bool
}

type BaseGraph[T any] struct {
	state         T
	currentNode   *string
	nodes         map[string]Node[T]
	completed     bool
	maxIterations int
	entryNode     string
	sessionId     string
	store         stores.IStore[T]
}

func NewBaseGraph[T any](state T, nodes map[string]Node[T], entryNode string, maxIterations int, store stores.IStore[T]) BaseGraph[T] {
	return BaseGraph[T]{
		state:         state,
		currentNode:   nil,
		nodes:         nodes,
		completed:     false,
		maxIterations: maxIterations,
		entryNode:     entryNode,
		store:         store,
	}
}

func (g *BaseGraph[T]) CurrentNode() string {
	if g.currentNode == nil {
		return g.entryNode
	} else {
		return *g.currentNode
	}
}

func (g *BaseGraph[T]) SetNode(n string) {
	g.currentNode = &n
}

func (g *BaseGraph[T]) SetEntryNode(n string) {
	g.entryNode = n
}

func (g *BaseGraph[T]) SetState(newState T) error {
	g.state = newState
	g.store.SaveState(g.sessionId, g.state)
	return nil
}

func (g *BaseGraph[T]) SetCompleted() {
	g.completed = true
}

func (g *BaseGraph[T]) Reset() {
	g.completed = false
	g.currentNode = nil
}

func (g *BaseGraph[T]) GetState() T {
	return g.state
}

func (g *BaseGraph[T]) Run(ctx context.Context) error {
	// TODO: Detect cycles.

	for i := 0; i <= g.maxIterations; i++ {
		cmd, err := g.nodes[g.CurrentNode()](ctx, g.state)
		if err != nil {
			return err
		}

		if err := cmd.ApplyTo(g); err != nil {
			panic(err)
		}

		if g.completed {
			slog.Debug("Resetting graph to entry point.")
			g.Reset()
			return nil
		}
	}
	return errors.New("Max graph iterations reached.")
}

func (g *BaseGraph[T]) HasNodeName(n string) bool {
	if _, ok := g.nodes[n]; ok {
		return true
	}
	return false
}

func (g *BaseGraph[T]) NewSession() (string, error) {
	id, err := g.store.NewSession()
	if err != nil {
		return "", err
	}
	g.sessionId = id
	return id, nil
}
