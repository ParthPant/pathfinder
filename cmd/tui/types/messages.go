package types

import (
	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/graph"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/stores"
)

// Deprecated: AgentEventMsg is superseded by ConversationEvent.
// Kept for backward compatibility during transition.
type AgentEventMsg struct {
	Event *agent.AgentEvent
}

// ConversationEvent is the unified message type for the conversation panel.
// Only one of AgentEvent or UserInput should be set per message.
// Agent events come from the runner goroutine; user input comes from RootModel.
type ConversationEvent struct {
	AgentEvent *agent.AgentEvent
	UserInput  *messages.Message
}

// FocusChangeMsg is emitted by RootModel when Tab cycles focus.
// Sub-models consume this to update their focused field.
type FocusChangeMsg struct {
	ActivePaneId int
}

// AgentInterruptMsg wraps an agent interrupt along with its response channel
// so the TUI can prompt the user and send the response back.
type AgentInterruptMsg struct {
	Interrupt graph.RunInterrupt[agent.AgentState, agent.AgentEvent, agent.AgentInterrupt]
}

// SendUserInputMsg signals that the user has submitted text from the input panel.
type SendUserInputMsg struct {
	Text string
}

// SessionSwitchMsg requests switching the active session to a different one.
type SessionSwitchMsg struct {
	ID string
}

// SessionListToggleMsg toggles the session list panel between collapsed/expanded.
type SessionListToggleMsg struct{}

// SessionListUpdateMsg carries an updated list of sessions for the session list panel.
type SessionListUpdateMsg struct {
	Sessions []stores.Session
}

// ProcessingStartedMsg signals the agent has started processing (show spinner).
type ProcessingStartedMsg struct{}

// ProcessingFinishedMsg signals the agent has finished processing (hide spinner).
type ProcessingFinishedMsg struct{}

// ExitTUIMsg signals that the TUI should shut down cleanly.
type ExitTUIMsg struct{}
