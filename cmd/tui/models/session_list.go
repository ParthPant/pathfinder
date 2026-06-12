package models

import (
	"strings"

	"github.com/ParthPant/pathfinder/cmd/tui/styles"
	"github.com/ParthPant/pathfinder/cmd/tui/types"
	"github.com/ParthPant/pathfinder/stores"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SessionListModel renders a collapsible sidebar listing available sessions.
// When expanded, the user can navigate sessions with up/down arrows and press
// Enter to switch to the selected session.
type SessionListModel struct {
	sessions         []stores.Session
	cursor           int
	currentSessionID string
	width            int
	height           int
	focused          bool
}

// NewSessionListModel creates a new SessionListModel with sensible defaults.
func NewSessionListModel() SessionListModel {
	return SessionListModel{
		sessions: make([]stores.Session, 0),
		cursor:   0,
	}
}

// Init returns nil — no initial commands needed for the session list.
func (m SessionListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the session list panel.
func (m SessionListModel) Update(msg tea.Msg) (SessionListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case types.SessionListUpdateMsg:
		m.sessions = msg.Sessions
		// Clamp cursor to valid range
		if len(m.sessions) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.sessions) {
			m.cursor = len(m.sessions) - 1
		}

	case types.SessionSwitchMsg:
		m.currentSessionID = msg.ID
		m.focused = false

	case types.FocusChangeMsg:
		m.focused = msg.ActivePaneId == ActivePaneSessionList // ActivePaneSessionList

	case tea.KeyMsg:
		if !m.focused {
			break
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
				m.focused = false
				sessionID := m.sessions[m.cursor].Id
				return m, func() tea.Msg {
					return types.SessionSwitchMsg{ID: sessionID}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View renders the session list panel.
// When collapsed: shows a narrow header with a hint to expand.
// When expanded: shows the full session list with cursor navigation.
func (m SessionListModel) View() string {
	collapsedWidth := 14

	if !m.focused {
		// Collapsed: narrow strip with hint
		content := lipgloss.JoinVertical(
			lipgloss.Top,
			styles.HeaderStyle.Render("Sessions ▸"),
			styles.DimmedStyle.Render("Tab"),
			styles.DimmedStyle.Render("expand"),
		)
		return styles.SessionListStyle.Width(collapsedWidth).Render(content)
	}

	// Expanded: full list
	var sb strings.Builder
	sb.WriteString(styles.HeaderStyle.Render("◂ Sessions"))
	sb.WriteString("\n")

	if len(m.sessions) == 0 {
		sb.WriteString(styles.DimmedStyle.Render("  No sessions"))
		sb.WriteString("\n")
	} else {
		for i, s := range m.sessions {
			// Truncate long session IDs for display
			displayID := s.Id
			if len(displayID) > 10 {
				displayID = displayID[:10] + "…"
			}

			var line string
			if s.Id == m.currentSessionID {
				line = styles.ActiveSessionStyle.Render("● " + displayID)
			} else if i == m.cursor {
				line = styles.CursorStyle.Render("▸ " + displayID)
			} else {
				line = styles.DimmedStyle.Render("  " + displayID)
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styles.DimmedStyle.Render("↑/↓ navigate"))
	sb.WriteString("\n")
	sb.WriteString(styles.DimmedStyle.Render("Enter switch"))
	sb.WriteString("\n")
	sb.WriteString(styles.DimmedStyle.Render("Esc collapse"))

	// Apply focused/unfocused border based on focus state
	if m.focused {
		return styles.FocusedBorderStyle.Width(m.width).Render(sb.String())
	}
	return styles.UnfocusedBorderStyle.Width(m.width).Render(sb.String())
}
