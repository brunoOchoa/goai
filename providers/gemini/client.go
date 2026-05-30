// Package gemini implements core.Client using the official Google GenAI Go SDK.
//
// # Installation
//
//	go get github.com/brunoochoa/goai/providers/gemini
//
// # Authentication
//
// Set GEMINI_API_KEY (or GOOGLE_API_KEY) in the environment, or pass [WithAPIKey].
// For Vertex AI, use [WithVertexAI] and ensure Application Default Credentials are configured.
// The client returns an error at construction time if no key is found.
//
// # Example
//
//	client, err := gemini.NewClient(ctx)
//	resp, err := client.Chat(ctx, core.Request{
//	    Messages: []core.Message{core.UserMessage("Olá!")},
//	})
package gemini

import (
	"context"
	"errors"
	"iter"
	"os"

	genai "google.golang.org/genai"
	"github.com/brunoochoa/goai/core"
)

const defaultModel = "gemini-2.5-flash"

// Client implements [core.Client] for the Google Gemini API.
// Construct with [NewClient]; the zero value is not usable.
type Client struct {
	inner *genai.Client
	model string
}

// Option configures a [Client].
type Option func(*clientConfig)

type clientConfig struct {
	apiKey  string
	project string
	region  string
	backend genai.Backend
	model   string
	maxToks int32
}

// WithAPIKey sets the Gemini API key explicitly.
// Overrides GEMINI_API_KEY and GOOGLE_API_KEY environment variables.
func WithAPIKey(k string) Option { return func(c *clientConfig) { c.apiKey = k } }

// WithModel sets the default model for all calls on this client.
// Default: "gemini-2.5-flash".
func WithModel(m string) Option { return func(c *clientConfig) { c.model = m } }

// WithMaxTokens sets the default maximum output tokens per request.
// Default: 8192.
func WithMaxTokens(n int) Option { return func(c *clientConfig) { c.maxToks = int32(n) } }

// WithVertexAI switches the backend to Vertex AI.
// Requires a GCP project ID and region; authentication uses Application Default Credentials.
func WithVertexAI(project, region string) Option {
	return func(c *clientConfig) {
		c.project = project
		c.region = region
		c.backend = genai.BackendVertexAI
	}
}

// NewClient creates a Gemini client.
// Returns an error if no API key is found (Gemini API backend) or if client initialization fails.
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	cfg := &clientConfig{
		apiKey:  firstEnv("GEMINI_API_KEY", "GOOGLE_API_KEY"),
		model:   defaultModel,
		maxToks: 8192,
		backend: genai.BackendGeminiAPI,
	}
	for _, o := range opts {
		o(cfg)
	}

	// Vertex AI usa Application Default Credentials — não precisa de API key.
	if cfg.backend == genai.BackendGeminiAPI && cfg.apiKey == "" {
		return nil, errors.New("gemini: API key ausente — defina GEMINI_API_KEY ou use WithAPIKey()")
	}

	cc := &genai.ClientConfig{
		APIKey:  cfg.apiKey,
		Backend: cfg.backend,
	}
	if cfg.project != "" {
		cc.Project = cfg.project
		cc.Location = cfg.region
	}

	inner, err := genai.NewClient(ctx, cc)
	if err != nil {
		return nil, err
	}

	return &Client{inner: inner, model: cfg.model}, nil
}

// ModelID returns the default model configured for this client.
func (c *Client) ModelID() string { return c.model }

// Chat performs a single-turn, non-streaming LLM call.
func (c *Client) Chat(ctx context.Context, req core.Request) (core.Response, error) {
	contents, cfg := toContentsAndConfig(req)

	resp, err := c.inner.Models.GenerateContent(ctx, c.modelID(req), contents, cfg)
	if err != nil {
		return core.Response{}, mapError(err)
	}

	return fromResponse(resp), nil
}

// Stream performs a streaming LLM call and returns an iterator of [core.Event].
// The iterator yields events until [core.EventDone] or [core.EventError].
// Breaking the iterator early is safe — no goroutine is leaked.
func (c *Client) Stream(ctx context.Context, req core.Request) iter.Seq2[core.Event, error] {
	return func(yield func(core.Event, error) bool) {
		contents, cfg := toContentsAndConfig(req)

		var accumulated string
		for resp, err := range c.inner.Models.GenerateContentStream(ctx, c.modelID(req), contents, cfg) {
			if err != nil {
				mapped := mapError(err)
				yield(core.Event{Kind: core.EventError, Err: mapped}, mapped)
				return
			}

			delta := resp.Text()
			if delta != "" {
				accumulated += delta
				if !yield(core.Event{
					Kind:  core.EventText,
					Delta: delta,
					Text:  accumulated,
				}, nil) {
					return
				}
			}
		}

		yield(core.Event{Kind: core.EventDone}, nil)
	}
}

// modelID retorna o modelo a usar: o do request tem prioridade sobre o padrão do client.
func (c *Client) modelID(req core.Request) string {
	if req.Model != "" {
		return req.Model
	}
	return c.model
}

// firstEnv retorna o valor da primeira variável de ambiente definida.
func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}
