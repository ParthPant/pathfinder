package agent

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) toolNode(ctx context.Context, state AgentState) (graph.ICommand[AgentState], error) {
	messages := state.messages

	lastMessage := messages[len(messages)-1]
	for _, call := range lastMessage.GetFunctionCalls() {
		slog.Info("Making FunctionCall", "call", call)
		toolMessage, err := agent.toolExecutor.Execute(ctx, call)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		state.messages = append(state.messages, toolMessage)
	}

	return graph.NewCommand("llmNode", state), nil
}
