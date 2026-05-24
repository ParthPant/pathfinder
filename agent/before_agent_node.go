package agent

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) beforeAgentNode(ctx context.Context, ch chan<- any, state AgentState) (graph.ICommand[AgentState], error) {
	for _, mware := range agent.middlewares {
		newState, err := mware.BeforeAgent(ctx, ch, state)
		if err != nil {
			slog.Error("Error running middleware", "middleware", reflect.TypeOf(mware).Name(), "error", err)
		}
		state = newState
	}

	return graph.NewCommand("beforeLlmNode", state), nil
}
