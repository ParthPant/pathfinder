package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ParthPant/pathfinder/graph"
)

// SlashCommandsMiddleware is the first middleware that runs when a user sends input.
// It checks the user's message for slash-commands and handles them.
type SlashCommandsMiddleware struct {
	agent    *Agent
	handlers map[string]ISlashCommandHandler // flattened: alias -> handler
}

func NewSlashCommandsMiddleware() *SlashCommandsMiddleware {
	m := &SlashCommandsMiddleware{
		handlers: make(map[string]ISlashCommandHandler),
	}
	m.registerCommand(&newSessionCommand{})
	m.registerCommand(&branchSessionCommand{})
	m.registerCommand(&listSessionsCommand{})
	m.registerCommand(&exitCommand{})
	m.registerCommand(&switchSessionCommand{})
	m.registerCommand(&copyCommand{})
	m.registerCommand(&summarizeCommand{})
	m.registerCommand(&helpCommand{})
	m.registerCommand(&effortCommand{})
	return m
}

func (m *SlashCommandsMiddleware) registerCommand(handler ISlashCommandHandler) {
	m.handlers[handler.Name()] = handler
	for _, alias := range handler.Aliases() {
		m.handlers[alias] = handler
	}
}

// OnAttach stores a reference to the agent.
func (m *SlashCommandsMiddleware) OnAttach(agent *Agent) error {
	m.agent = agent
	return nil
}

// BeforeAgent checks the user's last message for slash-commands and executes them.
// This is the only hook that does work.
func (m *SlashCommandsMiddleware) BeforeAgent(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentState, error) {
	// Check if there are any user messages
	if len(state.messages) == 0 {
		return state, nil
	}

	// Get the last message (which is the user's latest input)
	lastMsg := state.messages[len(state.messages)-1]

	// Only handle user messages (not tool responses, etc.)
	if lastMsg.Role != "user" {
		return state, nil
	}

	text := lastMsg.GetTextContent()

	// Check if the message starts with '/'
	if !strings.HasPrefix(text, "/") {
		return state, nil
	}

	// Parse the command: split on first space
	parts := strings.SplitN(text, " ", 2)
	cmdName := strings.TrimPrefix(parts[0], "/") // e.g., "new", "list", "q"
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	// Look up handler
	handler, ok := m.handlers[cmdName]
	if !ok {
		// Unknown command — emit error event and exit the graph
		slog.Warn("Unknown command", "command", cmdName)
		ch <- graph.NewEvent(NewCmdResponseEvent(
			fmt.Sprintf("Unknown command: /%s.", cmdName),
		))
		m.agent.SetCompleted()
		return state, fmt.Errorf("Unkown command: /%s", cmdName)
	}

	slog.Info("Executing command", "command", cmdName, "args", args)

	// Execute the command handler
	newState, err := handler.Execute(ctx, args, m, ch, state)
	if err != nil {
		slog.Error("Command execution failed", "command", cmdName, "error", err)
		ch <- graph.NewEvent(NewCmdResponseEvent(
			fmt.Sprintf("Command /%s failed: %s", cmdName, err.Error()),
		))
	}

	return newState, err
}

func (m *SlashCommandsMiddleware) AfterAgent(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *SlashCommandsMiddleware) BeforeLlm(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *SlashCommandsMiddleware) AfterLlm(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentState, error) {
	return state, nil
}
