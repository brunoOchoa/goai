package openai

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
	"github.com/brunoochoa/goai/core"
)

// toParams converte core.Request → sdk.ChatCompletionNewParams.
func toParams(req core.Request, defaultModel string, defaultMaxToks int) sdk.ChatCompletionNewParams {
	model := req.Model
	if model == "" {
		model = defaultModel
	}

	maxToks := req.MaxTokens
	if maxToks <= 0 {
		maxToks = defaultMaxToks
	}

	var msgs []sdk.ChatCompletionMessageParamUnion

	// Consolida system prompts (req.System + mensagens com RoleSystem).
	system := req.System
	for _, m := range req.Messages {
		if m.Role == core.RoleSystem {
			if system != "" {
				system += "\n"
			}
			system += m.Text()
		}
	}
	if system != "" {
		msgs = append(msgs, sdk.SystemMessage(system))
	}

	for _, m := range req.Messages {
		if m.Role == core.RoleSystem {
			continue
		}
		msgs = append(msgs, toMessage(m)...)
	}

	params := sdk.ChatCompletionNewParams{
		Model:               shared.ChatModel(model),
		Messages:            msgs,
		MaxCompletionTokens: sdk.Int(int64(maxToks)),
	}

	if len(req.Tools) > 0 {
		params.Tools = toTools(req.Tools)
	}

	if req.Temperature != nil {
		params.Temperature = sdk.Float(*req.Temperature)
	}

	if req.TopP != nil {
		params.TopP = sdk.Float(*req.TopP)
	}

	return params
}

// toMessage converte uma core.Message em uma ou mais mensagens do SDK.
// Retorna slice porque RoleTool com múltiplos ToolResults vira várias mensagens OpenAI
// (a API exige uma mensagem por tool result).
func toMessage(m core.Message) []sdk.ChatCompletionMessageParamUnion {
	switch m.Role {
	case core.RoleUser:
		return []sdk.ChatCompletionMessageParamUnion{sdk.UserMessage(m.Text())}

	case core.RoleAssistant:
		assistant := sdk.ChatCompletionAssistantMessageParam{}
		if text := m.Text(); text != "" {
			assistant.Content.OfString = sdk.String(text)
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]sdk.ChatCompletionMessageToolCallUnionParam, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				tcs[i] = sdk.ChatCompletionMessageToolCallUnionParam{
					OfFunction: &sdk.ChatCompletionMessageFunctionToolCallParam{
						ID: tc.ID,
						Function: sdk.ChatCompletionMessageFunctionToolCallFunctionParam{
							Name:      tc.Name,
							Arguments: string(tc.Arguments),
						},
					},
				}
			}
			assistant.ToolCalls = tcs
		}
		return []sdk.ChatCompletionMessageParamUnion{{OfAssistant: &assistant}}

	case core.RoleTool:
		// Cada ToolResult vira uma mensagem separada.
		out := make([]sdk.ChatCompletionMessageParamUnion, len(m.ToolResults))
		for i, tr := range m.ToolResults {
			out[i] = sdk.ToolMessage(tr.Content, tr.CallID)
		}
		return out

	default:
		// Role desconhecido: trata como user para não quebrar o fluxo.
		return []sdk.ChatCompletionMessageParamUnion{sdk.UserMessage(m.Text())}
	}
}

func toTools(tools []core.AnyTool) []sdk.ChatCompletionToolUnionParam {
	out := make([]sdk.ChatCompletionToolUnionParam, len(tools))
	for i, t := range tools {
		var params shared.FunctionParameters
		if err := json.Unmarshal(t.Schema(), &params); err != nil {
			// Schema inválido: usa objeto vazio; a API retornará erro descritivo.
			params = shared.FunctionParameters{}
		}

		out[i] = sdk.ChatCompletionToolUnionParam{
			OfFunction: &sdk.ChatCompletionFunctionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        t.Name(),
					Description: sdk.String(t.Description()),
					Parameters:  params,
				},
			},
		}
	}
	return out
}

// fromCompletion converte sdk.ChatCompletion → core.Response.
func fromCompletion(c *sdk.ChatCompletion) core.Response {
	resp := core.Response{
		Model: c.Model,
		Usage: core.Usage{
			InputTokens:  int(c.Usage.PromptTokens),
			OutputTokens: int(c.Usage.CompletionTokens),
		},
	}

	if len(c.Choices) == 0 {
		return resp
	}

	choice := c.Choices[0]
	resp.StopReason = string(choice.FinishReason)

	msg := choice.Message
	var parts []core.ContentPart
	var toolCalls []core.ToolCall

	if msg.Content != "" {
		parts = append(parts, core.ContentPart{Kind: core.ContentText, Text: msg.Content})
	}

	for _, tc := range msg.ToolCalls {
		if fn := tc.AsFunction(); fn.ID != "" {
			toolCalls = append(toolCalls, core.ToolCall{
				ID:        fn.ID,
				Name:      fn.Function.Name,
				Arguments: json.RawMessage(fn.Function.Arguments),
			})
		}
	}

	resp.Message = core.Message{
		Role:      core.RoleAssistant,
		Parts:     parts,
		ToolCalls: toolCalls,
	}

	return resp
}

// fromStreamChunk converte sdk.ChatCompletionChunk → core.Event.
func fromStreamChunk(chunk sdk.ChatCompletionChunk, accumulated *string) (core.Event, bool) {
	if len(chunk.Choices) == 0 {
		return core.Event{}, true
	}

	delta := chunk.Choices[0].Delta.Content
	if delta == "" {
		return core.Event{}, true
	}

	*accumulated += delta
	return core.Event{
		Kind:  core.EventText,
		Delta: delta,
		Text:  *accumulated,
	}, false
}

// mapError converte erros do SDK OpenAI em erros do core com sentinels.
func mapError(err error) error {
	var apiErr *sdk.Error
	if errors.As(err, &apiErr) {
		sentinel := core.ErrProviderError
		switch apiErr.StatusCode {
		case 401, 403:
			sentinel = core.ErrAuthFailed
		case 429:
			sentinel = core.ErrRateLimit
		case 404:
			sentinel = core.ErrModelNotFound
		case 400:
			// context_length_exceeded vem como 400 na OpenAI.
			sentinel = core.ErrContextLength
		}
		return &core.APIError{
			StatusCode: apiErr.StatusCode,
			Provider:   "openai",
			Message:    apiErr.Error(),
			Err:        sentinel,
		}
	}
	return fmt.Errorf("%w: openai: %w", core.ErrProviderError, err)
}
