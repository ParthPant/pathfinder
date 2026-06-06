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
	"github.com/ParthPant/pathfinder/graph"
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

	memoryMiddleware := agent.NewMemoryMiddleware(prompts.MemoryPrompt, ".pathfinder/memory", fsBackend)
	a.AddMiddleware(memoryMiddleware)

	skillsMiddleware, err := agent.NewSkillsMiddleware(".pathfinder/skills")
	if err != nil {
		panic(err)
	}
	a.AddMiddleware(skillsMiddleware)

	summaryLlmConfig := llms.LlmConfig{
		BaseUrl:         os.Getenv("OPENROUTER_BASE_URL"),
		APIKey:          os.Getenv("OPENROUTER_API_KEY"),
		Model:           os.Getenv("SUMMARY_MODEL"),
		MaxOutputTokens: 25000,
	}
	summaryLlm := llms.NewOpenAiLlm(summaryLlmConfig)
	summarizeMiddleware := agent.NewSummarizationMiddleware(summaryLlm, prompts.SummarizationPrompt, 90000, 10)
	a.AddMiddleware(summarizeMiddleware)

	hitlMiddleware := agent.NewHITLMiddleware()
	a.AddMiddleware(hitlMiddleware)

	// Register the spawn_subagent tool
	if err := a.RegisterSpawnSubagentTool(); err != nil {
		panic(err)
	}

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

		ch, chintr := a.Run(tctx)
		for ch != nil || chintr != nil {
			select {
			case e, ok := <-ch:
				if e.Err != nil {
					fmt.Printf("Error while running Agent: %s\n", e.Err.Error())
				} else if !ok {
					ch = nil
				} else {
					printEvent(e.Value)
				}
			case intr, ok := <-chintr:
				if !ok {
					chintr = nil
					continue
				}
				slog.Info("Interrupt Received.")
				intr.Resp <- handleIntr(intr.Value, scanner)
			}
		}
	}
}

func handleIntr(i *agent.AgentInterrupt, scanner *bufio.Scanner) agent.AgentCmd {
	switch i.Type {
	case agent.INTR_TOOLCALL:
		fmt.Printf("Agent wants to call %s with args %v (y/n): ", i.OfToolCall.Call.Name, i.OfToolCall.Call.Arguments)
	}
	for {
		var userInput string
		if scanner.Scan() {
			userInput = scanner.Text()
		}

		switch userInput {
		case "exit":
		case "n":
			return agent.RejectToolCommand(i.OfToolCall.Call.Name)
		case "y":
			return graph.NoOpCommand[agent.AgentState, agent.AgentEvent, agent.AgentInterrupt]()
		default:
			fmt.Printf("Invalid option. enter y/n: ")
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
