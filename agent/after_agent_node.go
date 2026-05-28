package agent

import (
	"context"
	"log/slog"
	"reflect"
	"slices"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) afterAgentNode(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (graph.ICommand[AgentState, AgentEvent, AgentInterrupt], error) {
	for _, mware := range slices.Backward(agent.middlewares) {
		newState, err := mware.AfterAgent(ctx, ch, chintr, state)
		if err != nil {
			slog.Error("Error running middleware", "middleware", reflect.TypeOf(mware).Name(), "error", err)
		}
		state = newState
	}

	return graph.NewExitCommand[AgentState, AgentEvent, AgentInterrupt](), nil
}
