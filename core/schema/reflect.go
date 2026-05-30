package schema

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func reflectSchema(v any) (*Schema, error) {
	t := reflect.TypeOf(v)
	if t == nil {
		return &Schema{Type: "object"}, nil
	}
	return reflectType(t)
}

func reflectType(t reflect.Type) (*Schema, error) {
	// Dereferenciar ponteiro
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}, nil
	case reflect.Bool:
		return &Schema{Type: "boolean"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer"}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}, nil
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			// []byte → string base64
			return &Schema{Type: "string"}, nil
		}
		items, err := reflectType(t.Elem())
		if err != nil {
			return nil, err
		}
		return &Schema{Type: "array", Items: items}, nil
	case reflect.Map:
		return &Schema{Type: "object"}, nil
	case reflect.Struct:
		return reflectStructType(t)
	case reflect.Interface:
		return &Schema{}, nil // qualquer tipo
	default:
		return nil, fmt.Errorf("tipo não suportado: %s", t.Kind())
	}
}

func reflectStructType(t reflect.Type) (*Schema, error) {
	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := jsonFieldName(field)
		if name == "-" {
			continue
		}

		prop, err := reflectType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("campo %s: %w", field.Name, err)
		}

		applyTags(prop, field, &s.Required, name)
		s.Properties[name] = prop
	}

	return s, nil
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return strings.ToLower(f.Name)
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		return strings.ToLower(f.Name)
	}
	return parts[0]
}

func applyTags(s *Schema, f reflect.StructField, required *[]string, name string) {
	// tag description:"..."
	if desc := f.Tag.Get("description"); desc != "" {
		s.Description = desc
	}

	// tag jsonschema:"required,enum=a|b,minimum=0,maximum=100"
	jsTag := f.Tag.Get("jsonschema")
	if jsTag == "" {
		return
	}

	for _, part := range strings.Split(jsTag, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "required":
			*required = appendUnique(*required, name)
		case strings.HasPrefix(part, "enum="):
			vals := strings.Split(strings.TrimPrefix(part, "enum="), "|")
			for _, v := range vals {
				s.Enum = append(s.Enum, v)
			}
		case strings.HasPrefix(part, "minimum="):
			if v, err := strconv.ParseFloat(strings.TrimPrefix(part, "minimum="), 64); err == nil {
				s.Minimum = &v
			}
		case strings.HasPrefix(part, "maximum="):
			if v, err := strconv.ParseFloat(strings.TrimPrefix(part, "maximum="), 64); err == nil {
				s.Maximum = &v
			}
		}
	}
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
