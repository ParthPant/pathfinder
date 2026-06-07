package agent

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/graph"
	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/stores"
	"github.com/ParthPant/pathfinder/tools"
)

type AgentState struct {
	systemMessages []messages.Message
	messages       []messages.Message

	userRejectedTools map[string]struct{}
}

type Agent struct {
	graph.BaseGraph[AgentState, AgentEvent, AgentInterrupt]
	llm              llms.IToolCallingLlm
	toolExecutor     tools.IToolExecutor
	executionBackend backends.IExecutionBackend
	fsBackend        backends.IFileSystemBackend
	middlewares      []IMiddleware[AgentState]
	exitRequested    bool
}

func (agent *Agent) IsExitRequested() bool {
	return agent.exitRequested
}

type AgentEventCh = chan<- graph.RunEvent[AgentEvent]
type AgentIntrCh = chan<- graph.RunInterrupt[AgentState, AgentEvent, AgentInterrupt]
type AgentCmd = graph.ICommand[AgentState, AgentEvent, AgentInterrupt]

func NewAgent(llm llms.IToolCallingLlm, toolExecutor tools.IToolExecutor, sessionRepo stores.IStore[AgentState]) *Agent {
	var agent Agent

	nodes := map[string]graph.Node[AgentState, AgentEvent, AgentInterrupt]{
		"beforeAgentNode": agent.beforeAgentNode,
		"beforeLlmNode":   agent.beforeLlmNode,
		"llmNode":         agent.llmNode,
		"afterLlmNode":    agent.afterLlmNode,
		"toolNode":        agent.toolNode,
		"afterAgentNode":  agent.afterAgentNode,
	}

	base := graph.NewBaseGraph(AgentState{}, nodes, "beforeAgentNode", 500, sessionRepo)

	agent.BaseGraph = base
	agent.llm = llm
	agent.toolExecutor = toolExecutor

	// Auto-register CommandsMiddleware as the first middleware
	cmdsMiddleware := NewSlashCommandsMiddleware()
	agent.middlewares = append([]IMiddleware[AgentState]{cmdsMiddleware}, agent.middlewares...)
	cmdsMiddleware.OnAttach(&agent)

	return &agent
}

func (agent *Agent) UserInput(message messages.Message) error {
	state := agent.GetState()
	state.messages = append(state.messages, message)
	agent.SetState(state)
	return nil
}

func (agent *Agent) RegisterFunctionCall(fd tools.FunctionDefinition) error {
	if err := agent.llm.RegisterFunctionDefinition(fd); err != nil {
		return err
	}
	if err := agent.toolExecutor.RegisterFunction(fd); err != nil {
		return err
	}
	return nil
}

func (agent *Agent) RegisterExecutionBackend(e backends.IExecutionBackend) error {
	agent.executionBackend = e

	fd, err := backends.ExecuteToolDefinition(e)
	if err != nil {
		slog.Error("Error while creating function definition", "error", err)
		return err
	}
	if err = agent.RegisterFunctionCall(fd); err != nil {
		slog.Error("Error while registering funciton definition", "error", err)
		return err
	}
	slog.Debug("Registered Execution Backend.")
	return nil
}

func (agent *Agent) RegisterFileSystemBackend(fs backends.IFileSystemBackend) error {
	agent.fsBackend = fs

	toolDefinitions := []func(backends.IFileSystemBackend) (tools.FunctionDefinition, error){
		backends.LsToolDefinition,
		backends.ReadToolDefinition,
		backends.GrepToolDefinition,
		backends.WriteToolDefinition,
		backends.GlobToolDefinition,
		backends.EditToolDefinition,
	}

	for _, toolDef := range toolDefinitions {
		fd, err := toolDef(fs)
		if err != nil {
			return err
		}
		agent.RegisterFunctionCall(fd)
	}

	state := agent.GetState()
	state.systemMessages = append(state.systemMessages, messages.NewTextMessage(
		"system",
		fmt.Sprintf("Project's current working directory: %s", fs.GetRoot()),
		nil),
	)

	slog.Debug("Registered File System Backend.")
	return nil
}

func (agent *Agent) AddMiddleware(m IMiddleware[AgentState]) error {
	agent.middlewares = append(agent.middlewares, m)
	if err := m.OnAttach(agent); err != nil {
		return err
	}
	slog.Info("Attached middleware", "middleware", reflect.TypeOf(m).Elem().Name())
	return nil
}

// SummarizeConversation finds the SummarizationMiddleware and uses it to summarize the conversation.
func (agent *Agent) SummarizeConversation(ctx context.Context, state *AgentState) error {
	for _, mware := range agent.middlewares {
		if sm, ok := mware.(*SummarizationMiddleware); ok {
			result, err := sm.Summarize(ctx, state.messages)
			if err != nil {
				return err
			}
			state.messages = result
			return nil
		}
	}
	return fmt.Errorf("SummarizationMiddleware not found")
}

func (agent *Agent) GetModel() llms.IToolCallingLlm {
	return agent.llm
}

func (agent *Agent) GetTools() []tools.FunctionDefinition {
	return agent.toolExecutor.GetTools()
}

// GetLlmConfig returns a pointer to the agent's LLM configuration.
func (agent *Agent) GetLlmConfig() *llms.LlmConfig {
	return agent.llm.Config()
}

// GetFileSystemBackend returns the agent's filesystem backend.
func (agent *Agent) GetFileSystemBackend() backends.IFileSystemBackend {
	return agent.fsBackend
}

// GetExecutionBackend returns the agent's execution backend.
func (agent *Agent) GetExecutionBackend() backends.IExecutionBackend {
	return agent.executionBackend
}

// RegisterSpawnSubagentTool creates and registers the spawn_subagent tool.
func (agent *Agent) RegisterSpawnSubagentTool() error {
	fd, err := tools.NewFunctionDefinition(
		"spawn_subagent",
		"Use this tool to spawn a subagent to complete a given task. "+
			"The subagent will have access to configurable tools and reasoning effort. "+
			"Available tools options: 'files' (read/write/ls/grep/glob/edit), 'execution' (files + shell), "+
			"'internet' (files + shell + internet_search + open_url), 'all' (everything except spawn_subagent). "+
			"Reasoning options: 'none', 'low', 'medium', 'high'.",
		tools.ParamsFor[SpawnSubagentInput](),
		false,
		agent.spawnSubagent,
	)
	if err != nil {
		return err
	}
	return agent.RegisterFunctionCall(fd)
}
