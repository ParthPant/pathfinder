package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/tools"
)

type MemoryMiddleware struct {
	backend         backends.IFileSystemBackend
	systemPrompt    string
	memoriesDir     string
	promptMessageId string
}

type Memory struct {
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	createMemoryInput
}

func NewMemoryMiddleware(promptTemplate string, memoriesDir string, fsBackend backends.IFileSystemBackend) *MemoryMiddleware {
	err := os.Mkdir(memoriesDir, 0744)
	if err != nil {
		slog.Warn("Error while creating memories directory", "error", err)
	}
	return &MemoryMiddleware{
		backend:         fsBackend,
		systemPrompt:    promptTemplate,
		memoriesDir:     memoriesDir,
		promptMessageId: "memory_middleware_sys_prompt",
	}
}

func (m *MemoryMiddleware) OnAttach(agent *Agent) error {
	createMemoryToolDefinition, err := tools.NewFunctionDefinition(
		"create_memory",
		"Make a persistent memory entry for future reference.",
		tools.ParamsFor[createMemoryInput](),
		false,
		m.MakeMemory,
	)
	if err != nil {
		return err
	}
	err = agent.RegisterFunctionCall(createMemoryToolDefinition)
	return err
}

func (m *MemoryMiddleware) BeforeAgent(ctx context.Context, state AgentState) (AgentState, error) {
	slog.Debug("Running Memory Middleware.")
	memories, err := m.loadMemoriesFromDir(m.memoriesDir)
	if err != nil {
		return state, err
	}
	slog.Debug("Memories Loaded", "entries", len(memories))

	systemPrompt, err := m.assembleMemoryPrompt(memories)
	if err != nil {
		return state, err
	}
	slog.Debug("Memory prompt assembled", "prompt", systemPrompt[:50])

	for _, message := range state.messages {
		if message.SystemMessage.Id == m.promptMessageId {
			slog.Debug("Memory system prompt aleready present. Replacing with latest content.")
			message = messages.NewTextMessage("system", systemPrompt, &m.promptMessageId)
			return state, nil
		}
	}

	state.messages = append(state.messages, messages.NewTextMessage("system", systemPrompt, &m.promptMessageId))

	return state, nil
}

func (m *MemoryMiddleware) AfterAgent(ctx context.Context, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *MemoryMiddleware) BeforeLlm(ctx context.Context, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *MemoryMiddleware) AfterLlm(ctx context.Context, state AgentState) (AgentState, error) {
	return state, nil
}

type memoriesStruct struct {
	Memories []Memory
}

func (m *MemoryMiddleware) assembleMemoryPrompt(memories []Memory) (string, error) {
	mem := memoriesStruct{
		Memories: memories,
	}

	t, err := template.New("sys_prompt").Parse(m.systemPrompt)
	if err != nil {
		return "", err
	}

	w := strings.Builder{}
	err = t.Execute(&w, mem)
	if err != nil {
		return "", err
	}

	return w.String(), nil
}

func (m *MemoryMiddleware) loadMemoriesFromDir(dir string) ([]Memory, error) {
	slog.Debug("Loading Memories", "path", dir)
	memories := make([]Memory, 0)

	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		} else {
			memories = append(memories, readMemoryFile(path)...)
		}
		return nil
	})
	return memories, nil
}

func readMemoryFile(path string) []Memory {
	file, err := os.Open(path)
	if err != nil {
		slog.Warn("Error while reading memory file", "error", err)
		return nil
	}

	memories := make([]Memory, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var memory Memory
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &memory); err != nil {
			slog.Warn("Error unmarshalling memory", "error", err)
			continue
		}
		memory.Source = path
		memories = append(memories, memory)
	}
	return memories
}

type createMemoryInput struct {
	Title   string `json:"title" tool:"Name of the memory,required"`
	Content string `json:"content" tool:"Content of the memory that should be persisted.,required"`
	Type    string `json:"type" tool:"Type of memory,,user_preferences|skills|facts|workflows|events"`
}

func (m *MemoryMiddleware) MakeMemory(ctx context.Context, params createMemoryInput) (any, error) {
	memoryPath := filepath.Join(m.memoriesDir, params.Type)
	memoryFile := filepath.Join(memoryPath, params.Title+".jsonl")

	os.MkdirAll(memoryPath, 0744)

	memory := Memory{
		Source:            memoryFile,
		CreatedAt:         time.Now(),
		createMemoryInput: params,
	}

	content, err := json.Marshal(memory)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(memoryFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		slog.Warn("Error while opening memory file", "path", memoryFile, "error", err)
		return nil, err
	}

	_, err = f.Write(content)
	if err != nil {
		slog.Warn("Error while writing to memory file", "path", memoryFile, "error", err)
		return nil, err
	}
	return "Successfully written to memory", nil
}
