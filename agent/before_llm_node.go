package agent

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) beforeLlmNode(ctx context.Context,
	ch AgentEventCh,
	chintr AgentIntrCh,
	state AgentState) (graph.ICommand[AgentState, AgentEvent, AgentInterrupt], error) {
	for _, mware := range agent.middlewares {
		newState, err := mware.BeforeLlm(ctx, ch, chintr, state)
		if err != nil {
			slog.Error("Error running middleware", "middleware", reflect.TypeOf(mware).Name(), "error", err)
		}
		state = newState
	}
	return graph.NewCommand[AgentState, AgentEvent, AgentInterrupt]("llmNode", state), nil
}
