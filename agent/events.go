package agent

import (
	"github.com/ParthPant/pathfinder/messages"
)

type EventAIResponse struct {
	Message messages.Message
}

type EventToolCall struct {
	Call messages.OutputFunctionCall
}

type EventToolResponse struct {
	Message messages.Message
}
