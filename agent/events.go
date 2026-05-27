package agent

import (
	"github.com/ParthPant/pathfinder/messages"
)

type AgentEventType = string

const (
	AIRESP   AgentEventType = "AIRESP"
	TOOLCALL AgentEventType = "TOOLCALL"
	TOOLRESP AgentEventType = "TOOLRESP"
	AGENTERR AgentEventType = "AGENTERR"
)

type AgentEvent struct {
	Type           AgentEventType
	OfAiResponse   EventAIResponse
	OfToolCall     EventToolCall
	OfToolResponse EventToolResponse
	OfError        EventError
}

type EventAIResponse struct {
	Message messages.Message
}

type EventToolCall struct {
	Call messages.OutputFunctionCall
}

type EventToolResponse struct {
	Message messages.Message
}

type EventError struct {
	Err error
}

// NewAiResponseEvent creates a new AgentEvent of type AIRESP.
func NewAiResponseEvent(msg messages.Message) *AgentEvent {
	return &AgentEvent{
		Type:         AIRESP,
		OfAiResponse: EventAIResponse{Message: msg},
	}
}

// NewToolCallEvent creates a new AgentEvent of type TOOLCALL.
func NewToolCallEvent(call messages.OutputFunctionCall) *AgentEvent {
	return &AgentEvent{
		Type:       TOOLCALL,
		OfToolCall: EventToolCall{Call: call},
	}
}

// NewToolResponseEvent creates a new AgentEvent of type TOOLRESP.
func NewToolResponseEvent(msg messages.Message) *AgentEvent {
	return &AgentEvent{
		Type:           TOOLRESP,
		OfToolResponse: EventToolResponse{Message: msg},
	}
}

// NewErrorEvent creates a new AgentEvent of type AGENTERR.
func NewErrorEvent(err error) *AgentEvent {
	return &AgentEvent{
		Type:    AGENTERR,
		OfError: EventError{Err: err},
	}
}
