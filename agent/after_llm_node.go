package agent

import (
	"context"
	"log/slog"
	"reflect"
	"slices"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) afterLlmNode(ctx context.Context, state AgentState) (graph.ICommand[AgentState], error) {
	for _, mware := range slices.Backward(agent.middlewares) {
		newState, err := mware.AfterLlm(ctx, state)
		if err != nil {
			slog.Error("Error running middleware", "middleware", reflect.TypeOf(mware).Name(), "error", err)
		}
		state = newState
	}

	lastMessage := state.messages[len(state.messages)-1]
	// TODO: Add retry logic, if llm response is corrupt.
	if lastMessage.HasFunctionCalls() {
		slog.Debug("Directing to toolCallNode.")
		return graph.NewCommand("toolNode", state), nil
	} else {
		// end if no tool calls
		slog.Debug("No tool calls in response.")
		return graph.NewCommand("afterAgentNode", state), nil
	}
}
