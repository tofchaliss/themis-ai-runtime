package skills

// The v1 input schema: a deliberately dumb closed vocabulary
// (D-L9-4). Types string|integer|boolean, required, max_length, enum
// — and nothing else. Rich features (regex, $ref, conditionals,
// defaults, coercions, expressions) are unrepresentable: the structs
// have no fields for them and unknown JSON keys refuse. The validator
// performs only bounded structural checks; no computation hides in a
// schema.

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type FieldType string

const (
	TypeString  FieldType = "string"
	TypeInteger FieldType = "integer"
	TypeBoolean FieldType = "boolean"
)

type Field struct {
	Name      string    `json:"name"`
	Type      FieldType `json:"type"`
	Required  bool      `json:"required,omitempty"`
	MaxLength int       `json:"max_length,omitempty"`
	Enum      []string  `json:"enum,omitempty"`
}

type InputSchema struct {
	Version int     `json:"version"`
	Fields  []Field `json:"fields"`
}

const (
	maxSchemaFields = 32
	maxEnumValues   = 32
	maxFieldLength  = 65536
	// Integer inputs are bounded: the v1 vocabulary has no range
	// dimension, so the validator supplies a structural bound rather
	// than admitting an unbounded value.
	minIntegerInput = -1 << 53
	maxIntegerInput = 1 << 53
)

// fieldNameSyntax: lowercase segments joined by "-" or "_", matched
// verbatim (no normalization — see parseSchema).
var fieldNameSyntax = regexp.MustCompile(`^[a-z0-9]+([-_][a-z0-9]+)*$`)

func parseSchema(raw []byte) (*InputSchema, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var s InputSchema
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInputs, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: trailing content", ErrInputs)
	}
	if s.Version < 1 || len(s.Fields) == 0 {
		return nil, fmt.Errorf("%w: version and at least one field required", ErrInputs)
	}
	if len(s.Fields) > maxSchemaFields {
		return nil, fmt.Errorf("%w: %d fields exceeds the cap %d", ErrInputs, len(s.Fields), maxSchemaFields)
	}
	seen := map[string]bool{}
	for _, f := range s.Fields {
		// One field-name syntax, no normalization: conflating "_" and
		// "-" would let two visually distinct names pass the duplicate
		// check while reading as the same field to a human reviewer.
		if !fieldNameSyntax.MatchString(f.Name) {
			return nil, fmt.Errorf("%w: bad field name %q", ErrInputs, f.Name)
		}
		if seen[f.Name] {
			return nil, fmt.Errorf("%w: duplicate field %q", ErrInputs, f.Name)
		}
		seen[f.Name] = true
		switch f.Type {
		case TypeString, TypeInteger, TypeBoolean:
		default:
			return nil, fmt.Errorf("%w: field %q: unknown type %q — the v1 vocabulary is closed", ErrInputs, f.Name, f.Type)
		}
		// Nothing is defaulted: a string field states its own bound.
		if f.Type == TypeString && f.MaxLength <= 0 {
			return nil, fmt.Errorf("%w: field %q: string fields must declare a positive max_length — nothing is defaulted", ErrInputs, f.Name)
		}
		if f.Type != TypeString && f.MaxLength != 0 {
			return nil, fmt.Errorf("%w: field %q: max_length applies to string fields only", ErrInputs, f.Name)
		}
		if f.MaxLength < 0 || f.MaxLength > maxFieldLength {
			return nil, fmt.Errorf("%w: field %q: max_length out of bounds", ErrInputs, f.Name)
		}
		if len(f.Enum) > maxEnumValues {
			return nil, fmt.Errorf("%w: field %q: enum exceeds the cap", ErrInputs, f.Name)
		}
		if len(f.Enum) > 0 && f.Type != TypeString {
			return nil, fmt.Errorf("%w: field %q: enum applies to string fields only", ErrInputs, f.Name)
		}
	}
	return &s, nil
}

// LoadInputSchema is the standalone loader (registers/tests).
func LoadInputSchema(path string) (*InputSchema, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInputs, err)
	}
	return parseSchema(raw)
}

// Validate checks caller inputs against the schema: type, required,
// length bound, enum membership — bounded structural checks only.
// Unknown input fields refuse (closed world); nothing is defaulted.
func (s *InputSchema) Validate(inputs map[string]any) error {
	byName := map[string]Field{}
	for _, f := range s.Fields {
		byName[f.Name] = f
	}
	for name := range inputs {
		if _, ok := byName[name]; !ok {
			return fmt.Errorf("%w: unknown input %q — the input surface is closed", ErrInputs, name)
		}
	}
	for _, f := range s.Fields {
		v, present := inputs[f.Name]
		if !present {
			if f.Required {
				return fmt.Errorf("%w: missing required input %q — nothing is defaulted", ErrInputs, f.Name)
			}
			continue
		}
		switch f.Type {
		case TypeString:
			str, ok := v.(string)
			if !ok {
				return fmt.Errorf("%w: input %q must be a string", ErrInputs, f.Name)
			}
			// The schema's declared bound, never a defaulted one:
			// parseSchema refuses a string field without one.
			if len(str) > f.MaxLength {
				return fmt.Errorf("%w: input %q exceeds max_length %d", ErrInputs, f.Name, f.MaxLength)
			}
			if len(f.Enum) > 0 {
				ok := false
				for _, e := range f.Enum {
					if str == e {
						ok = true
					}
				}
				if !ok {
					return fmt.Errorf("%w: input %q is not an allowed enum value", ErrInputs, f.Name)
				}
			}
		case TypeInteger:
			// Integral in every representation: a Go caller's int and
			// a JSON-decoded float64 are held to the same rule, and
			// the value is bounded so an integer input cannot be
			// unbounded (the v1 vocabulary has no range dimension).
			var n int64
			switch t := v.(type) {
			case int:
				n = int64(t)
			case int64:
				n = t
			case float64:
				if t != float64(int64(t)) {
					return fmt.Errorf("%w: input %q must be an integer", ErrInputs, f.Name)
				}
				n = int64(t)
			default:
				return fmt.Errorf("%w: input %q must be an integer", ErrInputs, f.Name)
			}
			if n < minIntegerInput || n > maxIntegerInput {
				return fmt.Errorf("%w: input %q is outside the bounded integer range", ErrInputs, f.Name)
			}
		case TypeBoolean:
			if _, ok := v.(bool); !ok {
				return fmt.Errorf("%w: input %q must be a boolean", ErrInputs, f.Name)
			}
		}
	}
	return nil
}
