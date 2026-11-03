package indexeradmin

import (
	"context"
	"fmt"
	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"time"
)

// This session makes configuration validation demonstrably independent of
// networking, authentication, rate limits and an SQL write transaction.
type validationIndexerSession struct{}

func (validationIndexerSession) Do(context.Context, cardigann.Request) (cardigann.HTTPResponse, error) {
	return cardigann.HTTPResponse{}, indexer.ErrInvalidSettings
}
func (validationIndexerSession) SetCookieHeader(string, string) error {
	return indexer.ErrInvalidSettings
}
func ValidateSettings(plan, old *cardigann.Plan, instance config.BuiltinIndexerInstance) error {
	values := map[string]cardigann.Scalar{}
	for key, value := range instance.Settings {
		text, err := (indexer.SettingValue{String: value.String, Boolean: value.Boolean, Number: value.Number}).Text()
		if err != nil {
			return fmt.Errorf("invalid setting type")
		}
		values[key] = cardigann.Scalar(text)
	}
	oldSettings := map[string]cardigann.Setting{}
	if old != nil {
		for _, setting := range old.Definition.Settings {
			oldSettings[setting.Name] = setting
		}
	}
	for _, setting := range plan.Definition.Settings {
		if configured, ok := instance.Settings[setting.Name]; ok {
			switch setting.Type {
			case "checkbox":
				if configured.Boolean == nil {
					return fmt.Errorf("setting %s requires boolean", setting.Name)
				}
			case "text", "password":
				if configured.String == nil {
					return fmt.Errorf("setting %s requires text", setting.Name)
				}
			case "select":
				if configured.Boolean != nil {
					return fmt.Errorf("setting %s requires selection", setting.Name)
				}
			}
		}
		if old != nil {
			prior, existed := oldSettings[setting.Name]
			if existed && prior.Type != setting.Type {
				return fmt.Errorf("setting %s changed type", setting.Name)
			}
			if !existed && (setting.Type == "text" || setting.Type == "password") && setting.Default == "" {
				if _, confirmed := instance.Settings[setting.Name]; !confirmed {
					return fmt.Errorf("setting %s requires configuration review", setting.Name)
				}
			}
		}
	}
	if _, err := plan.NewExecution(validationIndexerSession{}, instance.BaseURL, values, time.Now()); err != nil {
		return fmt.Errorf("definition settings are incompatible")
	}
	return nil
}
