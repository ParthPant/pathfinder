package agent

import (
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
}

type Agent struct {
	graph.BaseGraph[AgentState, AgentEvent]
	llm              llms.IToolCallingLlm
	toolExecutor     tools.IToolExecutor
	executionBackend backends.IExecutionBackend
	fsBackend        backends.IFileSystemBackend
	middlewares      []IMiddleware[AgentState]
}

func NewAgent(llm llms.IToolCallingLlm, toolExecutor tools.IToolExecutor, sessionRepo stores.IStore[AgentState]) *Agent {
	var agent Agent

	nodes := map[string]graph.Node[AgentState, AgentEvent]{
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
	return &agent
}

func (agent *Agent) UserInput(message messages.Message) error {
	state := agent.GetState()
	state.messages = append(state.messages, message)
	agent.SetState(state)
	return nil
}

func (agent *Agent) StartSession() (string, error) {
	return agent.NewSession()
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

func (agent *Agent) GetModel() llms.IToolCallingLlm {
	return agent.llm
}

func (agent *Agent) GetTools() []tools.FunctionDefinition {
	return agent.toolExecutor.GetTools()
}
