package agent

import (
	"context"
	"log/slog"
	"reflect"
	"slices"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) afterLlmNode(ctx context.Context,
	ch chan<- graph.RunEvent[AgentEvent],
	chintr chan<- graph.RunInterrupt[AgentInterrupt],
	state AgentState) (graph.ICommand[AgentState, AgentEvent, AgentInterrupt], error) {
	for _, mware := range slices.Backward(agent.middlewares) {
		newState, err := mware.AfterLlm(ctx, ch, chintr, state)
		if err != nil {
			slog.Error("Error running middleware", "middleware", reflect.TypeOf(mware).Name(), "error", err)
		}
		state = newState
	}

	lastMessage := state.messages[len(state.messages)-1]
	// TODO: Add retry logic, if llm response is corrupt.
	if lastMessage.HasFunctionCalls() {
		slog.Debug("Directing to toolCallNode.")
		return graph.NewCommand[AgentState, AgentEvent, AgentInterrupt]("toolNode", state), nil
	} else {
		// end if no tool calls
		slog.Debug("No tool calls in response.")
		return graph.NewCommand[AgentState, AgentEvent, AgentInterrupt]("afterAgentNode", state), nil
	}
}
