package main

import (
	"context"

	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/tools"
)

type ILlm interface {
	NewResponse(context.Context, []messages.Message) (messages.Message, error)
	//NewStream(input []Message) Message
}

type IToolCallingLlm interface {
	ILlm
	RegisterFunctionDefinition(tool tools.FunctionDefinition) error
}
