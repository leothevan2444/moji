package graphqlapi

import (
	"errors"
	"reflect"
	"strings"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
)

type settingsPatcher interface {
	PatchSettings(map[string]any) (*SettingsSnapshot, error)
}

func (r *mutationResolver) applySettingsPatch(input model.SettingsPatchInput) (*model.Settings, error) {
	editor, ok := r.SettingsEditor.(settingsPatcher)
	if !ok {
		return nil, errors.New("settings patch editor is not configured")
	}
	patch := settingsPatchValue(reflect.ValueOf(input)).(map[string]any)
	if len(patch) == 0 {
		return nil, config.ErrInvalidSettings
	}
	snapshot, err := editor.PatchSettings(patch)
	if err != nil {
		return nil, err
	}
	return settingsSnapshotToModel(snapshot, r.AppVersion), nil
}

// Omitted/null fields are unchanged, while empty non-nil lists must remain
// explicit replacements. encoding/json omitempty would incorrectly drop [].
func settingsPatchValue(value reflect.Value) any {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		return settingsPatchValue(value.Elem())
	}
	switch value.Kind() {
	case reflect.Struct:
		out := map[string]any{}
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			item := settingsPatchValue(value.Field(i))
			if item != nil {
				if object, ok := item.(map[string]any); ok && len(object) == 0 {
					continue
				}
				out[name] = item
			}
		}
		return out
	case reflect.Slice:
		if value.IsNil() {
			return nil
		}
		out := make([]any, value.Len())
		for i := range out {
			out[i] = settingsPatchValue(value.Index(i))
		}
		return out
	default:
		return value.Interface()
	}
}
