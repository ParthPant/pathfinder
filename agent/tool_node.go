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
	state AgentState) (graph.ICommand[AgentState, AgentEvent, AgentInterrupt], error) {
	messages := state.messages

	lastMessage := messages[len(messages)-1]
	for _, call := range lastMessage.GetFunctionCalls() {
		slog.Info("Making FunctionCall", "call", call)

		select {
		case ch <- graph.NewEvent(NewToolCallEvent(call)):
		default:
			slog.Warn("Channel buffer is full. Event dropped.")
		}

		intr := AgentInterrupt{
			Type: INTR_TOOLCALL,
			OfToolCall: ToolCallInterrupt{
				Call: call,
			},
		}

		if ok, err := agent.interrupt(ctx, intr, chintr); err != nil {
			return nil, err
		} else if !ok {
			return graph.NoOpCommand[AgentState, AgentEvent, AgentInterrupt](), nil
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
