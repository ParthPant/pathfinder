package agent

import (
	"context"

	"github.com/ParthPant/pathfinder/graph"
)

type IMiddleware[T any] interface {
	OnAttach(*Agent) error
	BeforeAgent(context.Context, chan<- graph.RunEvent[AgentEvent], chan<- graph.RunInterrupt[AgentInterrupt], T) (T, error)
	AfterAgent(context.Context, chan<- graph.RunEvent[AgentEvent], chan<- graph.RunInterrupt[AgentInterrupt], T) (T, error)
	BeforeLlm(context.Context, chan<- graph.RunEvent[AgentEvent], chan<- graph.RunInterrupt[AgentInterrupt], T) (T, error)
	AfterLlm(context.Context, chan<- graph.RunEvent[AgentEvent], chan<- graph.RunInterrupt[AgentInterrupt], T) (T, error)
}
