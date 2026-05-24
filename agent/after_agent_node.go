package agent

import (
	"context"
	"log/slog"
	"reflect"
	"slices"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) afterAgentNode(ctx context.Context, ch chan<- any, state AgentState) (graph.ICommand[AgentState], error) {
	for _, mware := range slices.Backward(agent.middlewares) {
		newState, err := mware.AfterAgent(ctx, ch, state)
		if err != nil {
			slog.Error("Error running middleware", "middleware", reflect.TypeOf(mware).Name(), "error", err)
		}
		state = newState
	}

	return graph.NewExitCommand[AgentState](), nil
}
