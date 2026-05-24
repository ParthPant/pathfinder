package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/prompts"
	"github.com/ParthPant/pathfinder/stores"
	"github.com/ParthPant/pathfinder/tools"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

// --- Styles ---
var (
	userStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	agentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	systemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Align(lipgloss.Center).
			Padding(0, 1)
)

// --- Messages ---

type agentResponseMsg string
type agentErrorMsg error
type agentFinishedMsg struct{}
type agentEventMsg struct {
	event any
}
type agentRunMsg struct {
	ctx   context.Context
	ch    <-chan any
	cherr <-chan error
	send  func(tea.Msg)
}

// --- Model ---

type model struct {
	agent      *agent.Agent
	textInput  textinput.Model
	viewport   viewport.Model
	messages   []string
	width      int
	height     int
	err        error
	loading    bool
	ctx        context.Context
	cancelFunc context.CancelFunc
	send       func(tea.Msg)
}

func newModel(a *agent.Agent) *model {
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()

	ctx, cancel := context.WithCancel(context.Background())

	return &model{
		agent:      a,
		textInput:  ti,
		viewport:   viewport.New(0, 0),
		messages:   []string{systemStyle.Render("Welcome to Pathfinder! Type something to start. Press Ctrl+C to quit.")},
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = m.width - 4
		m.viewport.Height = m.height - 2
		m.textInput.Width = m.width - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelFunc()
			return m, tea.Quit
		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}
			return m.handleEnter()
		}

		// Handle scrolling when text input is not focused
		if !m.textInput.Focused() {
			switch msg.String() {
			case "up", "k":
				m.viewport, cmd = m.viewport.Update(msg)
			case "down", "j":
				m.viewport, cmd = m.viewport.Update(msg)
			case "pageup":
				m.viewport, cmd = m.viewport.Update(msg)
			case "pagedown":
				m.viewport, cmd = m.viewport.Update(msg)
			case "home":
				m.viewport, cmd = m.viewport.Update(msg)
			case "end":
				m.viewport, cmd = m.viewport.Update(msg)
			}
		}

	case agentResponseMsg:
		m.messages = append(m.messages, fmt.Sprintf("%s: %s", agentStyle.Render("Agent"), string(msg)))
		m.loading = false
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		return m, nil

	case agentErrorMsg:
		m.err = msg.(error)
		m.messages = append(m.messages, fmt.Sprintf("%s: %v", errorStyle.Render("Error"), m.err))
		m.loading = false
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		return m, nil

	case agentFinishedMsg:
		slog.Debug("Agent Finished Event received.")
		m.loading = false
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		return m, nil

	case agentEventMsg:
		event := msg.event
		switch e := event.(type) {
		case agent.EventAIResponse:
			content := e.Message.OutputText()
			slog.Debug("Agent Message Event", "message", content)
			if content != "" {
				m.messages = append(m.messages, fmt.Sprintf("%s: %s", agentStyle.Render("Agent"), content))
			}
		case agent.EventToolCall:
			slog.Debug("Agent Tool Call Event", "name", e.Call.Name, "arguments", e.Call.Arguments)
			m.messages = append(m.messages, fmt.Sprintf("%s: Calling tool %s...", systemStyle.Render("System"), e.Call.Name))
		case agent.EventToolResponse:
			content := e.Message.OutputText()
			slog.Debug("Agent Tool Response", "message", content)
			m.messages = append(m.messages, fmt.Sprintf("%s: Tool response: %s", systemStyle.Render("System"), content))
		}
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		return m, nil

	case agentRunMsg:
		m.loading = true
		go agentSupervisor(m.ctx, msg.ch, msg.cherr, m.send)
		return m, nil
	}

	// Handle text input updates
	var tiCmd tea.Cmd
	m.textInput, tiCmd = m.textInput.Update(msg)
	cmd = tiCmd

	// Handle viewport updates
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)

	// Combine commands
	if cmd != nil && vpCmd != nil {
		return m, tea.Batch(cmd, vpCmd)
	} else if cmd != nil {
		return m, cmd
	} else {
		return m, vpCmd
	}
}

func agentSupervisor(ctx context.Context, ch <-chan any, cherr <-chan error, send func(tea.Msg)) {
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-cherr:
			if ok && err != nil {
				send(agentErrorMsg(err))
			}
		case event, ok := <-ch:
			if !ok {
				send(agentFinishedMsg{})
				return
			}
			send(agentEventMsg{event: event})
		}
	}
}

func (m *model) handleEnter() (tea.Model, tea.Cmd) {
	userInput := m.textInput.Value()
	if userInput == "" {
		return m, nil
	}

	m.messages = append(m.messages, fmt.Sprintf("%s: %s", userStyle.Render("User"), userInput))
	m.textInput.SetValue("")
	m.loading = true

	return m, func() tea.Msg {
		a := m.agent
		a.UserInput(messages.NewTextMessage("user", userInput, nil))

		runCtx, _ := context.WithTimeout(m.ctx, 5*60*time.Second)
		ch, cherr := a.Run(runCtx)

		return agentRunMsg{ctx: runCtx, ch: ch, cherr: cherr, send: m.send}
	}
}

func (m *model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("\nError: %v\n", m.err))
	}

	// Header
	header := headerStyle.Width(m.width).Render("Pathfinder TUI")

	// Chat area
	chatArea := strings.Join(m.messages, "\n")

	// Loading indicator
	if m.loading {
		chatArea += "\n" + systemStyle.Render("Agent is thinking...")
	}

	// Wrap chat area to width using viewport
	chatAreaStyle := lipgloss.NewStyle().
		Width(m.width - 4).
		MarginLeft(2).
		MarginRight(2)

	m.viewport.SetContent(chatArea)
	renderedChat := chatAreaStyle.Render(m.viewport.View())

	// Footer (input)
	footer := fmt.Sprintf("\n%s", m.textInput.View())
	footerStyle := lipgloss.NewStyle().Width(m.width).PaddingLeft(2)
	renderedFooter := footerStyle.Render(footer)

	// Join everything
	return lipgloss.JoinVertical(lipgloss.Left, header, renderedChat, renderedFooter)
}

func main() {
	godotenv.Load()

	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	logLevel := slog.Level(getEnvAsInt("LOG_LEVEL"))
	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)

	slog.SetDefault(logger)

	config := llms.LlmConfig{
		BaseUrl:         os.Getenv("OPENROUTER_BASE_URL"),
		APIKey:          os.Getenv("OPENROUTER_API_KEY"),
		Model:           os.Getenv("MODEL"),
		MaxOutputTokens: 25000,
	}

	llm := llms.NewOpenAiLlm(config)
	toolExecutor := tools.NewToolExecutor()
	inMemStore := stores.NewInMemoryStore[agent.AgentState]()

	a := agent.NewAgent(llm, toolExecutor, inMemStore)

	executionBackend := backends.NewShellBackend(os.Getenv("WORK_DIR"), map[string]string{})
	a.RegisterExecutionBackend(executionBackend)

	fsBackend := backends.NewLocalFileSystemBackend(os.Getenv("WORK_DIR"))
	a.RegisterFileSystemBackend(fsBackend)

	a.RegisterFunctionCall(tools.GetDateTimeTool)
	a.RegisterFunctionCall(tools.InternetSearchTool)
	a.RegisterFunctionCall(tools.OpenURLTool)

	memoryMiddleware := agent.NewMemoryMiddleware(prompts.MemoryPrompt, ".pathfinder", fsBackend)
	a.AddMiddleware(memoryMiddleware)

	summaryLlmConfig := llms.LlmConfig{
		BaseUrl:         os.Getenv("OPENROUTER_BASE_URL"),
		APIKey:          os.Getenv("OPENROUTER_API_KEY"),
		Model:           os.Getenv("SUMMARY_MODEL"),
		MaxOutputTokens: 25000,
	}
	summaryLlm := llms.NewOpenAiLlm(summaryLlmConfig)
	summarizeMiddleware := agent.NewSummarizationMiddleware(summaryLlm, prompts.SummarizationPrompt, 90000, 10)
	a.AddMiddleware(summarizeMiddleware)

	_, err = a.StartSession()
	if err != nil {
		panic(err)
	}

	m := newModel(a)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.send = p.Send

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func getEnvAsInt(name string) int {
	str := os.Getenv(name)
	i, _ := strconv.Atoi(str)
	return i
}
