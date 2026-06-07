package agent

import (
	"context"
	"testing"

	"github.com/ParthPant/pathfinder/graph"
	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/stores"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestSlashMiddleware creates a SlashCommandsMiddleware with a real (but minimal) Agent
// wired to an InMemoryStore so that session-related commands work.
func newTestSlashMiddleware(t *testing.T) (*SlashCommandsMiddleware, *Agent) {
	t.Helper()

	llmConfig := llms.LlmConfig{}
	llm := llms.NewOpenAiLlm(llmConfig)
	store := stores.NewInMemoryStore[AgentState]()

	agent := &Agent{
		BaseGraph: graph.NewBaseGraph(
			AgentState{},
			map[string]graph.Node[AgentState, AgentEvent, AgentInterrupt]{},
			"beforeAgentNode",
			10,
			store,
		),
		middlewares: nil,
		llm:         llm,
	}

	m := NewSlashCommandsMiddleware()
	err := m.OnAttach(agent)
	require.NoError(t, err)

	return m, agent
}

// agentStateWithMessages creates an AgentState containing the given messages.
func agentStateWithMessages(msgs ...messages.Message) AgentState {
	return AgentState{
		systemMessages:    nil,
		messages:          msgs,
		userRejectedTools: nil,
	}
}

// userMessage is a shorthand helper to build a user message with the given text.
func userMessage(text string) messages.Message {
	return messages.NewTextMessage("user", text, nil)
}

// aiMessage is a shorthand helper to build an AI (assistant) message with the given text.
func aiMessage(text string) messages.Message {
	return messages.Message{
		Role: "assistant",
		AiMessage: messages.AiMessage{
			Output: []messages.OutputItem{
				{
					Type: "message",
					OfMessage: messages.OutputMessage{
						Content: []messages.OutputMessageContent{
							{Type: "output_text", OutputText: text},
						},
					},
				},
			},
		},
	}
}

// collectEventsFromBuffered reads from an already-closed buffered channel.
func collectEventsFromBuffered(ch <-chan graph.RunEvent[AgentEvent]) []graph.RunEvent[AgentEvent] {
	var events []graph.RunEvent[AgentEvent]
	for e := range ch {
		events = append(events, e)
	}
	return events
}

// ---------------------------------------------------------------------------
// NewSlashCommandsMiddleware
// ---------------------------------------------------------------------------

func TestNewSlashCommandsMiddleware_RegistersDefaultCommands(t *testing.T) {
	m := NewSlashCommandsMiddleware()
	require.NotNil(t, m)

	expectedCommands := map[string]bool{
		"new":       true,
		"branch":    true,
		"list":      true,
		"ls":        true,
		"q":         true,
		"exit":      true,
		"quit":      true,
		"close":     true,
		"switch":    true,
		"copy":      true,
		"summarize": true,
		"effort":    true,
		"?":         true,
		"h":         true,
		"help":      true,
	}

	got := make(map[string]bool)
	for name := range m.handlers {
		got[name] = true
	}

	assert.Equal(t, expectedCommands, got, "all default commands and aliases should be registered")
}

func TestNewSlashCommandsMiddleware_CommandsReferenceSameHandler(t *testing.T) {
	m := NewSlashCommandsMiddleware()

	assert.Same(t, m.handlers["q"], m.handlers["exit"], "exit should be alias of q")
	assert.Same(t, m.handlers["q"], m.handlers["quit"], "quit should be alias of q")
	assert.Same(t, m.handlers["q"], m.handlers["close"], "close should be alias of q")
	assert.Same(t, m.handlers["list"], m.handlers["ls"], "ls should be alias of list")
}

// ---------------------------------------------------------------------------
// OnAttach
// ---------------------------------------------------------------------------

func TestOnAttach_StoresAgentReference(t *testing.T) {
	m := NewSlashCommandsMiddleware()
	require.NotNil(t, m)

	agent := &Agent{}
	err := m.OnAttach(agent)
	assert.NoError(t, err)
	assert.Same(t, agent, m.agent)
}

// ---------------------------------------------------------------------------
// BeforeAgent — edge cases (early returns)
// ---------------------------------------------------------------------------

func TestBeforeAgent_NoMessages_ReturnsStateUnchanged(t *testing.T) {
	m, _ := newTestSlashMiddleware(t)
	state := AgentState{}

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.Equal(t, state, newState)
	assert.Empty(t, ch, "no events should be emitted")
}

func TestBeforeAgent_NonUserMessage_ReturnsStateUnchanged(t *testing.T) {
	m, _ := newTestSlashMiddleware(t)
	state := agentStateWithMessages(
		messages.NewTextMessage("system", "/new", nil),
	)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.Equal(t, state, newState)
	assert.Empty(t, ch, "no events should be emitted for non-user messages")
}

func TestBeforeAgent_NoSlashPrefix_ReturnsStateUnchanged(t *testing.T) {
	m, _ := newTestSlashMiddleware(t)
	state := agentStateWithMessages(userMessage("hello world"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.Equal(t, state, newState)
	assert.Empty(t, ch, "no events should be emitted for non-slash messages")
}

func TestBeforeAgent_EmptyMessageText(t *testing.T) {
	m, _ := newTestSlashMiddleware(t)

	msg := messages.Message{
		Role:         "user",
		HumanMessage: messages.HumanMessage{},
	}
	state := agentStateWithMessages(msg)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.Equal(t, state, newState, "empty user message should pass through")
	assert.Empty(t, ch)
}

func TestBeforeAgent_OnlyLastUserMessageChecked(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(
		userMessage("first message"),
		userMessage("/list"),
	)

	_, err := agent.NewSession()
	require.NoError(t, err)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err = m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Available sessions")
}

func TestBeforeAgent_SlashInEarlierMessageIgnored(t *testing.T) {
	m, _ := newTestSlashMiddleware(t)

	state := agentStateWithMessages(
		userMessage("/new"),
		userMessage("hello world"),
	)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.Equal(t, state, newState, "state should be unchanged")
	assert.Empty(t, ch)
}

// ---------------------------------------------------------------------------
// Unknown command
// ---------------------------------------------------------------------------

func TestBeforeAgent_UnknownCommand_EmitsErrorEvent(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)
	state := agentStateWithMessages(userMessage("/foobar"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.Error(t, err)
	assert.Equal(t, state, newState)
	assert.True(t, agent.IsCompleted(), "agent should be completed after unknown command")

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Equal(t, CMDRESP, events[0].Value.Type)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Unknown command")
}

func TestBeforeAgent_OnlySlashWithNoCommandName(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)
	state := agentStateWithMessages(userMessage("/"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.Error(t, err)
	assert.Equal(t, state, newState)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1, "unknown command should emit one event")
	assert.Equal(t, CMDRESP, events[0].Value.Type)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Unknown command")
}

// ---------------------------------------------------------------------------
// /new command
// ---------------------------------------------------------------------------

func TestCommand_New_CreatesNewSession(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	firstID, err := agent.NewSession()
	require.NoError(t, err)
	agent.SetState(agentStateWithMessages())
	require.NotNil(t, agent.GetState())

	state := agentStateWithMessages(userMessage("/new"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted(), "agent should be completed after /new")

	sessions := agent.SessionStore().ListSessions()
	require.Len(t, sessions, 2, "should have 2 sessions now")

	assert.Empty(t, newState.messages)
	assert.Empty(t, newState.systemMessages)
	assert.Empty(t, newState.userRejectedTools)

	assert.NotEqual(t, firstID, sessions[1].Id)

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Equal(t, CMDRESP, events[0].Value.Type)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Started new session")
}

func TestCommand_New_IgnoresArgs(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	_, err := agent.NewSession()
	require.NoError(t, err)

	state := agentStateWithMessages(userMessage("/new some extra args"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err = m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	sessions := agent.SessionStore().ListSessions()
	assert.Len(t, sessions, 2, "/new should create a new session even with args")
}

// ---------------------------------------------------------------------------
// /branch command
// ---------------------------------------------------------------------------

func TestCommand_Branch_CopiesMessagesToNewSession(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	_, err := agent.NewSession()
	require.NoError(t, err)

	msg1 := userMessage("hello")
	msg2 := aiMessage("hi there")
	msg3 := userMessage("/branch")
	state := agentStateWithMessages(msg1, msg2, msg3)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted(), "agent should be completed after /branch")

	assert.Equal(t, state.messages, newState.messages, "messages should be preserved on branch")

	sessions := agent.SessionStore().ListSessions()
	require.Len(t, sessions, 2)

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Equal(t, CMDRESP, events[0].Value.Type)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Branched to new session")
}

func TestCommand_Branch_IgnoresArgs(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	_, err := agent.NewSession()
	require.NoError(t, err)

	state := agentStateWithMessages(
		userMessage("important context"),
		aiMessage("important response"),
		userMessage("/branch with args"),
	)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())
	assert.Len(t, newState.messages, 3)
}

// ---------------------------------------------------------------------------
// /list command
// ---------------------------------------------------------------------------

func TestCommand_List_ListsSessions(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	_, err := agent.NewSession()
	require.NoError(t, err)
	_, err = agent.NewSession()
	require.NoError(t, err)

	state := agentStateWithMessages(userMessage("/list"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err = m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Equal(t, CMDRESP, events[0].Value.Type)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Available sessions")
}

func TestCommand_List_AliasWorks(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	_, err := agent.NewSession()
	require.NoError(t, err)

	state := agentStateWithMessages(userMessage("/ls"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err = m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Equal(t, CMDRESP, events[0].Value.Type)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Available sessions")
}

func TestCommand_List_NoSessions(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(userMessage("/list"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Available sessions")
	assert.NotContains(t, events[0].Value.OfCmdResponse.Message, "  - ")
}

// ---------------------------------------------------------------------------
// /q command (and aliases: exit, quit, close)
// ---------------------------------------------------------------------------

func TestCommand_Q_SetsExitRequested(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(userMessage("/q"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())
	assert.True(t, agent.IsExitRequested(), "exit should be requested after /q")
}

func TestCommand_Q_Aliases(t *testing.T) {
	for _, alias := range []string{"/exit", "/quit", "/close"} {
		t.Run(alias, func(t *testing.T) {
			m, agent := newTestSlashMiddleware(t)

			state := agentStateWithMessages(userMessage(alias))

			ch := make(chan graph.RunEvent[AgentEvent], 10)
			_, err := m.BeforeAgent(context.Background(), ch, nil, state)
			close(ch)

			assert.NoError(t, err)
			assert.True(t, agent.IsCompleted())
			assert.True(t, agent.IsExitRequested(), "exit should be requested after %s", alias)
		})
	}
}

// ---------------------------------------------------------------------------
// /switch command
// ---------------------------------------------------------------------------

func TestCommand_Switch_NoArgs_ShowsUsage(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(userMessage("/switch"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Usage: /switch")
}

func TestCommand_Switch_InvalidSessionID(t *testing.T) {
	m, _ := newTestSlashMiddleware(t)

	state := agentStateWithMessages(userMessage("/switch nonexistent-id"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to switch session")

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "failed to switch session")
}

func TestCommand_Switch_ValidSessionID(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	sessionID, err := agent.NewSession()
	require.NoError(t, err)

	state1 := agentStateWithMessages(userMessage("hello from session 1"))
	agent.SetState(state1)

	_, err = agent.NewSession()
	require.NoError(t, err)

	state := agentStateWithMessages(userMessage("/switch " + sessionID))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err = m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Switched to session")
}

func TestCommand_Switch_PreservesSessionState(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	session1ID, err := agent.NewSession()
	require.NoError(t, err)

	session1State := agentStateWithMessages(
		userMessage("hello from session 1"),
		aiMessage("hi from session 1"),
	)
	agent.SetState(session1State)

	session2ID, err := agent.NewSession()
	require.NoError(t, err)

	session2State := agentStateWithMessages(
		userMessage("hello from session 2"),
		aiMessage("hi from session 2"),
	)
	agent.SessionStore().SaveState(session2ID, session2State)
	agent.SetSesssion(session2ID)

	state := agentStateWithMessages(userMessage("/switch " + session1ID))
	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err = m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	currentState := agent.GetState()
	assert.Equal(t, session1State.messages, currentState.messages,
		"switching back should restore session 1 messages")
}

// ---------------------------------------------------------------------------
// /copy command
// ---------------------------------------------------------------------------

func TestCommand_Copy_NoAiResponse_ShowsMessage(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(userMessage("/copy"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "No AI response to copy")
}

func TestCommand_Copy_FindsLastAssistantMessage(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	toolMsg := messages.Message{
		Role: "tool",
		ToolMessage: messages.ToolMessage{
			Type:   "function_call_output",
			CallId: "call_1",
			Output: `{"result": "some tool result"}`,
		},
	}

	aiMsg := messages.Message{
		Role: "assistant",
		AiMessage: messages.AiMessage{
			Output: []messages.OutputItem{
				{
					Type: "message",
					OfMessage: messages.OutputMessage{
						Content: []messages.OutputMessageContent{
							{Type: "output_text", OutputText: "Based on the tool result..."},
						},
					},
				},
			},
		},
	}

	state := agentStateWithMessages(
		userMessage("run a tool"),
		toolMsg,
		aiMsg,
		userMessage("/copy"),
	)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Copied last AI response to clipboard")
}

// ---------------------------------------------------------------------------
// /summarize command
// ---------------------------------------------------------------------------

func TestCommand_Summarize_RemovesSlashMessage(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	_, err := agent.NewSession()
	require.NoError(t, err)

	state := agentStateWithMessages(
		userMessage("hello"),
		aiMessage("hi"),
		userMessage("/summarize"),
	)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SummarizationMiddleware not found")
	assert.Len(t, newState.messages, 2, "should have removed the /summarize message")
}

func TestCommand_Summarize_NoMessages(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	_, err := agent.NewSession()
	require.NoError(t, err)

	state := agentStateWithMessages(userMessage("/summarize"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.Error(t, err)
	assert.Empty(t, newState.messages)
}

func TestCommand_Summarize_DoesNotComplete(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	_, err := agent.NewSession()
	require.NoError(t, err)

	state := agentStateWithMessages(userMessage("/summarize"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, _ = m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.False(t, agent.IsCompleted(), "/summarize should not complete the graph")
}

// ---------------------------------------------------------------------------
// /effort command
// ---------------------------------------------------------------------------

func TestCommand_Effort_NoArgs_ShowsUsage(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(userMessage("/effort"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Usage: /effort")
}

func TestCommand_Effort_InvalidLevel_ShowsError(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(userMessage("/effort extreme"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Invalid effort level")
}

func TestCommand_Effort_ValidLevel_SetsReasoningEffort(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"none", "/effort none"},
		{"low", "/effort low"},
		{"medium", "/effort medium"},
		{"high", "/effort high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, agent := newTestSlashMiddleware(t)

			state := agentStateWithMessages(userMessage(tt.input))

			ch := make(chan graph.RunEvent[AgentEvent], 10)
			_, err := m.BeforeAgent(context.Background(), ch, nil, state)
			close(ch)

			assert.NoError(t, err)
			assert.False(t, agent.IsCompleted(),
				"agent should NOT be completed after /effort — let graph continue")

			events := collectEventsFromBuffered(ch)
			require.Len(t, events, 1)
			assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Reasoning effort set to")
		})
	}
}

func TestCommand_Effort_WithUserMessage_StripsEffortPrefix(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(
		userMessage("/effort high tell me something interesting"),
	)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.False(t, agent.IsCompleted(), "agent should NOT be completed after /effort")

	require.Len(t, newState.messages, 1)
	assert.Equal(t, "tell me something interesting", newState.messages[0].GetTextContent())

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Reasoning effort set to")
}

func TestCommand_Effort_WithNoExtraMessage(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state := agentStateWithMessages(
		userMessage("/effort high"),
	)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.False(t, agent.IsCompleted())

	require.Len(t, newState.messages, 1)
	assert.Equal(t, "", newState.messages[0].GetTextContent())
}

func TestCommand_Effort_PreservesNonHumanMessage(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	sysMsg := messages.NewTextMessage("system", "/effort high do something", nil)
	state := agentStateWithMessages(sysMsg)

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.NoError(t, err)
	assert.False(t, agent.IsCompleted())

	require.Len(t, newState.messages, 1)
	assert.Equal(t, "/effort high do something", newState.messages[0].GetTextContent())
}

// ---------------------------------------------------------------------------
// Passthrough methods
// ---------------------------------------------------------------------------

func TestSlashCommands_PassthroughMethods_ReturnStateUnchanged(t *testing.T) {
	m, _ := newTestSlashMiddleware(t)

	ctx := context.Background()
	state := agentStateWithMessages(userMessage("hello"))

	state1, err := m.AfterAgent(ctx, nil, nil, state)
	assert.NoError(t, err)
	assert.Equal(t, state, state1)

	state2, err := m.BeforeLlm(ctx, nil, nil, state)
	assert.NoError(t, err)
	assert.Equal(t, state, state2)

	state3, err := m.AfterLlm(ctx, nil, nil, state)
	assert.NoError(t, err)
	assert.Equal(t, state, state3)
}

// ---------------------------------------------------------------------------
// Contract: commands have names and descriptions
// ---------------------------------------------------------------------------

func TestAllCommands_HaveNonEmptyDescriptions(t *testing.T) {
	m := NewSlashCommandsMiddleware()

	for name, handler := range m.handlers {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, handler.Description(),
				"command %q should have a non-empty description", name)
		})
	}
}

func TestAllCommands_HaveNonEmptyName(t *testing.T) {
	m := NewSlashCommandsMiddleware()

	seen := make(map[ISlashCommandHandler]bool)
	for _, handler := range m.handlers {
		if seen[handler] {
			continue
		}
		seen[handler] = true
		assert.NotEmpty(t, handler.Name(), "handler should have a non-empty Name()")
	}
}

func TestCommandHandlers_NameMatchesRegistration(t *testing.T) {
	m := NewSlashCommandsMiddleware()

	handlerNames := []string{"new", "branch", "list", "q", "switch", "copy", "summarize", "effort"}
	for _, name := range handlerNames {
		t.Run(name, func(t *testing.T) {
			handler, ok := m.handlers[name]
			require.True(t, ok, "handler %q should be registered", name)
			assert.Equal(t, name, handler.Name(),
				"handler registered as %q should have Name() == %q", name, name)
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: full command dispatch
// ---------------------------------------------------------------------------

func TestBeforeAgent_DispatchesAllKnownCommands(t *testing.T) {
	commands := []struct {
		name    string
		input   string
		expects func(t *testing.T, m *SlashCommandsMiddleware, agent *Agent, events []graph.RunEvent[AgentEvent])
	}{
		{
			name:  "/new",
			input: "/new",
			expects: func(t *testing.T, m *SlashCommandsMiddleware, agent *Agent, events []graph.RunEvent[AgentEvent]) {
				assert.True(t, agent.IsCompleted())
				require.Len(t, events, 1)
				assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Started new session")
			},
		},
		{
			name:  "/q",
			input: "/q",
			expects: func(t *testing.T, m *SlashCommandsMiddleware, agent *Agent, events []graph.RunEvent[AgentEvent]) {
				assert.True(t, agent.IsCompleted())
				assert.True(t, agent.IsExitRequested())
			},
		},
		{
			name:  "/list",
			input: "/list",
			expects: func(t *testing.T, m *SlashCommandsMiddleware, agent *Agent, events []graph.RunEvent[AgentEvent]) {
				assert.True(t, agent.IsCompleted())
				require.Len(t, events, 1)
				assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Available sessions")
			},
		},
		{
			name:  "/effort (no complete)",
			input: "/effort high",
			expects: func(t *testing.T, m *SlashCommandsMiddleware, agent *Agent, events []graph.RunEvent[AgentEvent]) {
				assert.False(t, agent.IsCompleted(), "/effort should not complete the graph")
			},
		},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			m, agent := newTestSlashMiddleware(t)

			_, err := agent.NewSession()
			require.NoError(t, err)

			state := agentStateWithMessages(userMessage(cmd.input))

			ch := make(chan graph.RunEvent[AgentEvent], 10)
			_, err = m.BeforeAgent(context.Background(), ch, nil, state)
			close(ch)
			assert.NoError(t, err)

			events := collectEventsFromBuffered(ch)
			cmd.expects(t, m, agent, events)
		})
	}
}

// ---------------------------------------------------------------------------
// Commands that do NOT complete the agent
// ---------------------------------------------------------------------------

func TestCommandsThatDoNotComplete(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"/summarize", "/summarize"},
		{"/effort medium", "/effort medium"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, agent := newTestSlashMiddleware(t)

			_, err := agent.NewSession()
			require.NoError(t, err)

			state := agentStateWithMessages(userMessage(tt.input))

			ch := make(chan graph.RunEvent[AgentEvent], 10)
			_, _ = m.BeforeAgent(context.Background(), ch, nil, state)
			close(ch)

			assert.False(t, agent.IsCompleted(),
				"%s should not set agent as completed", tt.name)
		})
	}
}

// ---------------------------------------------------------------------------
// Error: command returns error, middleware should propagate it
// ---------------------------------------------------------------------------

func TestBeforeAgent_CommandReturnsError_EmitsErrorEvent(t *testing.T) {
	m, _ := newTestSlashMiddleware(t)

	state := agentStateWithMessages(userMessage("/switch nonexistent-session"))

	ch := make(chan graph.RunEvent[AgentEvent], 10)
	newState, err := m.BeforeAgent(context.Background(), ch, nil, state)
	close(ch)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to switch session")
	assert.Equal(t, state, newState)

	events := collectEventsFromBuffered(ch)
	require.Len(t, events, 1)
	assert.Equal(t, CMDRESP, events[0].Value.Type)
}

// ---------------------------------------------------------------------------
// Benchmark: parsing and dispatching commands
// ---------------------------------------------------------------------------

func BenchmarkBeforeAgent_KnownCommand(b *testing.B) {
	m, agent := newTestSlashMiddleware(&testing.T{})

	_, err := agent.NewSession()
	require.NoError(b, err)

	state := agentStateWithMessages(userMessage("/list"))

	b.ResetTimer()
	for b.Loop() {
		ch := make(chan graph.RunEvent[AgentEvent], 10)
		_, _ = m.BeforeAgent(context.Background(), ch, nil, state)
		close(ch)
	}
}

func BenchmarkBeforeAgent_UnknownCommand(b *testing.B) {
	m, _ := newTestSlashMiddleware(&testing.T{})

	state := agentStateWithMessages(userMessage("/doesnotexist"))

	b.ResetTimer()
	for b.Loop() {
		ch := make(chan graph.RunEvent[AgentEvent], 10)
		_, _ = m.BeforeAgent(context.Background(), ch, nil, state)
		close(ch)
	}
}

// ---------------------------------------------------------------------------
// E2E: new-then-list workflow
// ---------------------------------------------------------------------------

func TestCommand_E2E_NewThenList(t *testing.T) {
	m, agent := newTestSlashMiddleware(t)

	state1 := agentStateWithMessages(userMessage("/new"))
	ch1 := make(chan graph.RunEvent[AgentEvent], 10)
	_, err := m.BeforeAgent(context.Background(), ch1, nil, state1)
	close(ch1)
	assert.NoError(t, err)
	assert.True(t, agent.IsCompleted())

	sessions := agent.SessionStore().ListSessions()
	require.Len(t, sessions, 1, "should have exactly 1 session after /new")

	m2, agent2 := newTestSlashMiddleware(t)
	agent2.SessionStore().SaveState(sessions[0].Id, agent.GetState())

	state2 := agentStateWithMessages(userMessage("/list"))
	ch2 := make(chan graph.RunEvent[AgentEvent], 10)
	_, err = m2.BeforeAgent(context.Background(), ch2, nil, state2)
	close(ch2)

	assert.NoError(t, err)
	assert.True(t, agent2.IsCompleted())

	events := collectEventsFromBuffered(ch2)
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Value.OfCmdResponse.Message, "Available sessions")
}
