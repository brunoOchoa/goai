// Package schema generates JSON Schema (draft 2020-12 subset) from Go struct types
// using reflection and struct tags. It has zero external dependencies.
//
// # Supported struct tags
//
//   - json:"name"            — field name in the schema (required)
//   - description:"..."      — populates the description field
//   - jsonschema:"required"  — marks the field as required
//   - jsonschema:"enum=a|b"  — adds an enum constraint
//   - jsonschema:"minimum=0,maximum=100" — numeric bounds
//
// # Example
//
//	type SearchInput struct {
//	    Query string `json:"query" description:"Search terms" jsonschema:"required"`
//	    Limit int    `json:"limit" jsonschema:"minimum=1,maximum=20"`
//	}
//
//	schema, err := schema.FromStruct[SearchInput]()
//	// or, for use at package init time:
//	raw := schema.MustFromStruct[SearchInput]()
package schema

import (
	"encoding/json"
	"fmt"

	"github.com/brunoochoa/goai/core"
)

// Schema represents a JSON Schema object (draft 2020-12 subset).
type Schema struct {
	Type        string             `json:"type,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Enum        []any              `json:"enum,omitempty"`
	Minimum     *float64           `json:"minimum,omitempty"`
	Maximum     *float64           `json:"maximum,omitempty"`
}

// FromStruct generates a Schema from the Go struct type T.
// T must be a struct type; returns [core.ErrSchemaGen] if not.
func FromStruct[T any]() (*Schema, error) {
	var zero T
	s, err := reflectSchema(zero)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", core.ErrSchemaGen, err)
	}
	return s, nil
}

// MustFromStruct generates a JSON Schema and returns it as a marshaled [json.RawMessage].
// It panics if schema generation or marshaling fails.
//
// Use this only in package-level variable initializers or init() functions,
// where a panic is acceptable (analogous to [regexp.MustCompile]).
// For runtime schema generation, use [FromStruct] instead.
func MustFromStruct[T any]() json.RawMessage {
	s, err := FromStruct[T]()
	if err != nil {
		panic(fmt.Sprintf("goai/schema: MustFromStruct failed: %v", err))
	}
	b, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("goai/schema: marshal failed: %v", err))
	}
	return b
}
