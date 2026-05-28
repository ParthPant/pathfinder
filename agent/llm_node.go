package agent

import (
	"context"
	"log"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) llmNode(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (graph.ICommand[AgentState, AgentEvent, AgentInterrupt], error) {
	// get conversation
	messages := append(state.systemMessages, state.messages...)

	// generate response
	slog.Debug("Generating LLM Response.", "model", agent.llm.Config().Model)
	response, err := agent.llm.NewResponse(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	select {
	case ch <- graph.NewEvent(NewAiResponseEvent(response)):
	default:
		slog.Warn("Channel buffer is full. Event dropped.")
	}

	state.messages = append(state.messages, response)

	return graph.NewCommand[AgentState, AgentEvent, AgentInterrupt]("afterLlmNode", state), nil
}
