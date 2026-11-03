package indexeradmin

import (
	"testing"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
)

func TestSelectProjectionPreservesEmptyAndDeclaredDefaults(t *testing.T) {
	for _, defaultValue := range []cardigann.Scalar{"", "b"} {
		plan := &cardigann.Plan{Definition: cardigann.Definition{Settings: []cardigann.Setting{{
			Name: "choice", Type: "select", Default: defaultValue,
			Options: map[string]cardigann.Scalar{"b": "Second declared first", "a": "First sorted last"},
		}}}}
		empty := ""
		for _, settings := range []map[string]config.IndexerSettingValue{nil, {"choice": {String: &empty}}} {
			projected := settingProjection(plan, settings)[0]
			if projected.Default == nil || projected.Default.String == nil || *projected.Default.String != string(defaultValue) {
				t.Fatalf("default %q became %+v", defaultValue, projected.Default)
			}
			if settings != nil && (projected.Value == nil || projected.Value.String == nil || *projected.Value.String != "") {
				t.Fatalf("explicit empty selection was lost: %+v", projected.Value)
			}
		}
	}
}
