package styles

import "github.com/charmbracelet/lipgloss"

var (
	// BaseStyle is the default style for all panels.
	BaseStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// HeaderStyle is used for section headers.
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	// ConversationStyle wraps the conversation history panel.
	ConversationStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(0, 1)

	// UserInputStyle wraps the user input panel.
	UserInputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(0, 1)

	// UserInputInterruptStyle is used when the input panel is in interrupt (y/n) mode.
	UserInputInterruptStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("220")).
				Padding(0, 1)

	// SessionListStyle wraps the session list sidebar.
	SessionListStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1)

	// HelpStyle is used for the help bar at the bottom.
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	// ErrorStyle formats error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	// ToolCallStyle formats tool call events.
	ToolCallStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Faint(true)

	// ToolResponseStyle formats tool response events.
	ToolResponseStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	// AIResponseStyle formats AI response messages.
	AIResponseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	// AIReasoningStyle formats AI reasoning content.
	AIReasoningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true)

	// UserMessageStyle formats user messages in the conversation history.
	UserMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true)

	// SystemMessageStyle formats system/command response messages.
	SystemMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51"))

	// DimmedStyle is used for less important text.
	DimmedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	// CursorStyle highlights the currently selected session in the session list.
	CursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	// ActiveSessionStyle highlights the currently active session in the list.
	ActiveSessionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true)

	// SpinnerStyle styles the spinner animation text.
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	// PromptPrefixStyle styles the "> " or interrupt prompt prefix.
	PromptPrefixStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)

	// FocusedBorderStyle is applied to a panel's border when it is focused.
	FocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("39"))

	// UnfocusedBorderStyle is applied to a panel's border when it is not focused.
	UnfocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240"))
)
