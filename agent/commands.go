package agent

import "github.com/ParthPant/pathfinder/graph"

type rejectToolCommand struct {
	toolName string
}

func (c *rejectToolCommand) ApplyTo(agent graph.IGraph[AgentState, AgentEvent, AgentInterrupt]) error {
	state := agent.GetState()
	state.userRejectedTools[c.toolName] = struct{}{}
	agent.SetState(state)
	return nil
}

func RejectToolCommand(toolName string) *rejectToolCommand {
	return &rejectToolCommand{
		toolName,
	}
}
