package graphqlapi

import (
	"errors"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
)

func settingsSectionInput(input model.UpdateSettingsSectionInput) (SettingsSectionUpdate, error) {
	update := SettingsSectionUpdate{Section: string(input.Section)}
	count := 0
	if input.TaskDeletePolicy != nil {
		count++
	}
	if input.Tasks != nil {
		count++
	}
	if input.Resources != nil {
		count++
	}
	if input.MetadataPriority != nil {
		count++
	}
	if input.QueryOrder != nil {
		count++
	}
	if count != 1 {
		return update, errors.New("exactly one settings section payload is required")
	}
	switch input.Section {
	case model.SettingsSectionTaskDeletion:
		if input.TaskDeletePolicy == nil {
			return update, errors.New("task deletion policy is required")
		}
		policy := string(*input.TaskDeletePolicy)
		update.TaskDeletePolicy = &policy
	case model.SettingsSectionTasks:
		task := input.Tasks
		if task == nil {
			return update, errors.New("tasks payload is required")
		}
		if task.SourcingRetryIntervalHours < 1 || task.SourcingRetryIntervalHours > 168 {
			return update, errors.New("resource search interval must be between 1 and 168 hours")
		}
		update.TaskProcessingEnabled = &task.TaskProcessingEnabled
		update.TaskProgressSyncIntervalSeconds = task.TaskProgressSyncIntervalSeconds
		update.SourcingRetryIntervalHours = task.SourcingRetryIntervalHours
		update.SubscriptionPollIntervalHours = task.SubscriptionPollIntervalHours
		policy := task.SubscriptionReleasePolicy
		update.SubscriptionReleasePolicy = SubscriptionReleasePolicySnapshot{
			SoloBehavior: string(policy.SoloBehavior), GroupBehavior: string(policy.GroupBehavior), CompilationBehavior: string(policy.CompilationBehavior),
			MaxGroupPerformerCount: policy.MaxGroupPerformerCount, ReleaseDateRange: string(policy.ReleaseDateRange),
		}
		if policy.VrBehavior != nil {
			update.SubscriptionReleasePolicy.VRBehavior = string(*policy.VrBehavior)
		}
	case model.SettingsSectionResources:
		if input.Resources == nil {
			return update, errors.New("resources payload is required")
		}
		update.TorrentSelection = torrentSelectionSettingsFromModel(input.Resources)
	case model.SettingsSectionMetadata:
		if input.MetadataPriority == nil {
			return update, errors.New("metadata priority is required")
		}
		update.StashBoxEndpoints = input.MetadataPriority
	case model.SettingsSectionIndexers:
		if input.QueryOrder == nil {
			return update, errors.New("query order is required")
		}
		update.TorrentSelection.FastRules = []TorrentSelectionRuleSnapshot{{Type: "INDEXER_PREFERENCE", Enabled: true, IndexerPreference: IndexerPreferenceRuleSnapshot{TrackerIDs: input.QueryOrder}}}
	default:
		return update, errors.New("unsupported settings section")
	}
	return update, nil
}
