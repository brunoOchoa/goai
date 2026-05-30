// Package openai implements core.Client using the official OpenAI Go SDK (v3).
//
// # Installation
//
//	go get github.com/brunoochoa/goai/providers/openai
//
// # Authentication
//
// Set OPENAI_API_KEY in the environment, or pass [WithAPIKey] explicitly.
// The client returns an error at construction time if no key is found.
//
// # Example
//
//	client, err := openai.NewClient()
//	resp, err := client.Chat(ctx, core.Request{
//	    Messages: []core.Message{core.UserMessage("Olá!")},
//	})
package openai

import (
	"context"
	"errors"
	"iter"
	"os"

	sdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/brunoochoa/goai/core"
)

const defaultModel = "gpt-4o"

// Client implements [core.Client] for the OpenAI API (and OpenAI-compatible endpoints).
// Construct with [NewClient]; the zero value is not usable.
type Client struct {
	inner   sdk.Client
	model   string
	maxToks int
}

// Option configures a [Client].
type Option func(*clientConfig)

type clientConfig struct {
	apiKey  string
	baseURL string
	model   string
	maxToks int
}

// WithAPIKey sets the OpenAI API key explicitly.
// Overrides the OPENAI_API_KEY environment variable.
func WithAPIKey(k string) Option { return func(c *clientConfig) { c.apiKey = k } }

// WithBaseURL overrides the API endpoint.
// Use this for OpenAI-compatible providers (Ollama, Groq, etc.) or for testing.
func WithBaseURL(u string) Option { return func(c *clientConfig) { c.baseURL = u } }

// WithModel sets the default model for all calls on this client.
// Default: "gpt-4o".
func WithModel(m string) Option { return func(c *clientConfig) { c.model = m } }

// WithMaxTokens sets the maximum output tokens per request.
// Default: 8192.
func WithMaxTokens(n int) Option { return func(c *clientConfig) { c.maxToks = n } }

// NewClient creates an OpenAI client.
// Returns an error if no API key is available and no custom base URL is set
// (OpenAI-compatible providers like Ollama don't require a real key).
func NewClient(opts ...Option) (*Client, error) {
	cfg := &clientConfig{
		apiKey:  os.Getenv("OPENAI_API_KEY"),
		model:   defaultModel,
		maxToks: 8192,
	}
	for _, o := range opts {
		o(cfg)
	}

	// Providers com baseURL customizado (Ollama, Groq...) podem não precisar de key real.
	if cfg.apiKey == "" && cfg.baseURL == "" {
		return nil, errors.New("openai: API key ausente — defina OPENAI_API_KEY ou use WithAPIKey()")
	}

	sdkOpts := []option.RequestOption{option.WithAPIKey(cfg.apiKey)}
	if cfg.baseURL != "" {
		sdkOpts = append(sdkOpts, option.WithBaseURL(cfg.baseURL))
	}

	return &Client{
		inner:   sdk.NewClient(sdkOpts...),
		model:   cfg.model,
		maxToks: cfg.maxToks,
	}, nil
}

// ModelID returns the default model configured for this client.
func (c *Client) ModelID() string { return c.model }

// Chat performs a single-turn, non-streaming LLM call.
func (c *Client) Chat(ctx context.Context, req core.Request) (core.Response, error) {
	params := toParams(req, c.model, c.maxToks)

	completion, err := c.inner.Chat.Completions.New(ctx, params)
	if err != nil {
		return core.Response{}, mapError(err)
	}

	return fromCompletion(completion), nil
}

// Stream performs a streaming LLM call and returns an iterator of [core.Event].
// The iterator yields events until [core.EventDone] or [core.EventError].
// Breaking the iterator early is safe — no goroutine is leaked.
func (c *Client) Stream(ctx context.Context, req core.Request) iter.Seq2[core.Event, error] {
	return func(yield func(core.Event, error) bool) {
		params := toParams(req, c.model, c.maxToks)

		stream := c.inner.Chat.Completions.NewStreaming(ctx, params)
		var accumulated string

		for stream.Next() {
			chunk := stream.Current()
			ev, skip := fromStreamChunk(chunk, &accumulated)
			if skip {
				continue
			}
			if !yield(ev, nil) {
				return
			}
		}

		if err := stream.Err(); err != nil {
			mapped := mapError(err)
			yield(core.Event{Kind: core.EventError, Err: mapped}, mapped)
			return
		}

		yield(core.Event{Kind: core.EventDone}, nil)
	}
}
