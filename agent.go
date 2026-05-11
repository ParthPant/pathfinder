package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/tools"
)

type AgentState struct {
}

type Agent struct {
	BaseGraph[AgentState]
	llm              IToolCallingLlm
	toolExecutor     tools.IToolExecutor
	executionBackend backends.IExecutionBackend
	fsBackend        backends.IFileSystemBackend
	sessionRepo      ISessionRepository
	sessionId        string
}

func NewAgent(llm IToolCallingLlm, toolExecutor tools.IToolExecutor, sessionRepo ISessionRepository) *Agent {
	var agent Agent

	base := BaseGraph[AgentState]{
		State:         AgentState{},
		node:          agentStartNode{&agent},
		completed:     false,
		maxIterations: 10,
		entryNode:     agentStartNode{&agent},
	}

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

	fd, err := backends.LsToolDefinition(fs)
	if err != nil {
		return err
	}
	agent.RegisterFunctionCall(fd)

	fd, err = backends.ReadToolDefinition(fs)
	if err != nil {
		return err
	}
	agent.RegisterFunctionCall(fd)

	fd, err = backends.GrepToolDefinition(fs)
	if err != nil {
		return err
	}
	agent.RegisterFunctionCall(fd)

	fd, err = backends.WriteToolDefinition(fs)
	if err != nil {
		return err
	}
	agent.RegisterFunctionCall(fd)

	fd, err = backends.GlobToolDefinition(fs)
	if err != nil {
		return err
	}
	agent.RegisterFunctionCall(fd)

	fd, err = backends.EditToolDefinition(fs)
	if err != nil {
		return err
	}
	agent.RegisterFunctionCall(fd)

	slog.Debug("Registered File System Backend.")
	return err
}

type agentStartNode struct {
	agent *Agent
}

func (n agentStartNode) Next(ctx context.Context) error {
	slog.Debug("Starting Agent Graph.")
	next := llmCallNode{n.agent}
	n.agent.SetNode(next)
	return nil
}

type llmCallNode struct {
	agent *Agent
}

func (n llmCallNode) Next(ctx context.Context) error {
	// get conversation
	sessionId := n.agent.sessionId
	slog.Debug("Retrieving conversation.", "sessionId", sessionId)
	messages, err := n.agent.sessionRepo.GetById(sessionId)
	if err != nil {
		panic(err)
	}

	// generate response
	slog.Debug("Generating LLM Response.")
	response, err := n.agent.llm.NewResponse(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Reasoning: %s\n", response.ReasoningContent())
	if len(response.OutputText()) > 0 {
		fmt.Printf("AI: %s\n", response.OutputText())
	}

	// save response
	slog.Debug("Saving Response.", "sessionId", sessionId)
	n.agent.sessionRepo.SaveMessage(sessionId, response)

	// check for tool calls
	if response.HasFunctionCalls() {
		slog.Debug("Directing to toolCallNode.")
		toolCallNode := toolCallNode{n.agent}
		n.agent.SetNode(toolCallNode)
		return nil
	}

	// end if no tool calls
	slog.Debug("No tool calls in response.")
	agentEndNode := agentEndNode{n.agent}
	n.agent.SetNode(agentEndNode)
	return nil
}

type toolCallNode struct {
	agent *Agent
}

func (n toolCallNode) Next(ctx context.Context) error {
	sessionId := n.agent.sessionId
	messages, err := n.agent.sessionRepo.GetById(sessionId)
	if err != nil {
		panic(err)
	}
	lastMessage := messages[len(messages)-1]
	for _, call := range lastMessage.GetFunctionCalls() {
		slog.Info("Making FunctionCall", "call", call)
		toolMessage, err := n.agent.toolExecutor.Execute(ctx, call)
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		n.agent.sessionRepo.SaveMessage(sessionId, toolMessage)
	}

	next := llmCallNode{n.agent}
	n.agent.SetNode(next)
	return nil
}

type agentEndNode struct {
	agent *Agent
}

func (n agentEndNode) Next(ctx context.Context) error {
	slog.Debug("Graph Run End.")
	n.agent.SetCompleted()
	return nil
}
