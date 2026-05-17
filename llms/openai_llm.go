package llms

import (
	"context"
	"fmt"

	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type LlmConfig struct {
	BaseUrl         string
	APIKey          string
	Model           string
	MaxOutputTokens int64
}

type OpenAiLlm struct {
	FunctionDefinitions []tools.FunctionDefinition
	Config              LlmConfig
	client              *openai.Client
}

func NewOpenAiLlm(config LlmConfig) *OpenAiLlm {
	opts := []option.RequestOption{option.WithBaseURL(config.BaseUrl), option.WithAPIKey(config.APIKey)}
	client := openai.NewClient(opts...)
	return &OpenAiLlm{
		Config: config,
		client: &client,
	}
}

func (m *OpenAiLlm) RegisterFunctionDefinition(tool tools.FunctionDefinition) error {
	m.FunctionDefinitions = append(m.FunctionDefinitions, tool)
	return nil
}

func (m *OpenAiLlm) NewResponse(ctx context.Context, input []messages.Message) (messages.Message, error) {
	response, err := m.client.Responses.New(ctx, responses.ResponseNewParams{
		Model:           m.Config.Model,
		MaxOutputTokens: openai.Int(m.Config.MaxOutputTokens),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: createInputItemList(input),
		},
		Reasoning: shared.ReasoningParam{
			Effort:  shared.ReasoningEffortMedium,
			Summary: shared.ReasoningSummaryConcise,
		},
		Tools: m.GetTools(),
	})

	if err != nil {
		return messages.Message{}, err
	}

	message := createAiMessageFromResponse(response)
	return message, nil
}

func (m *OpenAiLlm) NewStream(input []messages.Message) messages.Message {
	panic("Streaming not implemented yet!")
}

func createInputItemList(input []messages.Message) []responses.ResponseInputItemUnionParam {
	var params []responses.ResponseInputItemUnionParam
	for _, message := range input {
		params = append(params, createInputItem(message)...)
	}
	return params
}

func createInputItem(message messages.Message) []responses.ResponseInputItemUnionParam {
	responseInputItems := make([]responses.ResponseInputItemUnionParam, 0)
	switch message.Role {
	case messages.MessageRoleHuman:
		responseInputItems = append(responseInputItems, responses.ResponseInputItemParamOfInputMessage(
			responses.ResponseInputMessageContentListParam{
				createMessageContent(message.HumanMessage.Content),
			},
			message.Role,
		))
	case messages.MessageRoleSystem:
		responseInputItems = append(responseInputItems, responses.ResponseInputItemParamOfInputMessage(
			responses.ResponseInputMessageContentListParam{
				createMessageContent(message.SystemMessage.Content),
			},
			message.Role,
		))
	case messages.MessageRoleAI:
		assistantOutputItems := createAssistantOutputItems(message)
		responseInputItems = append(responseInputItems, assistantOutputItems...)
	case messages.MessageRoleTool:
		responseInputItems = append(responseInputItems, responses.ResponseInputItemParamOfFunctionCallOutput(
			message.ToolMessage.CallId,
			message.ToolMessage.Output,
		))
	default:
		panic(fmt.Sprintf("Unhandeled Message Role role=%s", message.Role))
	}
	return responseInputItems
}

func createMessageContent(content messages.InputContent) responses.ResponseInputContentUnionParam {
	switch content.Type {
	case "input_text":
		return responses.ResponseInputContentUnionParam{
			OfInputText: &responses.ResponseInputTextParam{
				Text: content.OfInputText,
			},
		}
	default:
		panic("Unhandeled InputContent Type.")
	}
}

// TODO: Add reasonig OutputItem handling
func createAssistantOutputItems(message messages.Message) []responses.ResponseInputItemUnionParam {
	responseInputItems := make([]responses.ResponseInputItemUnionParam, 0)
	for _, outputItem := range message.AiMessage.Output {
		switch outputItem.Type {
		case "message":
			contentInputList := make([]responses.ResponseOutputMessageContentUnionParam, 0)
			for _, contentItem := range outputItem.OfMessage.Content {
				contentInputList = append(contentInputList, responses.ResponseOutputMessageContentUnionParam{
					OfOutputText: &responses.ResponseOutputTextParam{
						Text:        contentItem.OutputText,
						Annotations: []responses.ResponseOutputTextAnnotationUnionParam{},
					},
				})
			}
			responseInputItems = append(responseInputItems, responses.ResponseInputItemUnionParam{
				OfOutputMessage: &responses.ResponseOutputMessageParam{
					ID:      outputItem.OfMessage.Id, // FIXME: Message Id is content Id?
					Content: contentInputList,
				},
			})
		case "function_call":
			responseInputItems = append(responseInputItems, responses.ResponseInputItemParamOfFunctionCall(
				outputItem.OfFunctionCall.Arguments,
				outputItem.OfFunctionCall.CallId,
				outputItem.OfFunctionCall.Name,
			))
		case "reasoning":
			reasoningSummaryItems := make([]responses.ResponseReasoningItemSummaryParam, 0)
			for _, summaryItem := range outputItem.OfReasoning.Summary {
				reasoningSummaryItems = append(reasoningSummaryItems, responses.ResponseReasoningItemSummaryParam{
					Type: "summary_text",
					Text: summaryItem.Text,
				})
			}
			responseInputItems = append(responseInputItems, responses.ResponseInputItemParamOfReasoning(
				outputItem.OfReasoning.Id,
				reasoningSummaryItems,
			))
		default:
			panic("Unhandeled AiMessage.Content.Type")
		}
	}
	return responseInputItems
}

func createAiMessageFromResponse(res *responses.Response) messages.Message {
	usage := messages.OutputUsage{
		InputTokens:  res.Usage.InputTokens,
		OutputTokens: res.Usage.OutputTokens,
		TotalTokens:  res.Usage.TotalTokens,
	}

	aiMessage := messages.AiMessage{
		Id:     res.ID,
		Output: createOutputContentFromResponse(res),
		Usage:  usage,
	}

	return messages.Message{
		Role:      messages.MessageRoleAI,
		AiMessage: aiMessage,
	}
}

// TODO: Add reasoning OutputItem handling
func createOutputContentFromResponse(res *responses.Response) []messages.OutputItem {
	outputList := make([]messages.OutputItem, 0)
	for _, responseOutputItem := range res.Output {
		var outputItem messages.OutputItem
		outputItem.Type = responseOutputItem.Type
		outputItem.Id = responseOutputItem.ID

		switch responseOutputItem.Type {
		case "message":
			messageItem := responseOutputItem.AsMessage()
			outputItem.OfMessage.Id = messageItem.ID
			for _, messageContentItem := range messageItem.Content {
				outputItem.OfMessage.Content = append(outputItem.OfMessage.Content, messages.OutputMessageContent{
					Type:       messageContentItem.Type,
					OutputText: messageContentItem.Text,
					Refusal:    messageContentItem.Refusal,
				})
			}
		case "function_call":
			functionCall := responseOutputItem.AsFunctionCall()
			outputItem.OfFunctionCall = messages.OutputFunctionCall{
				Id:        functionCall.ID,
				CallId:    functionCall.CallID,
				Arguments: functionCall.Arguments,
				Name:      functionCall.Name,
			}
		case "reasoning":
			reasoning := responseOutputItem.AsReasoning()
			reasoningContentList := make([]messages.ReasoningContent, 0)
			reasoningSummaryList := make([]messages.ReasoningSummary, 0)
			for _, item := range reasoning.Content {
				reasoningContentList = append(reasoningContentList, messages.ReasoningContent{
					Type: string(item.Type),
					Text: item.Text,
				})
			}
			for _, item := range reasoning.Summary {
				reasoningSummaryList = append(reasoningSummaryList, messages.ReasoningSummary{
					Type: string(item.Type),
					Text: item.Text,
				})
			}
			outputItem.OfReasoning = messages.OutputReasoning{
				Id:               reasoning.ID,
				Summary:          reasoningSummaryList,
				EncryptedContent: reasoning.EncryptedContent,
				Content:          reasoningContentList,
			}
		}

		outputList = append(outputList, outputItem)
	}

	return outputList
}

func (m *OpenAiLlm) GetTools() []responses.ToolUnionParam {
	tools := make([]responses.ToolUnionParam, 0)
	for _, functionDefinition := range m.FunctionDefinitions {
		tools = append(tools, makeToolUnionParam(&functionDefinition))
	}
	return tools
}

func makeToolUnionParam(t *tools.FunctionDefinition) responses.ToolUnionParam {
	return responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Strict:      openai.Bool(t.Strict),
			Parameters: openai.FunctionParameters{
				"type":       "object",
				"properties": t.Parameters.Properties,
				"required":   t.Parameters.Required,
			},
		},
	}
}
