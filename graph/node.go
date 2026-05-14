package graph

import "context"

// type INode[T any] interface {
// 	Run(context.Context) (ICommand[T], error)
// }

type Node[T any] = func(context.Context, T) (ICommand[T], error)
