package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/ParthPant/pathfinder/messages"
	"github.com/creasty/defaults"
)

type IToolExecutor interface {
	Execute(ctx context.Context, functionCall messages.OutputFunctionCall) (messages.Message, error)
	RegisterFunction(fd FunctionDefinition) error
	GetTools() []FunctionDefinition
}

type ToolExecutor struct {
	Functions map[string]FunctionDefinition
}

func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{
		Functions: make(map[string]FunctionDefinition),
	}
}

func (e *ToolExecutor) GetTools() []FunctionDefinition {
	tools := make([]FunctionDefinition, 0, len(e.Functions))
	for _, tool := range e.Functions {
		tools = append(tools, tool)
	}
	return tools
}

func (e *ToolExecutor) RegisterFunction(fd FunctionDefinition) error {
	key := fd.Name
	_, exists := e.Functions[key]
	if exists {
		return fmt.Errorf("Function Definition with name %s already exists.", fd.Name)
	}
	e.Functions[key] = fd
	return nil
}

func (e *ToolExecutor) Execute(ctx context.Context, functionCall messages.OutputFunctionCall) (messages.Message, error) {
	slog.Debug("Executiing Tool", "name", functionCall.Name, "callId", functionCall.CallId)

	fd, ok := e.Functions[functionCall.Name]
	if !ok {
		return messages.Message{}, fmt.Errorf("Function not found.")
	}

	output, err := call(ctx, fd.Function, functionCall.Arguments)

	if err != nil {
		slog.Error("Error while Function execution", "error", err.Error())
		// return messages.Message{}, err
	}

	outputStr, err := json.Marshal(output)
	if err != nil {
		slog.Error("Error while marshalling tool output", "output", output)
		return messages.Message{}, err
	}
	slog.Debug("Tool Output", "name", functionCall.Name, "callId", functionCall.CallId, "output", outputStr[:min(80, len(outputStr))])

	toolMessage := messages.ToolMessage{
		Type:   "function_call_output",
		CallId: functionCall.CallId,
		Output: string(outputStr),
		Id:     functionCall.Id,
	}

	return messages.Message{
		Role:        "tool",
		ToolMessage: toolMessage,
	}, nil
}

func call(ctx context.Context, fn any, params string) (any, error) {
	fnValue := reflect.ValueOf(fn)
	fnType := reflect.TypeOf(fn)

	paramType := fnType.In(1)
	paramPtr := reflect.New(paramType)

	defaults.Set(paramPtr.Interface())

	if err := json.Unmarshal([]byte(params), paramPtr.Interface()); err != nil {
		panic(err)
	}

	args := []reflect.Value{reflect.ValueOf(ctx), paramPtr.Elem()}

	out := fnValue.Call(args)

	result := out[0].Interface()
	var err error = nil

	if !out[1].IsNil() {
		err = out[1].Interface().(error)
	}

	return result, err
}
