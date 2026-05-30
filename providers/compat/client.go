// Package compat provides pre-configured clients for OpenAI-compatible AI providers:
// Ollama (local), Groq, Together AI, Mistral, DeepSeek, and any custom endpoint.
//
// These clients reuse 100% of the openai provider implementation — only the
// base URL and default model differ. No code is duplicated.
//
// # Installation
//
//	go get github.com/brunoochoa/goai/providers/compat
//
// # Example: Ollama (local, no API key required)
//
//	client, _ := compat.NewOllama()
//	resp, err := client.Chat(ctx, core.Request{
//	    Messages: []core.Message{core.UserMessage("Olá!")},
//	})
//
// # Example: Groq
//
//	// Set GROQ_API_KEY in the environment first.
//	client, err := compat.NewGroq(openai.WithAPIKey(os.Getenv("GROQ_API_KEY")))
package compat

import (
	coreopenai "github.com/brunoochoa/goai/providers/openai"
)

// Known base URLs for popular OpenAI-compatible providers.
const (
	OllamaBaseURL   = "http://localhost:11434/v1"
	GroqBaseURL     = "https://api.groq.com/openai/v1"
	TogetherBaseURL = "https://api.together.xyz/v1"
	MistralBaseURL  = "https://api.mistral.ai/v1"
	DeepSeekBaseURL = "https://api.deepseek.com/v1"
)

// NewOllama creates a client for a local Ollama instance.
// Ollama does not require an API key; a placeholder is used automatically.
// Default model: "llama3.2". Override with [coreopenai.WithModel].
func NewOllama(opts ...coreopenai.Option) (*coreopenai.Client, error) {
	base := []coreopenai.Option{
		coreopenai.WithBaseURL(OllamaBaseURL),
		coreopenai.WithAPIKey("ollama"), // Ollama ignora o valor da key
		coreopenai.WithModel("llama3.2"),
	}
	return coreopenai.NewClient(append(base, opts...)...)
}

// NewGroq creates a client for the Groq API.
// Set GROQ_API_KEY in the environment or pass [coreopenai.WithAPIKey].
// Default model: "llama-3.3-70b-versatile".
func NewGroq(opts ...coreopenai.Option) (*coreopenai.Client, error) {
	base := []coreopenai.Option{
		coreopenai.WithBaseURL(GroqBaseURL),
		coreopenai.WithModel("llama-3.3-70b-versatile"),
	}
	return coreopenai.NewClient(append(base, opts...)...)
}

// NewTogether creates a client for Together AI.
// Set TOGETHER_API_KEY in the environment or pass [coreopenai.WithAPIKey].
// Default model: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo".
func NewTogether(opts ...coreopenai.Option) (*coreopenai.Client, error) {
	base := []coreopenai.Option{
		coreopenai.WithBaseURL(TogetherBaseURL),
		coreopenai.WithModel("meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"),
	}
	return coreopenai.NewClient(append(base, opts...)...)
}

// NewMistral creates a client for the Mistral AI API.
// Set MISTRAL_API_KEY in the environment or pass [coreopenai.WithAPIKey].
// Default model: "mistral-large-latest".
func NewMistral(opts ...coreopenai.Option) (*coreopenai.Client, error) {
	base := []coreopenai.Option{
		coreopenai.WithBaseURL(MistralBaseURL),
		coreopenai.WithModel("mistral-large-latest"),
	}
	return coreopenai.NewClient(append(base, opts...)...)
}

// NewDeepSeek creates a client for the DeepSeek API.
// Set DEEPSEEK_API_KEY in the environment or pass [coreopenai.WithAPIKey].
// Default model: "deepseek-chat".
func NewDeepSeek(opts ...coreopenai.Option) (*coreopenai.Client, error) {
	base := []coreopenai.Option{
		coreopenai.WithBaseURL(DeepSeekBaseURL),
		coreopenai.WithModel("deepseek-chat"),
	}
	return coreopenai.NewClient(append(base, opts...)...)
}

// NewCustom creates a client for any OpenAI-compatible endpoint.
// baseURL must include the path prefix (e.g., "http://host:port/v1").
func NewCustom(baseURL string, opts ...coreopenai.Option) (*coreopenai.Client, error) {
	base := []coreopenai.Option{coreopenai.WithBaseURL(baseURL)}
	return coreopenai.NewClient(append(base, opts...)...)
}
