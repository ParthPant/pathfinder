package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/tools"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	slog.SetLogLoggerLevel(slog.Level(getEnvAsInt("LOG_LEVEL")))

	config := LlmConfig{
		BaseUrl:         os.Getenv("OPENROUTER_BASE_URL"),
		APIKey:          os.Getenv("OPENROUTER_API_KEY"),
		Model:           os.Getenv("MODEL"),
		MaxOutputTokens: 25000,
	}

	llm := NewOpenAiLlm(config)
	toolExecutor := tools.NewToolExecutor()
	sessionRepo := NewInMemorySessionRepo()
	executionBackend := backends.NewShellBackend(os.Getenv("WORK_DIR"), map[string]string{})
	fsBackend := backends.NewLocalFileSystemBackend(os.Getenv("WORK_DIR"))

	agent := NewAgent(llm, toolExecutor, sessionRepo)
	agent.RegisterExecutionBackend(executionBackend)
	agent.RegisterFileSystemBackend(fsBackend)

	agent.RegisterFunctionCall(tools.GetDateTimeTool)
	agent.RegisterFunctionCall(tools.InternetSearchTool)
	agent.RegisterFunctionCall(tools.OpenURLTool)

	_, err := agent.StartSession()
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

		agent.UserInput(messages.NewTextMessage("user", userInput))
		ctx_t, cancel := context.WithTimeout(ctx, time.Second*120)
		defer cancel()

		agent.Run(ctx_t)
	}
}
