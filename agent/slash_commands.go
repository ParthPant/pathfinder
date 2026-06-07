package agent

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"strings"

	"github.com/ParthPant/pathfinder/graph"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/openai/openai-go/v3/shared"
)

// ISlashCommandHandler represents a single slash-command that can be registered.
type ISlashCommandHandler interface {
	// Name returns the primary command name (e.g., "new", "list").
	Name() string
	// Aliases returns alternative names for the command (e.g., ["q", "exit", "quit", "close"]).
	Aliases() []string
	// Description returns a short description for help text.
	Description() string
	// Execute processes the command. The args are everything after the command name.
	Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error)
}

// --- Command Handlers ---

// newSessionCommand handles /new
type newSessionCommand struct{}

func (c *newSessionCommand) Name() string        { return "new" }
func (c *newSessionCommand) Aliases() []string   { return nil }
func (c *newSessionCommand) Description() string { return "Start a new empty session" }

func (c *newSessionCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	sessionID, err := m.agent.NewSession()
	if err != nil {
		return state, fmt.Errorf("failed to create new session: %w", err)
	}

	slog.Info("Started new session", "session_id", sessionID)

	freshState := AgentState{}
	m.agent.SetState(freshState)

	ch <- graph.NewEvent(NewCmdResponseEvent(fmt.Sprintf("Started new session: %s", sessionID)))

	m.agent.SetCompleted()

	return freshState, nil
}

// branchSessionCommand handles /branch
type branchSessionCommand struct{}

func (c *branchSessionCommand) Name() string      { return "branch" }
func (c *branchSessionCommand) Aliases() []string { return nil }
func (c *branchSessionCommand) Description() string {
	return "Start a new session with current history"
}

func (c *branchSessionCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	freshState := state

	newSessionID, err := m.agent.NewSession()
	if err != nil {
		return state, fmt.Errorf("failed to create branch session: %w", err)
	}

	_ = m.agent.SetState(freshState)

	slog.Info("Branched session", "new_session_id", newSessionID)

	ch <- graph.NewEvent(NewCmdResponseEvent(fmt.Sprintf(
		"Branched to new session: %s",
		newSessionID,
	)))

	m.agent.SetCompleted()
	return freshState, nil
}

// listSessionsCommand handles /list
type listSessionsCommand struct{}

func (c *listSessionsCommand) Name() string        { return "list" }
func (c *listSessionsCommand) Aliases() []string   { return []string{"ls"} }
func (c *listSessionsCommand) Description() string { return "List all sessions" }

func (c *listSessionsCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	sessions := m.agent.SessionStore().ListSessions()

	var sb strings.Builder
	sb.WriteString("Available sessions:\n")
	for _, s := range sessions {
		fmt.Fprintf(&sb, "  - %s\n", s.Id)
	}

	ch <- graph.NewEvent(NewCmdResponseEvent(sb.String()))

	m.agent.SetCompleted()
	return state, nil
}

// exitCommand handles /q, /exit, /quit, /close
type exitCommand struct{}

func (c *exitCommand) Name() string        { return "q" }
func (c *exitCommand) Aliases() []string   { return []string{"exit", "quit", "close"} }
func (c *exitCommand) Description() string { return "Exit the application" }

func (c *exitCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	slog.Info("User requested application exit")
	m.agent.exitRequested = true
	m.agent.SetCompleted()
	return state, nil
}

// switchSessionCommand handles /switch
type switchSessionCommand struct{}

func (c *switchSessionCommand) Name() string        { return "switch" }
func (c *switchSessionCommand) Aliases() []string   { return nil }
func (c *switchSessionCommand) Description() string { return "Switch to the given session ID" }

func (c *switchSessionCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	sessionID := strings.TrimSpace(args)
	if sessionID == "" {
		ch <- graph.NewEvent(NewCmdResponseEvent("Usage: /switch {sessionid}"))
		m.agent.SetCompleted()
		return state, nil
	}

	if err := m.agent.SetSesssion(sessionID); err != nil {
		return state, fmt.Errorf("failed to switch session: %w", err)
	}

	slog.Info("Switched to session", "session_id", sessionID)
	ch <- graph.NewEvent(NewCmdResponseEvent(fmt.Sprintf("Switched to session: %s", sessionID)))
	m.agent.SetCompleted()
	return state, nil
}

// copyCommand handles /copy
type copyCommand struct{}

func (c *copyCommand) Name() string        { return "copy" }
func (c *copyCommand) Aliases() []string   { return nil }
func (c *copyCommand) Description() string { return "Copy the last AI response to clipboard" }

func (c *copyCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	var lastAiText string
	for i := len(state.messages) - 1; i >= 0; i-- {
		if state.messages[i].Role == messages.MessageRoleAI {
			lastAiText = state.messages[i].OutputText()
			break
		}
	}

	if lastAiText == "" {
		ch <- graph.NewEvent(NewCmdResponseEvent("No AI response to copy"))
		m.agent.SetCompleted()
		return state, nil
	}

	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = bytes.NewBufferString(lastAiText)
	if err := cmd.Run(); err != nil {
		return state, fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	slog.Info("Copied last AI response to clipboard")
	ch <- graph.NewEvent(NewCmdResponseEvent("Copied last AI response to clipboard"))
	m.agent.SetCompleted()
	return state, nil
}

// summarizeCommand handles /summarize
type summarizeCommand struct{}

func (c *summarizeCommand) Name() string        { return "summarize" }
func (c *summarizeCommand) Aliases() []string   { return nil }
func (c *summarizeCommand) Description() string { return "Summarize the conversation" }

func (c *summarizeCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	if len(state.messages) > 0 {
		state.messages = state.messages[:len(state.messages)-1]
	}

	if err := m.agent.SummarizeConversation(ctx, &state); err != nil {
		return state, fmt.Errorf("failed to summarize conversation: %w", err)
	}

	slog.Info("Conversation summarized")
	ch <- graph.NewEvent(NewCmdResponseEvent("Conversation summarized"))

	return state, nil
}

// helpCommand handles /help
type helpCommand struct{}

func (c *helpCommand) Name() string      { return "help" }
func (c *helpCommand) Aliases() []string { return []string{"h", "?"} }
func (c *helpCommand) Description() string {
	return "List all available commands with their descriptions"
}

func (c *helpCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	var sb strings.Builder
	sb.WriteString("Available commands:\n\n")

	seen := make(map[string]ISlashCommandHandler)
	for _, handler := range m.handlers {
		seen[handler.Name()] = handler
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		handler := seen[name]
		desc := handler.Description()

		line := fmt.Sprintf("  /%s", name)

		aliases := handler.Aliases()
		if len(aliases) > 0 {
			aliasParts := make([]string, len(aliases))
			for i, alias := range aliases {
				aliasParts[i] = "/" + alias
			}
			line += fmt.Sprintf(" (%s)", strings.Join(aliasParts, ", "))
		}

		line += fmt.Sprintf("\n    %s\n", desc)
		sb.WriteString(line)
	}

	ch <- graph.NewEvent(NewCmdResponseEvent(sb.String()))
	m.agent.SetCompleted()
	return state, nil
}

// effortCommand handles /effort
type effortCommand struct{}

func (c *effortCommand) Name() string      { return "effort" }
func (c *effortCommand) Aliases() []string { return nil }
func (c *effortCommand) Description() string {
	return "Set reasoning effort: /effort {none|low|medium|high} [message]"
}

func (c *effortCommand) Execute(ctx context.Context, args string, m *SlashCommandsMiddleware, ch AgentEventCh, state AgentState) (AgentState, error) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 1 || parts[0] == "" {
		ch <- graph.NewEvent(NewCmdResponseEvent("Usage: /effort {none|low|medium|high} [user message...]"))
		m.agent.SetCompleted()
		return state, nil
	}

	effortLevel := shared.ReasoningEffort(parts[0])

	switch effortLevel {
	case shared.ReasoningEffortNone, shared.ReasoningEffortLow, shared.ReasoningEffortMedium, shared.ReasoningEffortHigh:
	default:
		ch <- graph.NewEvent(NewCmdResponseEvent(fmt.Sprintf("Invalid effort level: %s. Valid values: none, low, medium, high", parts[0])))
		m.agent.SetCompleted()
		return state, nil
	}

	m.agent.GetLlmConfig().ReasoningEffort = effortLevel

	userMessage := ""
	if len(parts) > 1 {
		userMessage = parts[1]
	}

	if len(state.messages) > 0 {
		lastMsg := state.messages[len(state.messages)-1]
		if lastMsg.Role == messages.MessageRoleHuman {
			lastMsg.HumanMessage.Content.OfInputText = userMessage
			state.messages[len(state.messages)-1] = lastMsg
		}
	}

	slog.Info("Set reasoning effort", "effort", effortLevel)
	ch <- graph.NewEvent(NewCmdResponseEvent(fmt.Sprintf("Reasoning effort set to: %s", effortLevel)))

	return state, nil
}
