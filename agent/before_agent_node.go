package agent

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) beforeAgentNode(ctx context.Context, ch chan<- graph.RunEvent[AgentEvent], state AgentState) (graph.ICommand[AgentState, AgentEvent], error) {
	for _, mware := range agent.middlewares {
		newState, err := mware.BeforeAgent(ctx, ch, state)
		if err != nil {
			slog.Error("Error running middleware", "middleware", reflect.TypeOf(mware).Name(), "error", err)
		}
		state = newState
	}

	return graph.NewCommand[AgentState, AgentEvent]("beforeLlmNode", state), nil
}
