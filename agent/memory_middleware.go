package agent

import (
	"context"
	"log/slog"
	"strings"
	"text/template"
	"time"

	"github.com/ParthPant/pathfinder/agent/memory"
	"github.com/ParthPant/pathfinder/backends"
	"github.com/ParthPant/pathfinder/graph"
	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/tools"
)

type MemoryMiddleware struct {
	memoryStore     memory.IMemoryStore
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
	memStore, err := memory.NewFTSMemoryStore(memoriesDir)
	if err != nil {
		slog.Error("Error while creating memory store", "error", err)
	}

	return &MemoryMiddleware{
		memoryStore:     memStore,
		systemPrompt:    promptTemplate,
		memoriesDir:     memoriesDir,
		promptMessageId: "memory_middleware_sys_prompt",
	}
}

func (m *MemoryMiddleware) OnAttach(agent *Agent) error {
	if createMemoryToolDefinition, err := tools.NewFunctionDefinition(
		"create_memory",
		"Make a persistent memory entry for future reference.",
		tools.ParamsFor[createMemoryInput](),
		false,
		m.MakeMemory,
	); err != nil {
		return err
	} else if err := agent.RegisterFunctionCall(createMemoryToolDefinition); err != nil {
		return err
	}

	// if recallMemoryToolDefinition, err := tools.NewFunctionDefinition(
	// 	"recall_memory",
	// 	"Search memory for any stored information.",
	// 	tools.ParamsFor[recallMemoryInput](),
	// 	false,
	// 	m.RecallMemory,
	// ); err != nil {
	// 	return err
	// } else if err := agent.RegisterFunctionCall(recallMemoryToolDefinition); err != nil {
	// 	return err
	// }
	return nil
}

func (m *MemoryMiddleware) BeforeAgent(ctx context.Context, ch chan<- graph.RunEvent[AgentEvent], chintr chan<- graph.RunInterrupt[AgentInterrupt], state AgentState) (AgentState, error) {
	for _, sysMessage := range state.systemMessages {
		if sysMessage.SystemMessage.Id == m.promptMessageId {
			slog.Debug("Memory prompt already present in the conversation. Skipping retrieving memories.")
			return state, nil
		}
	}

	if prompt, err := m.assembleMemoryPrompt(); err != nil {
		slog.Error("Error while creating memory system prompt", "error", err)
		return state, err
	} else {
		state.systemMessages = append(state.systemMessages, messages.NewTextMessage("system", prompt, &m.promptMessageId))
		return state, nil
	}
}

func (m *MemoryMiddleware) AfterAgent(ctx context.Context, ch chan<- graph.RunEvent[AgentEvent], chintr chan<- graph.RunInterrupt[AgentInterrupt], state AgentState) (AgentState, error) {
	return state, nil
}

func (m *MemoryMiddleware) BeforeLlm(ctx context.Context, ch chan<- graph.RunEvent[AgentEvent], chintr chan<- graph.RunInterrupt[AgentInterrupt], state AgentState) (AgentState, error) {
	return state, nil
}

func (m *MemoryMiddleware) AfterLlm(ctx context.Context, ch chan<- graph.RunEvent[AgentEvent], chintr chan<- graph.RunInterrupt[AgentInterrupt], state AgentState) (AgentState, error) {
	return state, nil
}

type memoriesStruct struct {
	Memories []memory.MemoryNote
}

func (m *MemoryMiddleware) assembleMemoryPrompt() (string, error) {
	memories, err := m.memoryStore.GetAll()
	if err != nil {
		slog.Error("Error while retrieving memories from store", "error", err)
		return "", err
	}

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

type createMemoryInput struct {
	Name    string   `json:"name" tool:"Name of the memory,required"`
	Content string   `json:"content" tool:"Content of the memory that should be persisted.,required"`
	Kind    string   `json:"kind" tool:"Type of memory,,semantic|procedural|event"`
	Tags    []string `json:"tags" tool:"A list of tags you want to attached to this memory. Each tag must be a single word only."`
}

func (m *MemoryMiddleware) MakeMemory(ctx context.Context, params createMemoryInput) (any, error) {
	mem := memory.MemoryNote{
		Name:    params.Name,
		Content: params.Content,
		Kind:    params.Kind,
		Tags:    params.Tags,
	}

	if err := m.memoryStore.Insert(mem); err != nil {
		return nil, err
	}

	return "Successfully inserted memory", nil
}

type recallMemoryInput struct {
	Query string `json:"query" tool:"Query stirng for what you want to search in the memory."`
}

func (m *MemoryMiddleware) RecallMemory(ctx context.Context, params recallMemoryInput) (any, error) {
	return m.memoryStore.Search(params.Query)
}
