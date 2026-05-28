package agent

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
	"github.com/ParthPant/pathfinder/messages"
)

type AgentInterruptType = string

const (
	INTR_TOOLCALL AgentInterruptType = "INTR_TOOLCALL"
)

type AgentInterrupt struct {
	Type       AgentInterruptType
	OfToolCall ToolCallInterrupt
}

type ToolCallInterrupt struct {
	Call messages.OutputFunctionCall
}

func (agent *Agent) interrupt(ctx context.Context, agentIntr AgentInterrupt, intch AgentIntrCh) (bool, error) {
	interrupt := graph.NewRunInterrupt[AgentState, AgentEvent](&agentIntr)
	slog.Debug("Raising Interrupt", "type", interrupt.Value.Type)

	select {
	case intch <- interrupt:
	case <-ctx.Done():
		slog.Error("Context finished.", "error", ctx.Err().Error())
		return false, ctx.Err()
	}

	select {
	case cmd := <-interrupt.Resp:
		cmd.ApplyTo(agent)
	case <-ctx.Done():
		slog.Error("Context finished.", "error", ctx.Err().Error())
		return false, ctx.Err()
	}
	if agent.IsCompleted() {
		return false, nil
	}
	return true, nil
}
