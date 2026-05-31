package agent

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) beforeAgentNode(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentCmd, error) {
	for _, mware := range agent.middlewares {
		newState, err := mware.BeforeAgent(ctx, ch, chintr, state)
		if err != nil {
			slog.Error("Error running middleware", "middleware", reflect.TypeOf(mware).Name(), "error", err)
		}
		state = newState

		if agent.IsCompleted() {
			return graph.NoOpCommand[AgentState, AgentEvent, AgentInterrupt](), nil
		}
	}

	return graph.NewCommand[AgentState, AgentEvent, AgentInterrupt]("beforeLlmNode", state), nil
}
