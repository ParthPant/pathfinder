package graph

import "context"

type INode[T any] interface {
	Run(context.Context) (ICommand[T], error)
}
