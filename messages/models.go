package messages

import (
	"io"
	"strings"

	"github.com/google/uuid"
)

type MessageRole = string
type OutputType = string

const (
	MessageRoleHuman  MessageRole = "user"
	MessageRoleAI     MessageRole = "assistant"
	MessageRoleSystem MessageRole = "system"
	MessageRoleTool   MessageRole = "tool"
)

type Message struct {
	Role MessageRole
	HumanMessage
	SystemMessage
	AiMessage
	ToolMessage
}

type HumanMessage struct {
	Id      string
	Content InputContent
}

type SystemMessage struct {
	Id      string
	Content InputContent
}

type ToolMessage struct {
	Type   string
	CallId string
	Output string
	Id     string
}

type AiMessage struct {
	Id     string
	Output []OutputItem // Output[] > Message > Content[]
	Usage  OutputUsage
}

type InputContent struct {
	Type        string
	OfInputFile io.Reader
	OfInputText string
}

type OutputMessage struct {
	Id      string
	Content []OutputMessageContent
}

type OutputMessageContent struct {
	Type       string
	OutputText string
	Refusal    string
}

type OutputFunctionCall struct {
	Id        string
	Arguments string
	CallId    string
	Name      string
}

type OutputUsage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	Cost         float64
}

type OutputReasoning struct {
	Id               string
	Summary          []ReasoningSummary
	EncryptedContent string
	Content          []ReasoningContent
}

type ReasoningSummary struct {
	Type string
	Text string
}

type ReasoningContent struct {
	Type string
	Text string
}

type OutputItem struct {
	Id             string
	Type           string
	OfMessage      OutputMessage
	OfFunctionCall OutputFunctionCall
	OfReasoning    OutputReasoning
}

func (m *Message) HasFunctionCalls() bool {
	for _, contentItem := range m.AiMessage.Output {
		if contentItem.Type == "function_call" {
			return true
		}
	}
	return false
}

func (m *Message) GetFunctionCalls() []OutputFunctionCall {
	functionCalls := make([]OutputFunctionCall, 0)
	for _, contentItem := range m.AiMessage.Output {
		if contentItem.Type == "function_call" {
			functionCalls = append(functionCalls, contentItem.OfFunctionCall)
		}
	}
	return functionCalls
}

func NewTextMessage(role string, text string, id *string) Message {
	if id == nil {
		uuid, _ := uuid.NewV7()
		id = new(uuid.String())
	}
	message := Message{}
	switch role {
	case "user":
		message.Role = role
		message.HumanMessage.Content.Type = "input_text"
		message.HumanMessage.Content.OfInputText = text
		message.HumanMessage.Id = *id
	case "system":
		message.Role = role
		message.SystemMessage.Content.Type = "input_text"
		message.SystemMessage.Content.OfInputText = text
		message.SystemMessage.Id = *id
	default:
		panic("Unhandeled role for input message.")
	}
	return message
}

func (m *Message) OutputText() string {
	var outputText strings.Builder
	for _, output := range m.AiMessage.Output {
		if output.Type == "message" {
			for _, content := range output.OfMessage.Content {
				if content.Type == "output_text" {
					outputText.WriteString(content.OutputText)
				}
			}
		}
	}
	return outputText.String()
}

// FIXME: Models are not returnign summary for some reason.
func (m *Message) ReasoningSummary() string {
	var outputText strings.Builder
	for _, output := range m.AiMessage.Output {
		if output.Type == "reasoning" {
			for _, summaryItem := range output.OfReasoning.Summary {
				outputText.WriteString(summaryItem.Text)
			}
		}
	}
	return outputText.String()
}

func (m *Message) ReasoningContent() string {
	var outputText strings.Builder
	for _, output := range m.AiMessage.Output {
		if output.Type == "reasoning" {
			for _, contentItem := range output.OfReasoning.Content {
				outputText.WriteString(contentItem.Text)
			}
		}
	}
	return outputText.String()
}

func (m *Message) GetTextContent() string {
	var sb strings.Builder

	switch m.Role {
	case MessageRoleHuman:
		sb.WriteString(m.HumanMessage.Content.OfInputText)
	case MessageRoleSystem:
		sb.WriteString(m.SystemMessage.Content.OfInputText)
	case MessageRoleTool:
		sb.WriteString(m.ToolMessage.Output)
	case MessageRoleAI:
		for _, item := range m.AiMessage.Output {
			switch item.Type {
			case "message":
				for _, content := range item.OfMessage.Content {
					sb.WriteString(content.OutputText)
					sb.WriteString(content.Refusal)
				}
			case "function_call":
				// TODO: Should this only take the arugments.
				// Primary usecase is to find token count.
				// Are function name, call_id etc. significant?
				sb.WriteString(item.OfFunctionCall.Arguments)
			case "reasoning":
				for _, summary := range item.OfReasoning.Summary {
					sb.WriteString(summary.Text)
				}
				for _, content := range item.OfReasoning.Content {
					sb.WriteString(content.Text)
				}
			}
		}
	}

	return sb.String()
}
