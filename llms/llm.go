package llms

import (
	"context"

	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/tools"
	"github.com/openai/openai-go/v3/shared"
)

type ReasoningEffortType = shared.ReasoningEffort

type ILlm interface {
	NewResponse(context.Context, []messages.Message) (messages.Message, error)
	Config() *LlmConfig
	//NewStream(input []Message) Message
}

type IToolCallingLlm interface {
	ILlm
	RegisterFunctionDefinition(tool tools.FunctionDefinition) error
}

type LlmConfig struct {
	BaseUrl         string
	APIKey          string
	Model           string
	MaxOutputTokens int64
	ReasoningEffort ReasoningEffortType // "" defaults to medium in NewResponse
}
