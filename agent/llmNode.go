package agent

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/ParthPant/pathfinder/graph"
)

type llmNode struct {
	agent *Agent
}

func (n llmNode) Run(ctx context.Context) (graph.ICommand[AgentState], error) {
	// get conversation
	sessionId := n.agent.sessionId
	slog.Debug("Retrieving conversation.", "sessionId", sessionId)
	messages, err := n.agent.sessionRepo.GetById(sessionId)
	if err != nil {
		panic(err)
	}

	// generate response
	slog.Debug("Generating LLM Response.")
	response, err := n.agent.llm.NewResponse(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Reasoning: %s\n", response.ReasoningContent())
	if len(response.OutputText()) > 0 {
		fmt.Printf("AI: %s\n", response.OutputText())
	}

	// save response
	slog.Debug("Saving Response.", "sessionId", sessionId)
	n.agent.sessionRepo.SaveMessage(sessionId, response)

	// check for tool calls
	if response.HasFunctionCalls() {
		slog.Debug("Directing to toolCallNode.")
		return graph.NewCommand("toolNode", n.agent.State), nil
	}

	// end if no tool calls
	slog.Debug("No tool calls in response.")
	return graph.NewExitCommand[AgentState](), nil
}
