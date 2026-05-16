package agent

import (
	"context"
)

type IMiddleware[T any] interface {
	OnAttach(*Agent) error
	BeforeAgent(context.Context, T) (T, error)
	AfterAgent(context.Context, T) (T, error)
	BeforeLlm(context.Context, T) (T, error)
	AfterLlm(context.Context, T) (T, error)
}
