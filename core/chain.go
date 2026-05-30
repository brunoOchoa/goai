package core

import "context"

// ChainInput e ChainOutput são map[string]any para máxima flexibilidade.
// Wrappers tipados ficam no pacote chain.
type ChainInput = map[string]any
type ChainOutput = map[string]any

// ChainStep é um nó em uma chain.
type ChainStep interface {
	Name() string
	Run(ctx context.Context, in ChainInput) (ChainOutput, error)
}

// Chain compõe múltiplos ChainSteps.
type Chain interface {
	ChainStep
	Steps() []ChainStep
}
