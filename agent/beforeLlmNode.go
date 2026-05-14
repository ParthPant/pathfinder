package agent

import (
	"context"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) beforeLlmNode(ctx context.Context, state AgentState) (graph.ICommand[AgentState], error) {
	return graph.NewCommand("llmNode", state), nil
}
