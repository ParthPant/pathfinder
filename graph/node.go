package graph

import "context"

// type INode[T any] interface {
// 	Run(context.Context) (ICommand[T], error)
// }

type Node[T any, E any] = func(context.Context, chan<- RunEvent[E], T) (ICommand[T, E], error)
