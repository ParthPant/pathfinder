package agent

import (
	"context"
	"fmt"
	"log/slog"
)

type HITLOpt = func(*HITLMiddleware)

type HITLMiddleware struct {
	agent     *Agent
	hitlTools map[string]struct{}
}

func NewHITLMiddleware(opts ...HITLOpt) *HITLMiddleware {
	m := HITLMiddleware{
		hitlTools: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(&m)
	}
	return &m
}

func WithHITLTool(name string) HITLOpt {
	return func(m *HITLMiddleware) {
		m.hitlTools[name] = struct{}{}
	}
}

// OnAttach registers the middleware with the agent.
func (m *HITLMiddleware) OnAttach(agent *Agent) error {
	m.agent = agent
	return nil
}

// BeforeAgent is a no-op for HITLMiddleware.
func (m *HITLMiddleware) BeforeAgent(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentState, error) {
	return state, nil
}

// AfterAgent is a no-op for HITLMiddleware.
func (m *HITLMiddleware) AfterAgent(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentState, error) {
	return state, nil
}

// BeforeLlm is a no-op for HITLMiddleware.
func (m *HITLMiddleware) BeforeLlm(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentState, error) {
	state.userRejectedTools = make(map[string]struct{})
	return state, nil
}

// AfterLlm intercepts the execution flow before a tool call is executed if the tool name is in the hitlTools set.
func (m *HITLMiddleware) AfterLlm(ctx context.Context, ch AgentEventCh, chintr AgentIntrCh, state AgentState) (AgentState, error) {
	if len(state.messages) == 0 {
		return state, nil
	}

	lastMessage := state.messages[len(state.messages)-1]
	if !lastMessage.HasFunctionCalls() {
		return state, nil
	}

	for _, call := range lastMessage.GetFunctionCalls() {
		if _, ok := m.hitlTools[call.Name]; ok {
			slog.Info("HITL: Intercepting tool", "call_name", call.Name)
			interrupt := AgentInterrupt{
				Type: INTR_TOOLCALL,
				OfToolCall: ToolCallInterrupt{
					Call: call,
				},
			}
			if state, err := m.agent.Interrupt(ctx, &interrupt, chintr); err != nil {
				return state, fmt.Errorf("failed to trigger HITL interrupt: %w", err)
			}
			return state, nil
		}
	}

	return state, nil
}
