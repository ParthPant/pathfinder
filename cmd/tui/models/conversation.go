package models

import (
	"fmt"
	"strings"

	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/cmd/tui/styles"
	"github.com/ParthPant/pathfinder/cmd/tui/types"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

// ConversationModel renders the full conversation history with scrolling support.
type ConversationModel struct {
	entries      []types.ConversationEvent
	viewport     viewport.Model
	spinner      spinner.Model
	isProcessing bool
	ready        bool
	width        int
	height       int
	focused      bool
}

// NewConversationModel creates a new ConversationModel with default settings.
func NewConversationModel() ConversationModel {
	s := spinner.New()
	s.Style = styles.SpinnerStyle
	s.Spinner = spinner.Dot

	vp := viewport.New(80, 20)
	vp.Style = styles.ConversationStyle

	return ConversationModel{
		entries:  make([]types.ConversationEvent, 0),
		viewport: vp,
		spinner:  s,
	}
}

// Init returns the spinner tick command.
func (m ConversationModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick)
}

// Update handles messages for the conversation panel.
func (m ConversationModel) Update(msg tea.Msg) (ConversationModel, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case types.ConversationEvent:
		m.entries = append(m.entries, msg)
		m.viewport.SetContent(m.buildContent())
		m.viewport.GotoBottom()

	case types.ProcessingStartedMsg:
		m.isProcessing = true

	case types.ProcessingFinishedMsg:
		m.isProcessing = false

	case types.FocusChangeMsg:
		m.focused = msg.ActivePaneId == ActivePaneConversation

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(m.width, m.height)
			m.viewport.Style = styles.ConversationStyle
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = m.height
		}
		// Rebuild content after resize to ensure proper wrapping
		m.viewport.SetContent(m.buildContent())

	default:
		// Handle spinner ticks, viewport key messages, etc.
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	// Always advance the spinner
	var spCmd tea.Cmd
	m.spinner, spCmd = m.spinner.Update(msg)
	cmds = append(cmds, spCmd)

	return m, tea.Batch(cmds...)
}

// View renders the conversation panel.
func (m ConversationModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.focused {
		m.viewport.Style = styles.FocusedBorderStyle
	}
	content := m.viewport.View()

	// Add spinner at the bottom when processing
	if m.isProcessing {
		spinnerLine := styles.SpinnerStyle.Render(m.spinner.View() + " Agent is thinking...")
		content = lipgloss.JoinVertical(lipgloss.Top, content, spinnerLine)
	}

	return content
}

// buildContent renders all entries as formatted strings for the viewport.
func (m ConversationModel) buildContent() string {
	if len(m.entries) == 0 {
		return styles.DimmedStyle.Render("No messages yet. Start a conversation!")
	}

	var sb strings.Builder
	for i, entry := range m.entries {
		if i > 0 {
			sb.WriteString("\n")
		}
		var s string
		if entry.AgentEvent != nil {
			s = renderEvent(entry.AgentEvent)
		} else if entry.UserInput != nil {
			s = styles.UserMessageStyle.Render("👨 " + entry.UserInput.InputText())
		}
		s = wrap.String(s, m.viewport.Width-4)
		sb.WriteString(s)
	}
	return sb.String()
}

// renderEvent formats a single agent event as a styled string.
func renderEvent(e *agent.AgentEvent) string {
	switch e.Type {
	case agent.AIRESP:
		msg := e.OfAiResponse.Message
		var parts []string

		reasoning := msg.ReasoningContent()
		if reasoning != "" {
			parts = append(parts, styles.AIReasoningStyle.Render("Reasoning: "+reasoning))
		}

		output := msg.OutputText()
		if output != "" {
			parts = append(parts, styles.AIResponseStyle.Render(output))
		}

		if len(parts) == 0 {
			return styles.AIResponseStyle.Render("[AI response with no text]")
		}
		return lipgloss.JoinVertical(lipgloss.Left, parts...)

	case agent.TOOLCALL:
		call := e.OfToolCall.Call
		header := fmt.Sprintf("🔧 Tool Call: %s", call.Name)
		args := fmt.Sprintf("    Args: %s", truncateString(call.Arguments, 200))
		return lipgloss.JoinVertical(
			lipgloss.Left,
			styles.ToolCallStyle.Render(header),
			styles.ToolCallStyle.Render(args),
		)

	case agent.TOOLRESP:
		return styles.ToolResponseStyle.Render("📎 Tool Response Sent")

	case agent.AGENTERR:
		return styles.ErrorStyle.Render("❌ Error: " + e.OfError.Err.Error())

	case agent.CMDRESP:
		return styles.SystemMessageStyle.Render("ℹ️ " + e.OfCmdResponse.Message)
	default:
		return styles.DimmedStyle.Render(fmt.Sprintf("[unknown event: %s]", e.Type))
	}
}

// truncateString truncates a string to maxLen chars, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
