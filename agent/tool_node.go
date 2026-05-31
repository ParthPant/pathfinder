package agent

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) toolNode(
	ctx context.Context,
	ch AgentEventCh,
	chintr AgentIntrCh,
	state AgentState) (AgentCmd, error) {
	messages := state.messages

	lastMessage := messages[len(messages)-1]
	for _, call := range lastMessage.GetFunctionCalls() {
		slog.Info("Making FunctionCall", "call", call)

		if _, ok := state.userRejectedTools[call.Name]; ok {
			slog.Debug("Skipping execution for tool", "name", call.Name)
			continue
		}

		select {
		case ch <- graph.NewEvent(NewToolCallEvent(call)):
		default:
			slog.Warn("Channel buffer is full. Event dropped.")
		}

		toolMessage, err := agent.toolExecutor.Execute(ctx, call)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		state.messages = append(state.messages, toolMessage)

		select {
		case ch <- graph.NewEvent(NewToolResponseEvent(toolMessage)):
		default:
			slog.Warn("Channel buffer is full. Event dropped.")
		}
	}

	return graph.NewCommand[AgentState, AgentEvent, AgentInterrupt]("beforeLlmNode", state), nil
}
