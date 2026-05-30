package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/brunoochoa/goai/core/schema"
)

type SearchInput struct {
	Query   string  `json:"query"   description:"termos de busca"           jsonschema:"required"`
	Limit   int     `json:"limit"   description:"máximo de resultados"       jsonschema:"minimum=1,maximum=100"`
	Safe    bool    `json:"safe"    description:"filtrar conteúdo sensível"`
	Score   float64 `json:"score"   description:"score mínimo de relevância"`
	Tags    []string `json:"tags"   description:"filtros por tag"`
}

func TestFromStruct(t *testing.T) {
	s, err := schema.FromStruct[SearchInput]()
	if err != nil {
		t.Fatalf("FromStruct: %v", err)
	}

	if s.Type != "object" {
		t.Errorf("tipo esperado 'object', got %q", s.Type)
	}

	if len(s.Properties) != 5 {
		t.Errorf("esperado 5 propriedades, got %d", len(s.Properties))
	}

	// query deve ser required
	found := false
	for _, r := range s.Required {
		if r == "query" {
			found = true
		}
	}
	if !found {
		t.Error("'query' deveria estar em Required")
	}

	// tipos
	checkType(t, s, "query", "string")
	checkType(t, s, "limit", "integer")
	checkType(t, s, "safe", "boolean")
	checkType(t, s, "score", "number")
	checkType(t, s, "tags", "array")

	// description
	if s.Properties["query"].Description != "termos de busca" {
		t.Errorf("description de query errada: %q", s.Properties["query"].Description)
	}

	// constraints numéricas
	if s.Properties["limit"].Minimum == nil || *s.Properties["limit"].Minimum != 1 {
		t.Error("Minimum de limit deveria ser 1")
	}
	if s.Properties["limit"].Maximum == nil || *s.Properties["limit"].Maximum != 100 {
		t.Error("Maximum de limit deveria ser 100")
	}
}

func TestMustFromStruct(t *testing.T) {
	raw := schema.MustFromStruct[SearchInput]()
	if len(raw) == 0 {
		t.Fatal("MustFromStruct retornou vazio")
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("json inválido: %v", err)
	}

	if m["type"] != "object" {
		t.Errorf("tipo esperado 'object', got %v", m["type"])
	}
}

func checkType(t *testing.T, s *schema.Schema, field, want string) {
	t.Helper()
	prop, ok := s.Properties[field]
	if !ok {
		t.Errorf("propriedade %q não encontrada", field)
		return
	}
	if prop.Type != want {
		t.Errorf("campo %q: tipo esperado %q, got %q", field, want, prop.Type)
	}
}
