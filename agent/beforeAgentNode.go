package agent

import (
	"context"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) beforeAgentNode(ctx context.Context, state AgentState) (graph.ICommand[AgentState], error) {
	return graph.NewCommand("beforeLlmNode", state), nil
}
