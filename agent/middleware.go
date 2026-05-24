package agent

import (
	"context"
)

type IMiddleware[T any] interface {
	OnAttach(*Agent) error
	BeforeAgent(context.Context, chan<- any, T) (T, error)
	AfterAgent(context.Context, chan<- any, T) (T, error)
	BeforeLlm(context.Context, chan<- any, T) (T, error)
	AfterLlm(context.Context, chan<- any, T) (T, error)
}
