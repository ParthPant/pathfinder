package main

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/ParthPant/pathfinder/agent"
	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/cmd/tui/models"
	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/prompts"
	"github.com/ParthPant/pathfinder/stores"
	"github.com/ParthPant/pathfinder/tools"
	tea "github.com/charmbracelet/bubbletea"
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

	ag := agent.NewAgent(llm, toolExecutor, inMemStore)

	executionBackend := backends.NewShellBackend(os.Getenv("WORK_DIR"), os.Environ())
	ag.RegisterExecutionBackend(executionBackend)

	fsBackend := backends.NewLocalFileSystemBackend(os.Getenv("WORK_DIR"))
	ag.RegisterFileSystemBackend(fsBackend)

	ag.RegisterFunctionCall(tools.GetDateTimeTool)
	ag.RegisterFunctionCall(tools.InternetSearchTool)
	ag.RegisterFunctionCall(tools.OpenURLTool)

	memoryMiddleware := agent.NewMemoryMiddleware(prompts.MemoryPrompt, ".pathfinder/memory", fsBackend)
	ag.AddMiddleware(memoryMiddleware)

	skillsMiddleware, err := agent.NewSkillsMiddleware(".pathfinder/skills")
	if err != nil {
		panic(err)
	}
	ag.AddMiddleware(skillsMiddleware)

	summaryLlmConfig := llms.LlmConfig{
		BaseUrl:         os.Getenv("OPENROUTER_BASE_URL"),
		APIKey:          os.Getenv("OPENROUTER_API_KEY"),
		Model:           os.Getenv("SUMMARY_MODEL"),
		MaxOutputTokens: 25000,
	}
	summaryLlm := llms.NewOpenAiLlm(summaryLlmConfig)
	summarizeMiddleware := agent.NewSummarizationMiddleware(summaryLlm, prompts.SummarizationPrompt, 90000, 10)
	ag.AddMiddleware(summarizeMiddleware)

	hitlMiddleware := agent.NewHITLMiddleware()
	ag.AddMiddleware(hitlMiddleware)

	if err := ag.RegisterSpawnSubagentTool(); err != nil {
		panic(err)
	}

	// Create the root model and wire it to the bubbletea program
	rootModel := models.NewRootModel(ag)
	program := tea.NewProgram(rootModel, tea.WithAltScreen())
	rootModel.SetProgram(program)

	if _, err := program.Run(); err != nil {
		slog.Error("TUI error", "error", err)
		os.Exit(1)
	}
}

func getEnvAsInt(name string) int {
	str := os.Getenv(name)
	i, _ := strconv.Atoi(str)
	return i
}
