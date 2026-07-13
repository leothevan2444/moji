package performer

import (
	"encoding/json"
	"testing"
)

func TestIsSubscribedAcceptsStashTruthyRepresentations(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "boolean", value: true, want: true},
		{name: "numeric map scalar", value: float64(1), want: true},
		{name: "integer", value: 1, want: true},
		{name: "json number", value: json.Number("1"), want: true},
		{name: "string one", value: "1", want: true},
		{name: "string true", value: " true ", want: true},
		{name: "zero", value: float64(0), want: false},
		{name: "false", value: false, want: false},
		{name: "unknown string", value: "enabled", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := map[string]any{DefaultCustomFieldKey: tt.value}
			if got := IsSubscribed(fields, DefaultCustomFieldKey); got != tt.want {
				t.Fatalf("IsSubscribed(%T(%v)) = %v, want %v", tt.value, tt.value, got, tt.want)
			}
		})
	}
}
