// Package anthropic implements core.Client using the official Anthropic Go SDK.
//
// # Installation
//
//	go get github.com/brunoochoa/goai/providers/anthropic
//
// # Authentication
//
// Set ANTHROPIC_API_KEY in the environment, or pass [WithAPIKey] explicitly.
// The client returns an error at construction time if no key is found.
//
// # Example
//
//	client := anthropic.NewClient()
//	resp, err := client.Chat(ctx, core.Request{
//	    Messages: []core.Message{core.UserMessage("Olá!")},
//	})
package anthropic

import (
	"context"
	"errors"
	"iter"
	"os"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/brunoochoa/goai/core"
)

const defaultModel = "claude-sonnet-4-5"

// Client implements [core.Client] for the Anthropic API.
// Construct with [NewClient]; the zero value is not usable.
type Client struct {
	inner   sdk.Client
	model   string
	maxToks int64
}

// Option configures a [Client].
type Option func(*clientConfig)

type clientConfig struct {
	apiKey  string
	baseURL string
	model   string
	maxToks int64
}

// WithAPIKey sets the Anthropic API key explicitly.
// Overrides the ANTHROPIC_API_KEY environment variable.
func WithAPIKey(k string) Option { return func(c *clientConfig) { c.apiKey = k } }

// WithBaseURL overrides the API endpoint (useful for testing with httptest).
func WithBaseURL(u string) Option { return func(c *clientConfig) { c.baseURL = u } }

// WithModel sets the default model for all calls on this client.
// Default: "claude-sonnet-4-5".
func WithModel(m string) Option { return func(c *clientConfig) { c.model = m } }

// WithMaxTokens sets the maximum output tokens per request.
// Default: 8192.
func WithMaxTokens(n int) Option { return func(c *clientConfig) { c.maxToks = int64(n) } }

// NewClient creates an Anthropic client.
// Returns an error if no API key is available (env or option).
func NewClient(opts ...Option) (*Client, error) {
	cfg := &clientConfig{
		apiKey:  os.Getenv("ANTHROPIC_API_KEY"),
		model:   defaultModel,
		maxToks: 8192,
	}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.apiKey == "" && cfg.baseURL == "" {
		return nil, errors.New("anthropic: API key ausente — defina ANTHROPIC_API_KEY ou use WithAPIKey()")
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
	params, err := toParams(req, c.model, c.maxToks)
	if err != nil {
		return core.Response{}, err
	}

	msg, err := c.inner.Messages.New(ctx, params)
	if err != nil {
		return core.Response{}, mapError(err)
	}

	return fromMessage(msg), nil
}

// Stream performs a streaming LLM call and returns an iterator of [core.Event].
// The iterator yields events until [core.EventDone] or [core.EventError].
// Breaking the iterator early is safe — no goroutine is leaked.
func (c *Client) Stream(ctx context.Context, req core.Request) iter.Seq2[core.Event, error] {
	return func(yield func(core.Event, error) bool) {
		params, err := toParams(req, c.model, c.maxToks)
		if err != nil {
			yield(core.Event{Kind: core.EventError, Err: err}, err)
			return
		}

		stream := c.inner.Messages.NewStreaming(ctx, params)
		var accumulated string

		for stream.Next() {
			ev, skip := fromStreamEvent(stream.Current(), &accumulated)
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
