// Exemplo básico: chat multi-turn com escolha de provider via flag de linha de comando.
//
// Uso:
//
//	ANTHROPIC_API_KEY=... go run ./examples/chat
//	OPENAI_API_KEY=...    go run ./examples/chat -provider openai
//	GEMINI_API_KEY=...    go run ./examples/chat -provider gemini
//	                      go run ./examples/chat -provider ollama -model llama3.2
//	GROQ_API_KEY=...      go run ./examples/chat -provider groq
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/brunoochoa/goai/chat"
	"github.com/brunoochoa/goai/core"
	anthropicprovider "github.com/brunoochoa/goai/providers/anthropic"
	compatprovider "github.com/brunoochoa/goai/providers/compat"
	geminiprovider "github.com/brunoochoa/goai/providers/gemini"
	openaiprovider "github.com/brunoochoa/goai/providers/openai"
)

func main() {
	provider := flag.String("provider", "anthropic", "provider: anthropic | openai | gemini | ollama | groq")
	model := flag.String("model", "", "modelo (vazio = padrão do provider)")
	stream := flag.Bool("stream", true, "habilitar streaming de resposta")
	flag.Parse()

	ctx := context.Background()

	client, err := buildClient(ctx, *provider, *model)
	if err != nil {
		log.Fatalf("erro ao criar client: %v", err)
	}

	session := chat.New(client,
		chat.WithSystem("Você é um assistente prestativo e conciso. Responda sempre em português."),
	)

	fmt.Printf("goai chat — provider: %s | modelo: %s | streaming: %v\n",
		*provider, client.ModelID(), *stream)
	fmt.Println("Digite sua mensagem. Ctrl+C para sair.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("você: ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if input == "" {
			continue
		}

		fmt.Print("assistente: ")
		if *stream {
			runStream(ctx, session, input)
		} else {
			runChat(ctx, session, input)
		}
		fmt.Println()
	}
}

// buildClient cria um core.Client para o provider escolhido.
func buildClient(ctx context.Context, provider, model string) (core.Client, error) {
	switch provider {
	case "openai":
		opts := []openaiprovider.Option{}
		if model != "" {
			opts = append(opts, openaiprovider.WithModel(model))
		}
		return openaiprovider.NewClient(opts...)

	case "gemini":
		opts := []geminiprovider.Option{}
		if model != "" {
			opts = append(opts, geminiprovider.WithModel(model))
		}
		return geminiprovider.NewClient(ctx, opts...)

	case "ollama":
		opts := []openaiprovider.Option{}
		if model != "" {
			opts = append(opts, openaiprovider.WithModel(model))
		}
		return compatprovider.NewOllama(opts...)

	case "groq":
		opts := []openaiprovider.Option{
			openaiprovider.WithAPIKey(os.Getenv("GROQ_API_KEY")),
		}
		if model != "" {
			opts = append(opts, openaiprovider.WithModel(model))
		}
		return compatprovider.NewGroq(opts...)

	default: // anthropic
		opts := []anthropicprovider.Option{}
		if model != "" {
			opts = append(opts, anthropicprovider.WithModel(model))
		}
		return anthropicprovider.NewClient(opts...)
	}
}

func runChat(ctx context.Context, session *chat.Session, input string) {
	msg, err := session.Send(ctx, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nerro: %v\n", err)
		return
	}
	fmt.Print(msg.Text())
}

func runStream(ctx context.Context, session *chat.Session, input string) {
	for ev, err := range session.Stream(ctx, input) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerro: %v\n", err)
			return
		}
		switch ev.Kind {
		case core.EventText:
			fmt.Print(ev.Delta)
		case core.EventDone:
			return
		case core.EventError:
			fmt.Fprintf(os.Stderr, "\nerro no stream: %v\n", ev.Err)
			return
		}
	}
}
