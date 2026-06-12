package models

import (
	"fmt"

	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/cmd/tui/styles"
	"github.com/ParthPant/pathfinder/cmd/tui/types"
	"github.com/ParthPant/pathfinder/graph"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputModel provides the user input panel for typing prompts and responding to interrupts.
type InputModel struct {
	// textArea             textinput.Model
	textArea              textarea.Model
	width                 int
	isWaitingForInterrupt bool
	interruptMsg          *agent.AgentInterrupt
	interruptRespCh       chan graph.ICommand[agent.AgentState, agent.AgentEvent, agent.AgentInterrupt]
	focused               bool
}

// NewInputModel creates a new InputModel with a default text input.
func NewInputModel() InputModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Prompt = ""
	ta.CharLimit = 0
	// ta.Width = 0
	ta.ShowLineNumbers = false
	ta.Focus()

	return InputModel{
		textArea: ta,
	}
}

// Init returns the textArea's blink command.
func (m InputModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the input panel.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.isWaitingForInterrupt {
				return m.handleInterruptSubmit()
			}
			// Normal submission
			text := m.textArea.Value()
			if text == "" {
				return m, nil
			}
			m.textArea.SetValue("")
			return m, func() tea.Msg {
				return types.SendUserInputMsg{Text: text}
			}

		case "ctrl+c":
			// Let the root model handle quit globally
			return m, nil

		default:
			if !m.isWaitingForInterrupt {
				m.textArea, cmd = m.textArea.Update(msg)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.textArea.SetWidth(msg.Width - 4)

	case types.AgentInterruptMsg:
		// Store the interrupt for display and the response channel
		m.isWaitingForInterrupt = true
		m.interruptMsg = msg.Interrupt.Value
		m.interruptRespCh = msg.Interrupt.Resp
		m.textArea.SetValue("")
		m.textArea.Placeholder = "Approve? (y/n)"

	case types.ProcessingStartedMsg:
		m.textArea.Blur()
		m.textArea.Placeholder = "Agent is processing..."
		m.textArea.SetValue("")

	case types.ProcessingFinishedMsg:
		m.isWaitingForInterrupt = false
		m.interruptMsg = nil
		m.interruptRespCh = nil
		m.textArea.Focus()
		m.textArea.Placeholder = "Type your message here..."

	case types.FocusChangeMsg:
		m.focused = msg.ActivePaneId == 1 // ActivePaneInput

	case cursor.BlinkMsg:
		m.textArea, cmd = m.textArea.Update(msg)
	}

	return m, cmd
}

// View renders the input panel.
func (m InputModel) View() string {
	if m.isWaitingForInterrupt {
		promptText := renderInterruptPrompt(m.interruptMsg)
		inputView := m.textArea.View()
		return styles.UserInputInterruptStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				styles.PromptPrefixStyle.Render(promptText),
				inputView,
			),
		)
	}

	// Apply focused/unfocused border based on focus state
	content := styles.PromptPrefixStyle.Render("> ") + m.textArea.View()
	if m.focused {
		return styles.FocusedBorderStyle.Render(content)
	}
	return styles.UnfocusedBorderStyle.Render(content)
}

// handleInterruptSubmit handles the Enter key when in interrupt mode.
func (m InputModel) handleInterruptSubmit() (InputModel, tea.Cmd) {
	text := m.textArea.Value()
	m.textArea.SetValue("")
	m.textArea.Placeholder = "Type your message here..."

	switch text {
	case "y", "Y":
		if m.interruptRespCh != nil {
			go func() {
				m.interruptRespCh <- graph.NoOpCommand[agent.AgentState, agent.AgentEvent, agent.AgentInterrupt]()
			}()
		}
	case "n", "N":
		if m.interruptRespCh != nil && m.interruptMsg != nil {
			toolName := m.interruptMsg.OfToolCall.Call.Name
			go func() {
				m.interruptRespCh <- agent.RejectToolCommand(toolName)
			}()
		}
	default:
		// Invalid response, keep waiting
		m.textArea.SetValue("")
		m.textArea.Placeholder = "Please enter y or n..."
		m.isWaitingForInterrupt = true
		return m, nil
	}

	m.isWaitingForInterrupt = false
	m.interruptMsg = nil
	m.interruptRespCh = nil

	return m, nil
}

// renderInterruptPrompt creates a user-facing prompt string for the pending interrupt.
func renderInterruptPrompt(interrupt *agent.AgentInterrupt) string {
	if interrupt == nil {
		return "Interrupt (y/n): "
	}

	switch interrupt.Type {
	case agent.INTR_TOOLCALL:
		call := interrupt.OfToolCall.Call
		return fmt.Sprintf("Allow tool call '%s' with args: %s? (y/n): ", call.Name, call.Arguments)
	default:
		return "Interrupt (y/n): "
	}
}
