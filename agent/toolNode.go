package agent

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

type toolNode struct {
	agent *Agent
}

func (n toolNode) Run(ctx context.Context) (graph.ICommand[AgentState], error) {
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

	return graph.NewCommand("llmNode", n.agent.State), nil
}
