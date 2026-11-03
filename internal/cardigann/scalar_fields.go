package cardigann

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ScalarFields retains mapping order for request inputs and category mappings.
// A loaded plan is immutable; runtime form overrides use separate formFields.
type ScalarField struct {
	Name  string
	Value Scalar
}

type ScalarFields []ScalarField

func (f *ScalarFields) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected scalar mapping", n.Line)
	}
	for i := 0; i < len(n.Content); i += 2 {
		var value Scalar
		if err := n.Content[i+1].Decode(&value); err != nil {
			return err
		}
		*f = append(*f, ScalarField{Name: n.Content[i].Value, Value: value})
	}
	return nil
}

func (f ScalarFields) Lookup(name string) (Scalar, bool) {
	for _, field := range f {
		if field.Name == name {
			return field.Value, true
		}
	}
	return "", false
}
