package core

import (
	"context"
	"iter"
)

// StepKind classifica um passo de raciocínio do agente.
type StepKind uint8

const (
	StepThought     StepKind = iota // raciocínio interno
	StepAction                      // invocação de tool
	StepObservation                 // resultado da tool
	StepFinal                       // resposta final
)

// Step é uma unidade do trace de execução do agente.
type Step struct {
	Kind       StepKind
	Content    string
	ToolCall   *ToolCall
	ToolResult *ToolResult
}

// AgentResult é a resposta final produzida pelo agente.
type AgentResult struct {
	Output string
	Steps  []Step
	Usage  Usage
}

// Agent é a interface de alto nível para execução autônoma de tarefas.
type Agent interface {
	Run(ctx context.Context, input string) (AgentResult, error)
	Stream(ctx context.Context, input string) iter.Seq2[Step, error]
}
