package agent

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) afterLlmNode(ctx context.Context, state AgentState) (graph.ICommand[AgentState], error) {
	lastMessage := state.messages[len(state.messages)-1]
	// TODO: Add retry logic, if llm response is corrupt.
	if lastMessage.HasFunctionCalls() {
		slog.Debug("Directing to toolCallNode.")
		return graph.NewCommand("toolNode", state), nil
	} else {
		// end if no tool calls
		slog.Debug("No tool calls in response.")
		return graph.NewExitCommand[AgentState](), nil
	}
}
