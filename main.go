package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/prompts"
	"github.com/ParthPant/pathfinder/stores"
	"github.com/ParthPant/pathfinder/tools"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	logLevel := slog.Level(getEnvAsInt("LOG_LEVEL"))
	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)

	slog.SetDefault(logger)

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

	memoryMiddleware := agent.NewMemoryMiddleware(prompts.MemoryPrompt, ".pathfinder", fsBackend)
	a.AddMiddleware(memoryMiddleware)

	summaryLlmConfig := llms.LlmConfig{
		BaseUrl:         os.Getenv("OPENROUTER_BASE_URL"),
		APIKey:          os.Getenv("OPENROUTER_API_KEY"),
		Model:           os.Getenv("SUMMARY_MODEL"),
		MaxOutputTokens: 25000,
	}
	summaryLlm := llms.NewOpenAiLlm(summaryLlmConfig)
	summarizeMiddleware := agent.NewSummarizationMiddleware(summaryLlm, prompts.SummarizationPrompt, 90000, 10)
	a.AddMiddleware(summarizeMiddleware)

	_, err = a.StartSession()
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()
	for {
		fmt.Print("User: ")

		var userInput string
		if scanner.Scan() {
			userInput = scanner.Text()
		}
		if userInput == "exit" {
			break
		}

		if userInput == "" {
			continue
		}

		a.UserInput(messages.NewTextMessage("user", userInput, nil))
		tctx, cancel := context.WithTimeout(ctx, 15*60*time.Second)
		defer cancel()

		ch := a.Run(tctx)
		for e := range ch {
			if e.Err != nil {
				fmt.Printf("Error while running Agent: %s\n", e.Err.Error())
			} else {
				printEvent(e.Value)
			}
		}
	}
}

func printEvent(e *agent.AgentEvent) {
	switch e.Type {
	case agent.AIRESP:
		msg := e.OfAiResponse.Message
		reasoning := msg.ReasoningContent()
		if reasoning != "" {
			fmt.Printf("AI (reasoning): %s\n", reasoning)
		}
		output := msg.OutputText()
		if output != "" {
			fmt.Printf("AI: %s\n", output)
		}
	case agent.TOOLCALL:
		fmt.Printf("AI [toolCall]: %s\n", e.OfToolCall.Call.Name)
	case agent.TOOLRESP:
		fmt.Printf("Tool Response: %s\n", e.OfToolResponse.Message.OutputText())
	case agent.AGENTERR:
		fmt.Printf("Agent Error: %s\n", e.OfError.Err.Error())
	default:
	}
}

func getEnvAsInt(name string) int {
	str := os.Getenv(name)
	i, _ := strconv.Atoi(str)
	return i
}
