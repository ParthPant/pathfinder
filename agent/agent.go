package agent

import (
	"log/slog"

	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/graph"
	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/stores"
	"github.com/ParthPant/pathfinder/tools"
)

type AgentState struct{}

type Agent struct {
	graph.BaseGraph[AgentState]
	llm              llms.IToolCallingLlm
	toolExecutor     tools.IToolExecutor
	executionBackend backends.IExecutionBackend
	fsBackend        backends.IFileSystemBackend
	sessionRepo      stores.ISessionRepository
	sessionId        string
}

func NewAgent(llm llms.IToolCallingLlm, toolExecutor tools.IToolExecutor, sessionRepo stores.ISessionRepository) *Agent {
	var agent Agent

	nodes := map[string]graph.INode[AgentState]{
		"llmNode":  llmNode{&agent},
		"toolNode": toolNode{&agent},
	}
	base := graph.NewBaseGraph(AgentState{}, nodes, "llmNode", 100)

	agent.BaseGraph = base
	agent.llm = llm
	agent.toolExecutor = toolExecutor
	agent.sessionRepo = sessionRepo
	return &agent
}

func (agent *Agent) UserInput(message messages.Message) error {
	sessionId := agent.sessionId
	if err := agent.sessionRepo.SaveMessage(sessionId, message); err != nil {
		return err
	}
	return nil
}

func (agent *Agent) StartSession() (string, error) {
	sessionId, err := agent.sessionRepo.NewSession()
	if err != nil {
		return "", err
	}
	agent.sessionId = sessionId
	return sessionId, nil
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
