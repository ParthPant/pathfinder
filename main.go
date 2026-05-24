package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

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
		ch, cherr := a.Run(ctx)

		for ch != nil || cherr != nil {
			select {
			case e, ok := <-ch:
				if !ok {
					ch = nil
				}
				printEvent(e)
			case err, ok := <-cherr:
				if err != nil {
					slog.Error("Error during Agent Run", "error", err)
					cherr = nil
				}

				if !ok {
					ch = nil
					cherr = nil
				}
			}
		}
	}
}

func printEvent(e any) {
	switch v := e.(type) {
	case agent.EventAIResponse:
		fmt.Printf("AI: %s\n", v.Message.OutputText())
	case agent.EventToolCall:
		fmt.Printf("AI [toolCall]: %s\n", v.Call.Name)
	case agent.EventToolResponse:
		fmt.Printf("Tool: %s\n", v.Message.OutputText())
	}
}

func getEnvAsInt(name string) int {
	str := os.Getenv(name)
	i, _ := strconv.Atoi(str)
	return i
}
