package anthropic

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/brunoochoa/goai/core"
)

// toParams converte core.Request → sdk.MessageNewParams.
func toParams(req core.Request, defaultModel string, defaultMaxToks int64) (sdk.MessageNewParams, error) {
	model := req.Model
	if model == "" {
		model = defaultModel
	}

	maxToks := int64(req.MaxTokens)
	if maxToks <= 0 {
		maxToks = defaultMaxToks
	}

	var system string
	var msgs []sdk.MessageParam

	for _, m := range req.Messages {
		if m.Role == core.RoleSystem {
			// Acumula todos os blocos de sistema separados por newline.
			if system != "" {
				system += "\n"
			}
			system += m.Text()
			continue
		}
		p, err := toMessageParam(m)
		if err != nil {
			return sdk.MessageNewParams{}, err
		}
		msgs = append(msgs, p)
	}

	// req.System tem prioridade, seguido das mensagens do sistema.
	if req.System != "" {
		if system != "" {
			system = req.System + "\n" + system
		} else {
			system = req.System
		}
	}

	params := sdk.MessageNewParams{
		Model:     sdk.Model(model),
		Messages:  msgs,
		MaxTokens: maxToks,
	}

	if system != "" {
		params.System = []sdk.TextBlockParam{{Text: system}}
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

	return params, nil
}

func toMessageParam(m core.Message) (sdk.MessageParam, error) {
	switch m.Role {
	case core.RoleUser:
		var blocks []sdk.ContentBlockParamUnion
		for _, p := range m.Parts {
			blocks = append(blocks, sdk.NewTextBlock(p.Text))
		}
		for _, tr := range m.ToolResults {
			blocks = append(blocks, sdk.NewToolResultBlock(tr.CallID, tr.Content, tr.IsError))
		}
		return sdk.NewUserMessage(blocks...), nil

	case core.RoleAssistant:
		var blocks []sdk.ContentBlockParamUnion
		for _, p := range m.Parts {
			blocks = append(blocks, sdk.NewTextBlock(p.Text))
		}
		for _, tc := range m.ToolCalls {
			var input any
			if err := json.Unmarshal(tc.Arguments, &input); err != nil {
				// Argumentos inválidos do modelo: propaga o erro para evitar comportamento silencioso.
				return sdk.MessageParam{}, fmt.Errorf("tool call %q: argumentos JSON inválidos: %w", tc.Name, err)
			}
			blocks = append(blocks, sdk.NewToolUseBlock(tc.ID, input, tc.Name))
		}
		return sdk.NewAssistantMessage(blocks...), nil

	default:
		return sdk.MessageParam{}, fmt.Errorf("role não suportado pelo Anthropic: %q", m.Role)
	}
}

func toTools(tools []core.AnyTool) []sdk.ToolUnionParam {
	out := make([]sdk.ToolUnionParam, len(tools))
	for i, t := range tools {
		var raw map[string]any
		if err := json.Unmarshal(t.Schema(), &raw); err != nil {
			// Schema inválido: registra no log mas não aborta — a API recusará se realmente inválido.
			raw = map[string]any{"type": "object"}
		}

		var props any
		var required []string
		props = raw["properties"]
		if r, ok := raw["required"].([]any); ok {
			for _, v := range r {
				if s, ok := v.(string); ok {
					required = append(required, s)
				}
			}
		}

		tool := sdk.ToolParam{
			Name:        t.Name(),
			Description: sdk.String(t.Description()),
			InputSchema: sdk.ToolInputSchemaParam{
				Properties: props,
				Required:   required,
			},
		}
		out[i] = sdk.ToolUnionParam{OfTool: &tool}
	}
	return out
}

// fromMessage converte sdk.Message → core.Response.
func fromMessage(msg *sdk.Message) core.Response {
	resp := core.Response{
		Model: string(msg.Model),
		Usage: core.Usage{
			InputTokens:  int(msg.Usage.InputTokens),
			OutputTokens: int(msg.Usage.OutputTokens),
			CachedTokens: int(msg.Usage.CacheReadInputTokens),
		},
		StopReason: string(msg.StopReason),
	}

	var parts []core.ContentPart
	var toolCalls []core.ToolCall

	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case sdk.TextBlock:
			parts = append(parts, core.ContentPart{Kind: core.ContentText, Text: b.Text})
		case sdk.ToolUseBlock:
			raw, _ := json.Marshal(b.Input)
			toolCalls = append(toolCalls, core.ToolCall{
				ID:        b.ID,
				Name:      b.Name,
				Arguments: raw,
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

// fromStreamEvent converte um MessageStreamEventUnion em core.Event.
// Retorna skip=true para eventos que devem ser ignorados.
func fromStreamEvent(ev sdk.MessageStreamEventUnion, accumulated *string) (core.Event, bool) {
	switch ev.Type {
	case "content_block_delta":
		if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
			*accumulated += ev.Delta.Text
			return core.Event{
				Kind:  core.EventText,
				Delta: ev.Delta.Text,
				Text:  *accumulated,
			}, false
		}
	case "message_delta":
		if ev.Usage.OutputTokens > 0 {
			return core.Event{
				Kind: core.EventUsage,
				Usage: &core.Usage{
					OutputTokens: int(ev.Usage.OutputTokens),
				},
			}, false
		}
	}
	return core.Event{}, true
}

// mapError converte erros do SDK Anthropic em erros do core com sentinels.
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
		case 413:
			sentinel = core.ErrContextLength
		}
		return &core.APIError{
			StatusCode: apiErr.StatusCode,
			Provider:   "anthropic",
			Message:    apiErr.Error(),
			Err:        sentinel,
		}
	}
	return fmt.Errorf("%w: anthropic: %w", core.ErrProviderError, err)
}
