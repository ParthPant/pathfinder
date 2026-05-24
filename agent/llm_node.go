package agent

import (
	"context"
	"log"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) llmNode(ctx context.Context, ch chan<- any, state AgentState) (graph.ICommand[AgentState], error) {
	// get conversation
	messages := append(state.systemMessages, state.messages...)

	// generate response
	slog.Debug("Generating LLM Response.")
	response, err := agent.llm.NewResponse(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	select {
	case ch <- EventAIResponse{Message: response}:
	default:
		slog.Warn("Channel buffer is full. Event dropped.")
	}

	state.messages = append(state.messages, response)

	return graph.NewCommand("afterLlmNode", state), nil
}
