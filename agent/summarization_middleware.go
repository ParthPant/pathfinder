package agent

import (
	"context"
	"log/slog"

	"github.com/ParthPant/pathfinder/llms"
	"github.com/ParthPant/pathfinder/messages"
)

type SummarizationMiddleware struct {
	tokenThreshold      int64
	summarizationPrompt string
	keep                int
	llm                 llms.ILlm
}

func NewSummarizationMiddleware(llm llms.ILlm, prompt string, tokenThreshold int64, keep int) *SummarizationMiddleware {
	return &SummarizationMiddleware{
		tokenThreshold:      tokenThreshold,
		summarizationPrompt: prompt,
		keep:                keep,
		llm:                 llm,
	}
}

func (m *SummarizationMiddleware) OnAttach(agent *Agent) error {
	return nil
}

func (m *SummarizationMiddleware) BeforeAgent(_ context.Context, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *SummarizationMiddleware) AfterAgent(_ context.Context, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *SummarizationMiddleware) BeforeLlm(ctx context.Context, state AgentState) (AgentState, error) {
	predictedTokens := m.estimateTokens(state.messages)

	if predictedTokens >= m.tokenThreshold {
		slog.Info("Summarizing Conversation", "predictedTokens", predictedTokens, "threshold", m.tokenThreshold, "keep", m.keep, "conversationLength", len(state.messages))
		if compactedMessages, err := m.summarize(ctx, state.messages); err != nil {
			slog.Warn("Error summarizing conversation", "error", err)
		} else {
			state.messages = compactedMessages
		}
	}
	return state, nil
}

func (m *SummarizationMiddleware) AfterLlm(_ context.Context, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *SummarizationMiddleware) summarize(ctx context.Context, conversation []messages.Message) ([]messages.Message, error) {
	if m.keep >= len(conversation) {
		return conversation, nil
	}

	summarizeMessages := []messages.Message{
		messages.NewTextMessage(messages.MessageRoleSystem, m.summarizationPrompt, nil),
	}

	conversationSplit := len(conversation) - m.keep
	summarizeMessages = append(summarizeMessages, conversation[:conversationSplit]...)

	response, err := m.llm.NewResponse(ctx, summarizeMessages)
	if err != nil {
		return nil, err
	}

	return append([]messages.Message{response}, conversation[conversationSplit:]...), nil
}

func (m *SummarizationMiddleware) estimateTokens(msgs []messages.Message) int64 {
	var totalChars int
	for _, msg := range msgs {
		totalChars += len(msg.GetTextContent())
	}
	// Heuristic: ~4 characters per token for English
	return int64(totalChars / 4)
}
