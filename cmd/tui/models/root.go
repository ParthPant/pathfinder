package models

import (
	"context"
	"time"

	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/cmd/tui/styles"
	"github.com/ParthPant/pathfinder/cmd/tui/types"
	"github.com/ParthPant/pathfinder/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	ActivePaneConversation = iota // 0
	ActivePaneInput               // 1
	ActivePaneSessionList         // 2
)

// RootModel is the top-level TUI model that owns all three panels
// (conversation, input, session list) and the shared agent reference.
// It acts as the central message dispatcher and layout manager.
type RootModel struct {
	conversation ConversationModel
	input        InputModel
	sessionList  SessionListModel

	ActivePaneId int

	agent  *agent.Agent
	ctx    context.Context
	cancel context.CancelFunc

	// program is set after tea.NewProgram() so the agent runner goroutine
	// can send messages back into the event loop.
	program *tea.Program

	width  int
	height int
	ready  bool
}

// NewRootModel creates a RootModel with the given agent and all sub-models initialized.
func NewRootModel(ag *agent.Agent) *RootModel {
	return &RootModel{
		conversation: NewConversationModel(),
		input:        NewInputModel(),
		sessionList:  NewSessionListModel(),
		agent:        ag,
	}
}

// SetProgram stores the tea.Program reference so the agent runner goroutine
// can send messages back to the event loop. Must be called before Run().
func (m *RootModel) SetProgram(p *tea.Program) {
	m.program = p
}

// Init initializes the root model. It starts the agent's initial session if needed,
// creates a cancellable context, fetches the session list, and returns the input
// blink command batched together.
func (m *RootModel) Init() tea.Cmd {
	// Input starts focused
	m.ActivePaneId = ActivePaneInput

	_, err := m.agent.NewSession()
	if err != nil {
		return func() tea.Msg {
			return types.ConversationEvent{
				AgentEvent: agent.NewErrorEvent(err),
			}
		}
	}

	// Set up a cancellable context for agent runs (15 min timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	m.ctx = ctx
	m.cancel = cancel

	// Get the current session ID for the session list
	sessionID := ""
	if m.agent.SessionStore() != nil {
		sessions := m.agent.SessionStore().ListSessions()
		if len(sessions) > 0 {
			sessionID = sessions[len(sessions)-1].Id
		}
	}

	if m.program == nil {
		panic("Root model must contain a reference to the program.")
	}

	return tea.Batch(
		// Fetch the initial session list
		fetchSessionListCmd(m.agent),
		// Start the text input blink
		m.input.Init(),
		// Start the conversation spinner
		m.conversation.Init(),
		// Set current session ID after init
		func() tea.Msg {
			return types.SessionSwitchMsg{ID: sessionID}
		},
	)
}

// Update is the central message dispatcher. It handles all top-level messages
// and forwards relevant messages to the appropriate sub-models.
func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	// ─── User submitted input ──────────────────────────────────────────────
	case types.SendUserInputMsg:
		if msg.Text == "" {
			return m, nil
		}
		// Send the user message to the agent
		userInput := messages.NewTextMessage("user", msg.Text, nil)
		m.agent.UserInput(userInput)

		// Also add the user message to the conversation history via ConversationEvent
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(types.ConversationEvent{
			UserInput: &userInput,
		})
		cmds = append(cmds, cmd)

		// Re-create context with timeout for this run
		if m.cancel != nil {
			m.cancel()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		m.ctx = ctx
		m.cancel = cancel

		// Run the agent in a goroutine, piping events back to the program
		if m.program != nil {
			go runAgentSession(m.ctx, m.agent, m.program)
		}

	// ─── Conversation event (agent event or user input) ───────────────────
	case types.ConversationEvent:
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		cmds = append(cmds, cmd)

	// ─── Agent interrupt (e.g., tool call approval) ───────────────────────
	case types.AgentInterruptMsg:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	// ─── Session switch ────────────────────────────────────────────────────
	case types.SessionSwitchMsg:
		// Set the new session on the agent
		if err := m.agent.SetSesssion(msg.ID); err != nil {
			evt := agent.NewErrorEvent(err)
			var cmd tea.Cmd
			m.conversation, cmd = m.conversation.Update(types.ConversationEvent{
				AgentEvent: evt,
			})
			cmds = append(cmds, cmd)
		} else {
			// Clear the conversation history for the new session
			m.conversation.entries = make([]types.ConversationEvent, 0)
			m.conversation.viewport.SetContent(m.conversation.View())
		}
		var cmd tea.Cmd
		m.sessionList, cmd = m.sessionList.Update(msg)
		cmds = append(cmds, cmd)

	// ─── Session list update (new data from store) ────────────────────────
	case types.SessionListUpdateMsg:
		var cmd tea.Cmd
		m.sessionList, cmd = m.sessionList.Update(msg)
		cmds = append(cmds, cmd)

	// ─── Processing started / finished ─────────────────────────────────────
	case types.ProcessingStartedMsg:
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		m.input, _ = m.input.Update(msg)
		m.sessionList, _ = m.sessionList.Update(msg)
		cmds = append(cmds, cmd)

	case types.ProcessingFinishedMsg:
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		m.input, _ = m.input.Update(msg)
		m.sessionList, _ = m.sessionList.Update(msg)
		cmds = append(cmds, cmd)

		// Refresh the session list (agent may have created new sessions)
		cmds = append(cmds, fetchSessionListCmd(m.agent))

		// After processing, check if exit was requested
		if m.agent.IsExitRequested() {
			return m, tea.Quit
		}

	// ─── Focus change ─────────────────────────────────────────────────────
	case types.FocusChangeMsg:
		// Forward focus change to all sub-models
		var convCmd, inputCmd, slCmd tea.Cmd
		m.conversation, convCmd = m.conversation.Update(msg)
		m.input, inputCmd = m.input.Update(msg)
		m.sessionList, slCmd = m.sessionList.Update(msg)
		cmds = append(cmds, convCmd, inputCmd, slCmd)

	// ─── Exit TUI ─────────────────────────────────────────────────────────
	case types.ExitTUIMsg:
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit

	// ─── Window resize ─────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Layout: session list takes 20% width when expanded, else collapsed width
		sessionListWidth := 14 // collapsed width
		if m.sessionList.focused {
			sessionListWidth = msg.Width * 20 / 100
			if sessionListWidth < 20 {
				sessionListWidth = 20
			}
		}

		mainWidth := msg.Width - sessionListWidth - 2 // -2 for borders
		conversationHeight := msg.Height * 70 / 100
		inputHeight := msg.Height * 30 / 100
		if inputHeight < 5 {
			inputHeight = 5
		}

		// Forward resized dimensions to sub-models
		var cmd tea.Cmd

		// Update session list size
		m.sessionList, cmd = m.sessionList.Update(tea.WindowSizeMsg{
			Width:  sessionListWidth,
			Height: msg.Height,
		})
		cmds = append(cmds, cmd)

		// Update conversation size
		m.conversation, cmd = m.conversation.Update(tea.WindowSizeMsg{
			Width:  mainWidth,
			Height: conversationHeight,
		})
		cmds = append(cmds, cmd)

		// Update input size
		m.input, cmd = m.input.Update(tea.WindowSizeMsg{
			Width:  mainWidth,
			Height: inputHeight,
		})
		cmds = append(cmds, cmd)

	// ─── Global keyboard shortcuts ────────────────────────────────────────
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit

		case "tab":
			// Cycle focus: Conversation → Input → SessionList → Conversation
			m.ActivePaneId = (m.ActivePaneId + 1) % 3
			// Notify sub-models of focus change
			focusMsg := types.FocusChangeMsg{ActivePaneId: m.ActivePaneId}
			var convCmd, inputCmd, slCmd tea.Cmd
			m.conversation, convCmd = m.conversation.Update(focusMsg)
			m.input, inputCmd = m.input.Update(focusMsg)
			m.sessionList, slCmd = m.sessionList.Update(focusMsg)
			cmds = append(cmds, convCmd, inputCmd, slCmd)

		default:
			// Forward key events to the currently focused pane only
			var cmd tea.Cmd
			switch m.ActivePaneId {
			case ActivePaneConversation:
				m.conversation, cmd = m.conversation.Update(msg)
			case ActivePaneInput:
				m.input, cmd = m.input.Update(msg)
			case ActivePaneSessionList:
				m.sessionList, cmd = m.sessionList.Update(msg)
			}
			cmds = append(cmds, cmd)
		}

	default:
		// Forward all other messages to sub-models so they can handle
		// spinner ticks, viewport scrolling, etc.
		var convCmd, inputCmd, slCmd tea.Cmd
		m.conversation, convCmd = m.conversation.Update(msg)
		m.input, inputCmd = m.input.Update(msg)
		m.sessionList, slCmd = m.sessionList.Update(msg)
		cmds = append(cmds, convCmd, inputCmd, slCmd)
	}

	return m, tea.Batch(cmds...)
}

// View lays out the three panels using lipgloss.
// Layout: [Session List (left)] [Conversation + Input (right)]
func (m *RootModel) View() string {
	if !m.ready {
		return "\n  Initializing Pathfinder TUI..."
	}

	// Session list (left sidebar)
	sessionListView := m.sessionList.View()

	// Main area (right): conversation on top, input on bottom
	mainArea := lipgloss.JoinVertical(
		lipgloss.Top,
		m.conversation.View(),
		m.input.View(),
	)

	// Help bar at the bottom
	helpBar := styles.HelpStyle.Render("Ctrl+Q: Quit | Tab: Cycle Focus | Enter: Send")

	// Join horizontally: session list + main area
	ui := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sessionListView,
		mainArea,
	)

	// Add help bar at the bottom
	full := lipgloss.JoinVertical(
		lipgloss.Top,
		ui,
		helpBar,
	)

	return full
}
