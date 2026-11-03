package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

type IndexerAdminService interface {
	Definitions(context.Context, indexeradmin.DefinitionFilter) (indexeradmin.DefinitionPage, error)
	Instances(context.Context) (indexeradmin.InstancePage, error)
	SaveBuiltin(context.Context, string, string, indexeradmin.BuiltinPatch) (indexeradmin.SaveResult, error)
	RemoveBuiltin(context.Context, string, string) (indexeradmin.InstancePage, error)
	SetEnabled(context.Context, string, string, bool) (indexeradmin.InstancePage, error)
	Reorder(context.Context, string, []string) (indexeradmin.InstancePage, error)
	RefreshJackett(context.Context) (indexeradmin.InstancePage, error)
}

func settingValueModel(value *indexeradmin.Value) *model.IndexerSettingValue {
	if value == nil {
		return nil
	}
	return &model.IndexerSettingValue{String: value.String, Boolean: value.Boolean, Number: value.Number}
}
func indexerSettingModels(settings []indexeradmin.Setting) []*model.IndexerSetting {
	result := make([]*model.IndexerSetting, 0, len(settings))
	for _, setting := range settings {
		item := &model.IndexerSetting{Name: setting.Name, Type: setting.Type, Label: setting.Label, Secret: setting.Secret, Configured: setting.Configured, Options: make([]*model.IndexerSettingOption, 0, len(setting.Options))}
		if !setting.Secret {
			item.Value = settingValueModel(setting.Value)
			item.DefaultValue = settingValueModel(setting.Default)
		}
		for _, option := range setting.Options {
			item.Options = append(item.Options, &model.IndexerSettingOption{Value: option.Value, Label: option.Label})
		}
		result = append(result, item)
	}
	return result
}
func indexerInstanceModel(instance indexeradmin.Instance) *model.IndexerInstance {
	return &model.IndexerInstance{Health: &model.IndexerHealth{Search: indexerObservationModel(instance.Health.Search), Fetch: indexerObservationModel(instance.Health.Fetch)}, SiteURL: instance.SiteURL, Privacy: instance.Privacy, Language: instance.Language, ID: instance.ID, Kind: model.IndexerSourceKind(instance.Kind), Name: instance.Name, Enabled: instance.Enabled, DefinitionID: instance.DefinitionID, BaseURL: instance.BaseURL, NetworkProfile: instance.NetworkProfile, TrustedOrigins: instance.TrustedOrigins, RuleDigest: instance.RuleDigest, ErrorCode: instance.ErrorCode, Modes: instance.Modes, Settings: indexerSettingModels(instance.Settings)}
}
func indexerInstancesModel(page indexeradmin.InstancePage) *model.IndexerInstances {
	result := &model.IndexerInstances{HealthErrorCode: page.HealthErrorCode, Revision: page.Revision, RuleRevision: page.RuleRevision, RuleErrorCode: page.RuleErrorCode, CatalogErrorCode: page.CatalogErrorCode, JackettEnabled: page.JackettEnabled, Order: page.Order, NetworkProfiles: page.NetworkProfiles, Items: make([]*model.IndexerInstance, 0, len(page.Items))}
	for _, instance := range page.Items {
		result.Items = append(result.Items, indexerInstanceModel(instance))
	}
	return result
}
func indexerDefinitionsModel(page indexeradmin.DefinitionPage) *model.IndexerDefinitionPage {
	result := &model.IndexerDefinitionPage{Revision: page.Revision, TotalCount: page.Total, Offset: page.Offset, Limit: page.Limit, Items: make([]*model.IndexerDefinition, 0, len(page.Items))}
	for _, definition := range page.Items {
		result.Items = append(result.Items, &model.IndexerDefinition{ID: definition.ID, Name: definition.Name, Description: definition.Description, Language: definition.Language, Privacy: definition.Privacy, Digest: definition.Digest, Links: definition.Links, Modes: definition.Modes, Warnings: definition.Warnings, Settings: indexerSettingModels(definition.Settings)})
	}
	return result
}
func builtinPatchInput(input model.BuiltinIndexerPatchInput) (indexeradmin.BuiltinPatch, error) {
	patch := indexeradmin.BuiltinPatch{Name: input.Name, Enabled: input.Enabled, BaseURL: input.BaseURL, NetworkProfile: input.NetworkProfile}
	if input.TrustedOrigins != nil {
		patch.TrustedOrigins = &input.TrustedOrigins
	}
	if len(input.Settings) > 128 {
		return patch, indexer.ErrInvalidSettings
	}
	if input.Settings != nil {
		patch.Settings = map[string]config.IndexerSettingChange{}
	}
	for _, setting := range input.Settings {
		if setting == nil || setting.Name == "" || len(setting.Name) > 128 {
			return patch, indexer.ErrInvalidSettings
		}
		if _, duplicate := patch.Settings[setting.Name]; duplicate {
			return patch, indexer.ErrInvalidSettings
		}
		change := config.IndexerSettingChange{Clear: setting.Clear != nil && *setting.Clear}
		if setting.Value != nil {
			change.Value = &config.IndexerSettingValue{String: setting.Value.String, Boolean: setting.Value.Boolean, Number: setting.Value.Number}
			if err := change.Value.Validate(); err != nil {
				return patch, indexer.ErrInvalidSettings
			}
		}
		if change.Clear && change.Value != nil {
			return patch, indexer.ErrInvalidSettings
		}
		patch.Settings[setting.Name] = change
	}
	return patch, nil
}
func (r *Resolver) saveBuiltinIndexer(ctx context.Context, id, expected string, patch indexeradmin.BuiltinPatch) (*model.BuiltinIndexerResult, error) {
	if r.IndexerAdmin == nil {
		return nil, indexer.ErrNotConfigured
	}
	saved, err := r.IndexerAdmin.SaveBuiltin(ctx, id, expected, patch)
	if err != nil {
		return nil, err
	}
	return &model.BuiltinIndexerResult{Revision: saved.Revision, Instance: indexerInstanceModel(saved.Instance)}, nil
}
