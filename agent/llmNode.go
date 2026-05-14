package agent

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

func (agent *Agent) llmNode(ctx context.Context, state AgentState) (graph.ICommand[AgentState], error) {
	// get conversation
	messages := state.messages

	// generate response
	slog.Debug("Generating LLM Response.")
	response, err := agent.llm.NewResponse(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Reasoning: %s\n", response.ReasoningContent())
	if len(response.OutputText()) > 0 {
		fmt.Printf("AI: %s\n", response.OutputText())
	}

	state.messages = append(state.messages, response)

	// check for tool calls
	if response.HasFunctionCalls() {
		slog.Debug("Directing to toolCallNode.")
		return graph.NewCommand("toolNode", state), nil
	}

	// end if no tool calls
	slog.Debug("No tool calls in response.")
	return graph.NewExitCommand[AgentState](), nil
}
