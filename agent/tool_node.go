package agent

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) toolNode(ctx context.Context, ch chan<- graph.RunEvent[AgentEvent], chintr chan<- graph.RunInterrupt[AgentInterrupt], state AgentState) (graph.ICommand[AgentState, AgentEvent, AgentInterrupt], error) {
	messages := state.messages

	lastMessage := messages[len(messages)-1]
	for _, call := range lastMessage.GetFunctionCalls() {
		slog.Info("Making FunctionCall", "call", call)

		select {
		case ch <- graph.NewEvent(NewToolCallEvent(call)):
		default:
			slog.Warn("Channel buffer is full. Event dropped.")
		}

		interrupt := graph.NewRunInterrupt(&AgentInterrupt{
			Type: INTR_TOOLCALL,
			OfToolCall: ToolCallInterrupt{
				Call: call,
			},
		})
		slog.Debug("Raising Interrupt", "type", interrupt.Value.Type)

		select {
		case chintr <- interrupt:
		case <-ctx.Done():
			slog.Error("Context finished.", "error", ctx.Err().Error())
			return nil, ctx.Err()
		}

		select {
		case ok := <-interrupt.Resp:
			if !ok {
				slog.Info("Interrupt Canceled.")
				return nil, errors.New("Interrupt Canceled.")
			}
			slog.Info("Interrupt Continued.")
		case <-ctx.Done():
			slog.Error("Context finished.", "error", ctx.Err().Error())
			return nil, ctx.Err()
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
