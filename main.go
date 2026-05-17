package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/stores"
	"github.com/ParthPant/pathfinder/tools"
	"github.com/joho/godotenv"
)

//go:embed prompts/memory.txt
var memoryPrompt string

//go:embed prompts/summarization.txt
var summarizationPrompt string

func main() {
	godotenv.Load()

	slog.SetLogLoggerLevel(slog.Level(getEnvAsInt("LOG_LEVEL")))

	config := llms.LlmConfig{
		BaseUrl:         os.Getenv("OPENROUTER_BASE_URL"),
		APIKey:          os.Getenv("OPENROUTER_API_KEY"),
		Model:           os.Getenv("MODEL"),
		MaxOutputTokens: 25000,
	}

	llm := llms.NewOpenAiLlm(config)
	toolExecutor := tools.NewToolExecutor()
	inMemStore := stores.NewInMemoryStore[agent.AgentState]()

	a := agent.NewAgent(llm, toolExecutor, inMemStore)

	executionBackend := backends.NewShellBackend(os.Getenv("WORK_DIR"), map[string]string{})
	a.RegisterExecutionBackend(executionBackend)

	fsBackend := backends.NewLocalFileSystemBackend(os.Getenv("WORK_DIR"))
	a.RegisterFileSystemBackend(fsBackend)

	a.RegisterFunctionCall(tools.GetDateTimeTool)
	a.RegisterFunctionCall(tools.InternetSearchTool)
	a.RegisterFunctionCall(tools.OpenURLTool)

	memoryMiddleware := agent.NewMemoryMiddleware(memoryPrompt, ".pathfinder", fsBackend)
	a.AddMiddleware(memoryMiddleware)

	summaryLlmConfig := llms.LlmConfig{
		BaseUrl:         os.Getenv("OPENROUTER_BASE_URL"),
		APIKey:          os.Getenv("OPENROUTER_API_KEY"),
		Model:           os.Getenv("SUMMARY_MODEL"),
		MaxOutputTokens: 25000,
	}
	summaryLlm := llms.NewOpenAiLlm(summaryLlmConfig)
	summarizeMiddleware := agent.NewSummarizationMiddleware(summaryLlm, summarizationPrompt, 90000, 10)
	a.AddMiddleware(summarizeMiddleware)

	_, err := a.StartSession()
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()
	for {
		var userInput string

		fmt.Print("User: ")
		if scanner.Scan() {
			userInput = scanner.Text()
		}

		if userInput == "exit" {
			break
		}

		a.UserInput(messages.NewTextMessage("user", userInput, nil))
		ctx_t, cancel := context.WithTimeout(ctx, time.Second*60*10)
		defer cancel()

		a.Run(ctx_t)
	}
}
