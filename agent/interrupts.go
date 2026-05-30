package agent

import (
	"github.com/ParthPant/pathfinder/messages"
)

type AgentInterruptType = string

const (
	INTR_TOOLCALL AgentInterruptType = "INTR_TOOLCALL"
)

type AgentInterrupt struct {
	Type       AgentInterruptType
	OfToolCall ToolCallInterrupt
}

type ToolCallInterrupt struct {
	Call messages.OutputFunctionCall
}