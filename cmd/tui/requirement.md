# Pathfinder TUI
The tui should provide a clean terminal based user interface to interact with the pathfinder agent.

## TUI components
- **Conversation History**: Displays the current conversation between the user and the agent in the current session. Also indicates tool calls, events, interrupts etc... Should allow for scrolling
- **User Input**: Place where the user inputs their prompt or command for the agent or react to interrupts raised by the agent.
- **Session List**: collapsible pane that allows user to switch between sessions.

# Guidance
Use the bubbletea module with lipgloss for stype/layout and bubbless for components

DO NOT put the entire state of the application in a single model struct. Ideally each panel should have it's own model and update loop handling.
The implementation should be modular enough that it's easy to add new panes in the future.
