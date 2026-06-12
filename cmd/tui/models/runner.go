package models

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/cmd/tui/types"
	tea "github.com/charmbracelet/bubbletea"
)

// runAgentSession calls agent.Run(ctx) in a dedicated goroutine and pipes all
// events and interrupts back to the bubbletea program as typed messages.
// When the agent finishes (both channels closed), a ProcessingFinishedMsg is sent.
func runAgentSession(ctx context.Context, ag *agent.Agent, program *tea.Program) {
	// Signal that the agent has started processing
	slog.Info("Running Agent session")
	program.Send(types.ProcessingStartedMsg{})

	ch, chintr := ag.Run(ctx)

	for ch != nil || chintr != nil {
		select {
		case e, ok := <-ch:
			if !ok {
				ch = nil
				continue
			}
			if e.Err != nil {
				// Wrap the error as an AGENTERR event
				program.Send(types.ConversationEvent{
					AgentEvent: agent.NewErrorEvent(e.Err),
				})
			} else if e.Value != nil {
				program.Send(types.ConversationEvent{AgentEvent: e.Value})
			}

		case intr, ok := <-chintr:
			if !ok {
				chintr = nil
				continue
			}
			program.Send(types.AgentInterruptMsg{Interrupt: intr})
		}
	}

	program.Send(types.ProcessingFinishedMsg{})
}

// fetchSessionListCmd returns a tea.Cmd that queries the agent's session store
// and emits a SessionListUpdateMsg with the current sessions.
func fetchSessionListCmd(ag *agent.Agent) tea.Cmd {
	return func() tea.Msg {
		sessions := ag.SessionStore().ListSessions()
		return types.SessionListUpdateMsg{Sessions: sessions}
	}
}