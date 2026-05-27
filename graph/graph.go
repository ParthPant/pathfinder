package graph

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/stores"
)

type IGraph[T any, E any] interface {
	Run(context.Context) <-chan RunEvent[E]
	SetState(T) error
	SetEntryNode(string)
	SetNode(string)
	CurrentNode() string
	SetCompleted()
	HasNodeName(string) bool
}

type BaseGraph[T any, E any] struct {
	state         T
	currentNode   *string
	nodes         map[string]Node[T, E]
	completed     bool
	maxIterations int
	entryNode     string
	sessionId     string
	store         stores.IStore[T]
}

type RunEvent[E any] struct {
	Value *E
	Err   error
}

func NewEvent[E any](v *E) RunEvent[E] {
	return RunEvent[E]{
		Value: v,
		Err:   nil,
	}
}

func NewGraphErrorEvent[E any](e error) RunEvent[E] {
	return RunEvent[E]{Value: nil, Err: e}
}

func NewBaseGraph[T any, E any](state T, nodes map[string]Node[T, E], entryNode string, maxIterations int, store stores.IStore[T]) BaseGraph[T, E] {
	return BaseGraph[T, E]{
		state:         state,
		currentNode:   nil,
		nodes:         nodes,
		completed:     false,
		maxIterations: maxIterations,
		entryNode:     entryNode,
		store:         store,
	}
}

func (g *BaseGraph[T, E]) CurrentNode() string {
	if g.currentNode == nil {
		return g.entryNode
	} else {
		return *g.currentNode
	}
}

func (g *BaseGraph[T, E]) SetNode(n string) {
	g.currentNode = &n
}

func (g *BaseGraph[T, E]) SetEntryNode(n string) {
	g.entryNode = n
}

func (g *BaseGraph[T, E]) SetState(newState T) error {
	g.state = newState
	g.store.SaveState(g.sessionId, g.state)
	return nil
}

func (g *BaseGraph[T, E]) SetCompleted() {
	g.completed = true
}

func (g *BaseGraph[T, E]) Reset() {
	g.completed = false
	g.currentNode = nil
}

func (g *BaseGraph[T, E]) GetState() T {
	return g.state
}

func (g *BaseGraph[T, E]) Run(ctx context.Context) <-chan RunEvent[E] {
	// TODO: Detect cycles.

	ch := make(chan RunEvent[E], 10)

	go func() {
		defer close(ch)

		for i := 0; i <= g.maxIterations; i++ {
			cmd, err := g.nodes[g.CurrentNode()](ctx, ch, g.state)
			if err != nil {
				ch <- NewGraphErrorEvent[E](err)
				return
			}

			if err := cmd.ApplyTo(g); err != nil {
				ch <- NewGraphErrorEvent[E](err)
				return
			}

			if g.completed {
				slog.Debug("Resetting graph to entry point.")
				g.Reset()
				return
			}
		}
	}()

	return ch
}

func (g *BaseGraph[T, E]) HasNodeName(n string) bool {
	if _, ok := g.nodes[n]; ok {
		return true
	}
	return false
}

func (g *BaseGraph[T, E]) NewSession() (string, error) {
	id, err := g.store.NewSession()
	if err != nil {
		return "", err
	}
	g.sessionId = id
	return id, nil
}
