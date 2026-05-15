package middleware

import (
	"context"
)

type IMiddleware[T any] interface {
	IBeforeAgentMiddleware[T]
	IAfterAgentMiddleware[T]
	IBeforeLlmMiddleware[T]
	IAfterLlmMiddleware[T]
}

type IBeforeAgentMiddleware[T any] interface {
	BeforeAgent(context.Context, T) (T, error)
}

type IAfterAgentMiddleware[T any] interface {
	AfterAgent(context.Context, T) (T, error)
}

type IBeforeLlmMiddleware[T any] interface {
	BeforeLlm(context.Context, T) (T, error)
}

type IAfterLlmMiddleware[T any] interface {
	AfterLlm(context.Context, T) (T, error)
}
