package performer

import (
	"encoding/json"
	"strings"
)

// IsSubscribed reports whether the configured Stash custom field carries a
// truthy subscription marker. Stash's Map scalar may decode numeric values as
// float64, while existing data can also contain booleans, strings, or integers.
func IsSubscribed(fields map[string]any, key string) bool {
	value, ok := fields[key]
	if !ok {
		return false
	}

	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		}
	case json.Number:
		number, err := typed.Float64()
		return err == nil && number != 0
	case int:
		return typed != 0
	case int8:
		return typed != 0
	case int16:
		return typed != 0
	case int32:
		return typed != 0
	case int64:
		return typed != 0
	case uint:
		return typed != 0
	case uint8:
		return typed != 0
	case uint16:
		return typed != 0
	case uint32:
		return typed != 0
	case uint64:
		return typed != 0
	case float32:
		return typed != 0
	case float64:
		return typed != 0
	}

	return false
}
