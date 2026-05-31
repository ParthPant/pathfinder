package graph

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/stores"
)

type IGraph[T any, E any, I any] interface {
	Run(context.Context) (<-chan RunEvent[E], <-chan RunInterrupt[T, E, I])
	SetState(T) error
	GetState() T
	SetEntryNode(string)
	SetNode(string)
	CurrentNode() string
	IsCompleted() bool
	SetCompleted()
	HasNodeName(string) bool
	Interrupt(context.Context, *I, chan<- RunInterrupt[T, E, I]) (T, error)
}

type BaseGraph[T any, E any, I any] struct {
	state         T
	currentNode   *string
	nodes         map[string]Node[T, E, I]
	completed     bool
	maxIterations int
	entryNode     string
	sessionId     string
	store         stores.IStore[T]
}

type RunInterrupt[T any, E any, I any] struct {
	Value *I
	Resp  chan ICommand[T, E, I]
}

func NewRunInterrupt[T any, E any, I any](v *I) RunInterrupt[T, E, I] {
	return RunInterrupt[T, E, I]{
		Value: v,
		Resp:  make(chan ICommand[T, E, I]),
	}
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

func NewBaseGraph[T any, E any, I any](state T, nodes map[string]Node[T, E, I], entryNode string, maxIterations int, store stores.IStore[T]) BaseGraph[T, E, I] {
	return BaseGraph[T, E, I]{
		state:         state,
		currentNode:   nil,
		nodes:         nodes,
		completed:     false,
		maxIterations: maxIterations,
		entryNode:     entryNode,
		store:         store,
	}
}

func (g *BaseGraph[T, E, I]) IsCompleted() bool {
	return g.completed
}

func (g *BaseGraph[T, E, I]) CurrentNode() string {
	if g.currentNode == nil {
		return g.entryNode
	} else {
		return *g.currentNode
	}
}

func (g *BaseGraph[T, E, I]) SetNode(n string) {
	g.currentNode = &n
}

func (g *BaseGraph[T, E, I]) SetEntryNode(n string) {
	g.entryNode = n
}

func (g *BaseGraph[T, E, I]) SetState(newState T) error {
	g.state = newState
	g.store.SaveState(g.sessionId, g.state)
	return nil
}

func (g *BaseGraph[T, E, I]) SetCompleted() {
	g.completed = true
}

func (g *BaseGraph[T, E, I]) Reset() {
	g.completed = false
	g.currentNode = nil
}

func (g *BaseGraph[T, E, I]) GetState() T {
	return g.state
}

func (g *BaseGraph[T, E, I]) Run(ctx context.Context) (<-chan RunEvent[E], <-chan RunInterrupt[T, E, I]) {
	// TODO: Detect cycles.

	ch := make(chan RunEvent[E], 10)
	intch := make(chan RunInterrupt[T, E, I])

	go func() {
		defer close(ch)
		defer close(intch)

		for i := 0; i <= g.maxIterations; i++ {
			cmd, err := g.nodes[g.CurrentNode()](ctx, ch, intch, g.state)
			if err != nil {
				ch <- NewGraphErrorEvent[E](err)
				return
			}

			if err := cmd.ApplyTo(g); err != nil {
				ch <- NewGraphErrorEvent[E](err)
				g.Reset()
				return
			}

			if g.completed {
				slog.Debug("Resetting graph to entry point.")
				g.Reset()
				return
			}
		}
	}()

	return ch, intch
}

func (g *BaseGraph[T, E, I]) HasNodeName(n string) bool {
	if _, ok := g.nodes[n]; ok {
		return true
	}
	return false
}

func (g *BaseGraph[T, E, I]) Interrupt(ctx context.Context, val *I, intch chan<- RunInterrupt[T, E, I]) (T, error) {
	interrupt := NewRunInterrupt[T, E](val)
	slog.Debug("Raising Interrupt", "type", interrupt.Value)

	select {
	case intch <- interrupt:
	case <-ctx.Done():
		slog.Error("Context finished.", "error", ctx.Err().Error())
		return g.state, ctx.Err()
	}

	select {
	case cmd := <-interrupt.Resp:
		cmd.ApplyTo(g)
	case <-ctx.Done():
		slog.Error("Context finished.", "error", ctx.Err().Error())
		return g.state, ctx.Err()
	}
	return g.state, nil
}

func (g *BaseGraph[T, E, I]) NewSession() (string, error) {
	id, err := g.store.NewSession()
	if err != nil {
		return "", err
	}
	g.sessionId = id
	return id, nil
}
