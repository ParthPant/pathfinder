package agent

import (
	"context"
	"log/slog"
	"time"

	"github.com/ParthPant/pathfinder/graph"
	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/stores"
	"github.com/ParthPant/pathfinder/tools"
)

// SpawnSubagentInput is the input parameter struct for the spawn_subagent tool.
type SpawnSubagentInput struct {
	Task           string   `json:"task" tool:"The task to be completed by the subagent,required"`
	Files          []string `json:"files,omitzero" tool:"Optional list of file paths to provide as context to the subagent"`
	Reasoning      string   `json:"reasoning,omitzero" tool:"Reasoning effort for the subagent LLM,,none|low|medium|high"`
	AvailableTools string   `json:"available_tools,omitzero" tool:"Tool categories available to the subagent,,files|execution|internet|all"`
}

// SpawnSubagentTool is the FunctionDefinition for the spawn_subagent tool.
// The Function field is set dynamically in RegisterSpawnSubagentTool.
var SpawnSubagentTool = tools.FunctionDefinition{
	Type: "function",
	Name: "spawn_subagent",
	Description: "Use this tool to spawn a subagent to complete a given task. " +
		"The subagent will have access to configurable tools and reasoning effort. " +
		"Available tools: 'files' (read/write/ls/grep/glob/edit), 'execution' (files + shell), " +
		"'internet' (files + shell + internet_search + open_url), 'all' (everything except spawn_subagent). " +
		"Reasoning: 'none', 'low', 'medium', 'high', 'xhigh'.",
	Strict: false,
}

// spawnSubagent creates a new isolated subagent, runs it synchronously,
// and returns the final output text to the parent agent.
func (agent *Agent) spawnSubagent(ctx context.Context, params SpawnSubagentInput) (any, error) {
	slog.Info("Spawning subagent", "task", params.Task,
		"files", params.Files, "reasoning", params.Reasoning, "tools", params.AvailableTools)

	// Determine reasoning effort
	reasoningEffort := llms.ReasoningEffortType(params.Reasoning)
	if reasoningEffort == "" {
		reasoningEffort = llms.ReasoningEffortType("medium")
	}

	// 1. Clone LLM config and set reasoning effort
	cfg := *agent.GetLlmConfig()
	cfg.ReasoningEffort = reasoningEffort
	subLlm := llms.NewOpenAiLlm(cfg)

	// 2. Create a new fresh tool executor and session store
	subToolExecutor := tools.NewToolExecutor()
	subStore := stores.NewInMemoryStore[AgentState]()

	// 3. Construct a brand-new Agent (no middlewares attached)
	subAgent := NewAgent(subLlm, subToolExecutor, subStore)

	// 4. Determine tool availability
	toolSet := params.AvailableTools
	if toolSet == "" {
		toolSet = "all" // default
	}

	// Always add get_date_time
	if err := subAgent.RegisterFunctionCall(tools.GetDateTimeTool); err != nil {
		return nil, err
	}

	// Always add filesystem tools (read, write, ls, grep, glob, edit)
	if err := subAgent.RegisterFileSystemBackend(agent.GetFileSystemBackend()); err != nil {
		return nil, err
	}

	// Conditionally add execution (shell) tool
	if toolSet == "execution" || toolSet == "internet" || toolSet == "all" {
		if err := subAgent.RegisterExecutionBackend(agent.GetExecutionBackend()); err != nil {
			return nil, err
		}
	}

	// Conditionally add internet tools
	if toolSet == "internet" || toolSet == "all" {
		if err := subAgent.RegisterFunctionCall(tools.InternetSearchTool); err != nil {
			return nil, err
		}
		if err := subAgent.RegisterFunctionCall(tools.OpenURLTool); err != nil {
			return nil, err
		}
	}

	// NOTE: spawn_subagent and create_memory are NEVER registered on subagents.

	// 5. Optionally load files as context messages
	for _, filePath := range params.Files {
		subAgent.UserInput(messages.NewTextMessage("user",
			"Please read and use the file at: "+filePath, nil))
	}

	// 6. Inject the task as a user message
	subAgent.UserInput(messages.NewTextMessage("user", params.Task, nil))

	// 7. Start a new session for the subagent
	if _, err := subAgent.StartSession(); err != nil {
		return nil, err
	}

	// 8. Create a derived context with timeout to prevent hanging
	spawnCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// 9. Run the subagent synchronously and collect events
	var finalOutput string
	ch, chintr := subAgent.Run(spawnCtx)
	for ch != nil || chintr != nil {
		select {
		case e, ok := <-ch:
			if !ok {
				ch = nil
				continue
			}
			if e.Err != nil {
				slog.Error("Subagent error", "error", e.Err)
				continue
			}
			if e.Value != nil && e.Value.Type == AIRESP {
				text := e.Value.OfAiResponse.Message.OutputText()
				if text != "" {
					finalOutput = text
				}
			}
		case intr, ok := <-chintr:
			if !ok {
				chintr = nil
				continue
			}
			slog.Warn("Subagent interruption received, rejecting", "type", intr.Value.Type)
			intr.Resp <- graph.NoOpCommand[AgentState, AgentEvent, AgentInterrupt]()
		case <-spawnCtx.Done():
			slog.Warn("Subagent timed out", "error", spawnCtx.Err())
			return "Subagent timed out: " + spawnCtx.Err().Error(), nil
		}
	}

	slog.Info("Subagent completed", "outputLength", len(finalOutput))
	return finalOutput, nil
}
