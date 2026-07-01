package graphqlapi

import "github.com/leothevan2444/moji/internal/graphqlapi/model"

func torrentSelectionSettingsFromModel(input *model.TorrentSelectionSettingsInput) TorrentSelectionSettingsSnapshot {
	if input == nil {
		return TorrentSelectionSettingsSnapshot{}
	}
	return TorrentSelectionSettingsSnapshot{
		Enabled: input.Enabled,
		Rules:   torrentSelectionRulesFromModel(input.Rules),
	}
}

func torrentSelectionRulesFromModel(rules []*model.TorrentSelectionRuleInput) []TorrentSelectionRuleSnapshot {
	out := make([]TorrentSelectionRuleSnapshot, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		item := TorrentSelectionRuleSnapshot{
			ID:        rule.ID,
			Type:      string(rule.Type),
			Enabled:   rule.Enabled,
			Direction: string(rule.Direction),
		}
		if rule.IndexerPreference != nil {
			item.IndexerPreference = IndexerPreferenceRuleSnapshot{
				TrackerIDs: append([]string(nil), rule.IndexerPreference.TrackerIds...),
			}
		}
		if rule.TitleMatch != nil {
			item.TitleMatch.Clauses = make([]TitleMatchClauseSnapshot, 0, len(rule.TitleMatch.Clauses))
			for _, clause := range rule.TitleMatch.Clauses {
				if clause == nil {
					continue
				}
				item.TitleMatch.Clauses = append(item.TitleMatch.Clauses, TitleMatchClauseSnapshot{
					Pattern:     clause.Pattern,
					PatternMode: string(clause.PatternMode),
					Effect:      string(clause.Effect),
				})
			}
		}
		out = append(out, item)
	}
	return out
}
