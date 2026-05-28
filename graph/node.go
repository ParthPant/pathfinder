package graph

import "context"

// type INode[T any] interface {
// 	Run(context.Context) (ICommand[T], error)
// }

type Node[T any, E any, I any] = func(context.Context, chan<- RunEvent[E], chan<- RunInterrupt[T, E, I], T) (ICommand[T, E, I], error)
