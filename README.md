# goai

Framework de IA em Go — limpo, leve, rápido e sem dependências no núcleo.

Suporta múltiplos providers (Anthropic, OpenAI, Gemini, Ollama, Groq e qualquer endpoint OpenAI-compatível) através de uma interface unificada. Troque de provider sem mudar uma linha do código de aplicação.

---

## Índice

- [Requisitos](#requisitos)
- [Instalação](#instalação)
- [Variáveis de Ambiente](#variáveis-de-ambiente)
- [Uso Básico](#uso-básico)
- [Providers](#providers)
- [Chat com Histórico](#chat-com-histórico)
- [Streaming](#streaming)
- [Tool Calling](#tool-calling)
- [Segurança](#segurança)
- [Testes](#testes)
- [Estrutura do Projeto](#estrutura-do-projeto)

---

## Requisitos

- Go 1.26 ou superior
- Uma API key do provider escolhido

---

## Instalação

O módulo raiz (`github.com/brunoochoa/goai`) contém as interfaces core e o pacote `chat`.  
Cada provider é um módulo separado — instale apenas os que precisar.

```bash
# Núcleo + chat (zero dependências externas)
go get github.com/brunoochoa/goai

# Provider Anthropic (Claude)
go get github.com/brunoochoa/goai/providers/anthropic

# Provider OpenAI (GPT)
go get github.com/brunoochoa/goai/providers/openai

# Provider Gemini (Google)
go get github.com/brunoochoa/goai/providers/gemini

# Providers compatíveis (Ollama, Groq, Mistral, Together, DeepSeek)
go get github.com/brunoochoa/goai/providers/compat
```

---

## Variáveis de Ambiente

| Variável           | Provider   | Obrigatória |
|--------------------|------------|-------------|
| `ANTHROPIC_API_KEY`| Anthropic  | Sim         |
| `OPENAI_API_KEY`   | OpenAI     | Sim         |
| `GEMINI_API_KEY`   | Gemini     | Sim (ou `GOOGLE_API_KEY`) |
| `GROQ_API_KEY`     | Groq       | Sim         |
| `MISTRAL_API_KEY`  | Mistral    | Sim         |
| `DEEPSEEK_API_KEY` | DeepSeek   | Sim         |

> **Ollama** não requer API key (roda localmente).

### Boas práticas para API keys

```bash
# Use um arquivo .env e carregue via godotenv ou similar
# NUNCA commite keys no repositório
# Use variáveis de ambiente em CI/CD (GitHub Secrets, etc.)
```

---

## Uso Básico

Todos os providers implementam `core.Client`. O código de aplicação usa apenas as interfaces do pacote `core`.

### Anthropic (Claude)

```go
import (
    "github.com/brunoochoa/goai/core"
    "github.com/brunoochoa/goai/providers/anthropic"
)

client, err := anthropic.NewClient()
// ou: anthropic.NewClient(anthropic.WithModel("claude-opus-4-5"))
if err != nil {
    log.Fatal(err)
}

resp, err := client.Chat(ctx, core.Request{
    System:   "Você é um assistente útil.",
    Messages: []core.Message{core.UserMessage("O que é Go?")},
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Message.Text())
```

### OpenAI (GPT)

```go
import "github.com/brunoochoa/goai/providers/openai"

client, err := openai.NewClient()
// ou: openai.NewClient(openai.WithModel("gpt-4o-mini"))
```

### Gemini (Google)

```go
import "github.com/brunoochoa/goai/providers/gemini"

client, err := gemini.NewClient(ctx)
// ou: gemini.NewClient(ctx, gemini.WithModel("gemini-2.5-pro"))

// Vertex AI:
client, err := gemini.NewClient(ctx,
    gemini.WithVertexAI("meu-projeto", "us-central1"),
)
```

### Trocar de provider sem mudar o código

```go
func buildClient(provider string) (core.Client, error) {
    switch provider {
    case "openai":
        return openai.NewClient()
    case "gemini":
        return gemini.NewClient(ctx)
    default:
        return anthropic.NewClient()
    }
}

// O resto do código usa core.Client — provider-agnóstico.
func doChat(client core.Client, question string) (string, error) {
    resp, err := client.Chat(ctx, core.Request{
        Messages: []core.Message{core.UserMessage(question)},
    })
    if err != nil {
        return "", err
    }
    return resp.Message.Text(), nil
}
```

---

## Providers

### Ollama (local)

```go
import "github.com/brunoochoa/goai/providers/compat"

// Requer Ollama rodando: https://ollama.com
client, err := compat.NewOllama()
// ou com modelo específico:
client, err := compat.NewOllama(openai.WithModel("llama3.2"))
```

### Groq

```go
client, err := compat.NewGroq(
    openai.WithAPIKey(os.Getenv("GROQ_API_KEY")),
)
```

### Endpoint personalizado

```go
// Qualquer API compatível com OpenAI
client, err := compat.NewCustom("http://localhost:8080/v1",
    openai.WithAPIKey("minha-key"),
    openai.WithModel("meu-modelo"),
)
```

---

## Chat com Histórico

O pacote `chat` gerencia histórico automaticamente:

```go
import "github.com/brunoochoa/goai/chat"

client, _ := anthropic.NewClient()

session := chat.New(client,
    chat.WithSystem("Você é um assistente de programação Go."),
    chat.WithMaxTokens(4096),
)

// Cada Send/Stream mantém o histórico automaticamente.
msg, err := session.Send(ctx, "O que são goroutines?")
fmt.Println(msg.Text())

msg, err = session.Send(ctx, "Pode dar um exemplo?")
// O histórico da pergunta anterior é enviado automaticamente.
fmt.Println(msg.Text())

// Ver histórico atual
history, _ := session.History(ctx)
fmt.Printf("%d mensagens no histórico\n", len(history))

// Limpar histórico
session.Reset(ctx)
```

> **Session é thread-safe** — pode ser usada por múltiplas goroutines simultaneamente.

---

## Streaming

```go
for ev, err := range client.Stream(ctx, core.Request{
    Messages: []core.Message{core.UserMessage("Conte uma história curta.")},
}) {
    if err != nil {
        log.Fatal(err)
    }
    switch ev.Kind {
    case core.EventText:
        fmt.Print(ev.Delta) // imprime chunk por chunk
    case core.EventDone:
        fmt.Println() // fim do stream
        return
    case core.EventUsage:
        fmt.Printf("[tokens: in=%d out=%d]\n",
            ev.Usage.InputTokens, ev.Usage.OutputTokens)
    }
}
```

### Streaming com Session

```go
for ev, err := range session.Stream(ctx, "Me explique channels em Go.") {
    if err != nil { log.Fatal(err) }
    if ev.Kind == core.EventText {
        fmt.Print(ev.Delta)
    }
}
// Histórico salvo automaticamente após EventDone.
```

---

## Tool Calling

```go
import "github.com/brunoochoa/goai/core/schema"

// 1. Defina o schema de entrada
type WeatherInput struct {
    City    string `json:"city"    description:"Nome da cidade"           jsonschema:"required"`
    Country string `json:"country" description:"Código do país (ex: BR)"`
}

// 2. Implemente AnyTool
type WeatherTool struct{}

func (t *WeatherTool) Name() string        { return "get_weather" }
func (t *WeatherTool) Description() string { return "Retorna o clima atual de uma cidade." }
func (t *WeatherTool) Schema() json.RawMessage {
    return schema.MustFromStruct[WeatherInput]()
}
func (t *WeatherTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
    var input WeatherInput
    if err := json.Unmarshal(args, &input); err != nil {
        return "", err
    }
    // Validar entrada antes de chamar APIs externas
    if input.City == "" {
        return "", errors.New("cidade é obrigatória")
    }
    return fmt.Sprintf("Clima em %s: 25°C, ensolarado.", input.City), nil
}

// 3. Passe a tool no Request
resp, err := client.Chat(ctx, core.Request{
    Messages: []core.Message{
        core.UserMessage("Qual o clima em São Paulo?"),
    },
    Tools: []core.AnyTool{&WeatherTool{}},
})

// 4. Verifique se o modelo quer chamar a tool
if len(resp.Message.ToolCalls) > 0 {
    tc := resp.Message.ToolCalls[0]
    result, err := (&WeatherTool{}).Execute(ctx, tc.Arguments)
    // ... envie o resultado de volta ao modelo
}
```

---

## Segurança

### API Keys

- **Nunca** hardcode API keys no código ou em arquivos versionados.
- Use variáveis de ambiente ou um gerenciador de segredos (Vault, AWS Secrets Manager, etc.).
- O `NewClient` de cada provider falha explicitamente se a key estiver ausente.
- Em logs, nunca imprima a `core.Request` completa (pode conter dados sensíveis do usuário).

### Tool Calling

- **Valide sempre** os argumentos recebidos do modelo antes de executar side effects.
  O modelo pode gerar argumentos inesperados ou maliciosos.
- Aplique allow-lists para parâmetros críticos (ex: paths de arquivo, queries SQL).
- Prefira retornar `(result, error)` a fazer panic em tools de produção.

### Erros

Use `errors.Is` para tratar erros de forma segura:

```go
resp, err := client.Chat(ctx, req)
if err != nil {
    switch {
    case errors.Is(err, core.ErrRateLimit):
        // retry com backoff
    case errors.Is(err, core.ErrAuthFailed):
        // credencial inválida — não tente novamente
    case errors.Is(err, core.ErrContextLength):
        // reduza o histórico ou use summarization
    default:
        log.Printf("erro inesperado: %v", err)
    }
}
```

### Concorrência

- `chat.Session` é **thread-safe**.
- `core.Client` (providers) é **thread-safe** — pode ser compartilhado entre goroutines.
- Iteradores de stream são **single-consumer** — não compartilhe entre goroutines.

---

## Testes

### Pré-requisitos

```bash
go test ./...  # testa o módulo raiz (sem API key necessária)
```

### Testes de integração (requerem API keys reais)

```bash
# Define as keys e rode com a tag de integração
ANTHROPIC_API_KEY=... go test -tags integration ./providers/anthropic/...
OPENAI_API_KEY=...    go test -tags integration ./providers/openai/...
GEMINI_API_KEY=...    go test -tags integration ./providers/gemini/...
```

### Testes com httptest (sem API key)

Os providers suportam `WithBaseURL` para apontar para um servidor `httptest.NewServer`:

```go
func TestChat(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
            "content": []map[string]any{{"type": "text", "text": "olá!"}},
            "model":   "claude-sonnet-4-5",
            "usage":   map[string]int{"input_tokens": 10, "output_tokens": 5},
        })
    }))
    defer srv.Close()

    client, err := anthropic.NewClient(
        anthropic.WithAPIKey("test"),
        anthropic.WithBaseURL(srv.URL),
    )
    require.NoError(t, err)

    resp, err := client.Chat(t.Context(), core.Request{
        Messages: []core.Message{core.UserMessage("oi")},
    })
    require.NoError(t, err)
    assert.Equal(t, "olá!", resp.Message.Text())
}
```

---

## Estrutura do Projeto

```
goai/
├── go.work                    # workspace Go — une todos os módulos
├── go.mod                     # módulo raiz (zero deps externas)
│
├── core/                      # interfaces e tipos canônicos (zero deps)
│   ├── client.go              # ChatClient, StreamClient, Client
│   ├── message.go             # Message, Role, ContentPart, ToolCall, ToolResult
│   ├── event.go               # Event, EventKind, Usage (streaming)
│   ├── tool.go                # AnyTool, ToolRegistry
│   ├── memory.go              # Memory interface
│   ├── agent.go               # Agent, Step, AgentResult
│   ├── chain.go               # Chain, ChainStep
│   ├── mcp.go                 # MCPClient (Model Context Protocol)
│   ├── errors.go              # Sentinels + APIError
│   └── schema/                # Geração de JSON Schema via reflection
│
├── chat/
│   └── session.go             # Sessão multi-turn thread-safe
│
├── providers/
│   ├── anthropic/             # Claude (SDK v1.46.0)
│   ├── openai/                # GPT (SDK v3.37.0)
│   ├── gemini/                # Gemini (SDK v1.58.0)
│   └── compat/                # Ollama, Groq, Mistral, Together, DeepSeek
│
└── examples/
    └── chat/                  # CLI interativo multi-provider
```

### Princípios de design

| Princípio | Implementação |
|---|---|
| Zero deps no core | `go.mod` raiz não importa nada externo |
| Providers opt-in | Cada provider é um módulo separado |
| Thread-safe | `Session` usa `sync.RWMutex`; clients são stateless |
| Sem goroutine leak | Streaming usa `iter.Seq2` (pull model) |
| Erros explícitos | `NewClient` retorna `error`; sentinels para `errors.Is` |
| Troca de provider | Toda a aplicação usa `core.Client` — não tipos concretos |
