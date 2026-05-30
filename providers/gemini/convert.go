package gemini

import (
	"encoding/json"
	"fmt"
	"strings"

	genai "google.golang.org/genai"
	"github.com/brunoochoa/goai/core"
)

// toContentsAndConfig converte core.Request → ([]*genai.Content, *genai.GenerateContentConfig).
func toContentsAndConfig(req core.Request) ([]*genai.Content, *genai.GenerateContentConfig) {
	cfg := &genai.GenerateContentConfig{}

	// Consolida system prompts.
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
		cfg.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: system}},
		}
	}

	if req.MaxTokens > 0 {
		cfg.MaxOutputTokens = int32(req.MaxTokens)
	}

	if req.Temperature != nil {
		t := float32(*req.Temperature)
		cfg.Temperature = &t
	}

	if req.TopP != nil {
		p := float32(*req.TopP)
		cfg.TopP = &p
	}

	if len(req.Tools) > 0 {
		cfg.Tools = toTools(req.Tools)
	}

	var contents []*genai.Content
	for _, m := range req.Messages {
		if m.Role == core.RoleSystem {
			continue
		}
		if c := toContent(m); c != nil {
			contents = append(contents, c)
		}
	}

	return contents, cfg
}

func toContent(m core.Message) *genai.Content {
	role := genai.RoleUser
	if m.Role == core.RoleAssistant {
		role = genai.RoleModel
	}

	var parts []*genai.Part

	for _, p := range m.Parts {
		if p.Kind == core.ContentText && p.Text != "" {
			parts = append(parts, &genai.Part{Text: p.Text})
		}
	}

	// Tool calls geradas pelo modelo (assistant → user no próximo turno).
	for _, tc := range m.ToolCalls {
		var args map[string]any
		if err := json.Unmarshal(tc.Arguments, &args); err != nil {
			// Argumentos inválidos: propaga como mapa vazio com aviso.
			args = map[string]any{}
		}
		parts = append(parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				Name: tc.Name,
				Args: args,
			},
		})
	}

	// Tool results retornados pelo usuário.
	for _, tr := range m.ToolResults {
		var resp map[string]any
		if err := json.Unmarshal([]byte(tr.Content), &resp); err != nil {
			// Conteúdo não é JSON: encapsula como string.
			resp = map[string]any{"result": tr.Content}
		}
		parts = append(parts, &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     tr.Name,
				Response: resp,
			},
		})
		// Resultados de tools são sempre enviados com role user na API Gemini.
		role = genai.RoleUser
	}

	if len(parts) == 0 {
		return nil
	}

	return &genai.Content{Role: string(role), Parts: parts}
}

func toTools(tools []core.AnyTool) []*genai.Tool {
	decls := make([]*genai.FunctionDeclaration, len(tools))
	for i, t := range tools {
		decls[i] = &genai.FunctionDeclaration{
			Name:                 t.Name(),
			Description:          t.Description(),
			ParametersJsonSchema: json.RawMessage(t.Schema()),
		}
	}
	return []*genai.Tool{{FunctionDeclarations: decls}}
}

// fromResponse converte *genai.GenerateContentResponse → core.Response.
func fromResponse(resp *genai.GenerateContentResponse) core.Response {
	r := core.Response{}

	if resp.UsageMetadata != nil {
		r.Usage = core.Usage{
			InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
			OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		}
	}

	if len(resp.Candidates) == 0 {
		return r
	}

	cand := resp.Candidates[0]
	if cand.FinishReason != "" {
		r.StopReason = string(cand.FinishReason)
	}

	var parts []core.ContentPart
	var toolCalls []core.ToolCall

	if cand.Content != nil {
		for _, p := range cand.Content.Parts {
			if p.Text != "" {
				parts = append(parts, core.ContentPart{Kind: core.ContentText, Text: p.Text})
			}
			if p.FunctionCall != nil {
				raw, _ := json.Marshal(p.FunctionCall.Args)
				toolCalls = append(toolCalls, core.ToolCall{
					ID:        p.FunctionCall.ID,
					Name:      p.FunctionCall.Name,
					Arguments: raw,
				})
			}
		}
	}

	r.Message = core.Message{
		Role:      core.RoleAssistant,
		Parts:     parts,
		ToolCalls: toolCalls,
	}

	return r
}

// mapError converte erros do SDK Gemini em erros do core com sentinels.
// O SDK do Gemini não expõe um tipo de erro estruturado público estável,
// por isso usamos string matching como fallback.
func mapError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "401") || strings.Contains(msg, "API_KEY_INVALID") || strings.Contains(msg, "UNAUTHENTICATED"):
		return &core.APIError{Provider: "gemini", Message: msg, Err: core.ErrAuthFailed}
	case strings.Contains(msg, "429") || strings.Contains(msg, "RESOURCE_EXHAUSTED"):
		return &core.APIError{Provider: "gemini", Message: msg, Err: core.ErrRateLimit}
	case strings.Contains(msg, "404") || strings.Contains(msg, "NOT_FOUND"):
		return &core.APIError{Provider: "gemini", Message: msg, Err: core.ErrModelNotFound}
	}
	return fmt.Errorf("%w: gemini: %w", core.ErrProviderError, err)
}
