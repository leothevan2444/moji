package graphqlapi

import "github.com/leothevan2444/moji/internal/graphqlapi/model"

func parseGraphQLLogLevel(level string) model.LogLevel {
	switch level {
	case "debug":
		return model.LogLevelDebug
	case "warn", "warning":
		return model.LogLevelWarning
	case "error":
		return model.LogLevelError
	default:
		return model.LogLevelInfo
	}
}
