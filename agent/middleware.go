package agent

import (
	"context"
)

type IMiddleware[T any] interface {
	OnAttach(*Agent) error
	BeforeAgent(context.Context, AgentEventCh, AgentIntrCh, T) (T, error)
	AfterAgent(context.Context, AgentEventCh, AgentIntrCh, T) (T, error)
	BeforeLlm(context.Context, AgentEventCh, AgentIntrCh, T) (T, error)
	AfterLlm(context.Context, AgentEventCh, AgentIntrCh, T) (T, error)
}
