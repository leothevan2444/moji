package graphqlapi

import "github.com/leothevan2444/moji/internal/graphqlapi/model"

func torrentSelectionSettingsFromModel(input *model.TorrentSelectionSettingsInput) TorrentSelectionSettingsSnapshot {
	if input == nil {
		return TorrentSelectionSettingsSnapshot{}
	}
	fastRules := torrentSelectionRulesFromModel(input.FastRules)
	torrentRules := torrentSelectionRulesFromModel(input.TorrentRules)
	return TorrentSelectionSettingsSnapshot{
		Enabled:                  input.Enabled,
		InspectionCandidateLimit: input.InspectionCandidateLimit,
		FastRules:                fastRules,
		TorrentRules:             torrentRules,
	}
}

func torrentSelectionRulesFromModel(rules []*model.TorrentSelectionRuleInput) []TorrentSelectionRuleSnapshot {
	out := make([]TorrentSelectionRuleSnapshot, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		item := TorrentSelectionRuleSnapshot{
			Type:    string(rule.Type),
			Enabled: rule.Enabled,
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
		if rule.PublishDate != nil {
			item.PublishDate = DirectionRuleSnapshot{
				Direction: string(rule.PublishDate.Direction),
			}
		}
		if rule.Seeders != nil {
			item.Seeders = DirectionRuleSnapshot{
				Direction: string(rule.Seeders.Direction),
			}
		}
		if rule.Size != nil {
			item.Size = DirectionRuleSnapshot{
				Direction: string(rule.Size.Direction),
			}
		}
		if rule.TorrentFileNameMatch != nil {
			item.TorrentFileNameMatch.Clauses = make([]TorrentFileNameMatchClauseSnapshot, 0, len(rule.TorrentFileNameMatch.Clauses))
			for _, clause := range rule.TorrentFileNameMatch.Clauses {
				if clause == nil {
					continue
				}
				item.TorrentFileNameMatch.Clauses = append(item.TorrentFileNameMatch.Clauses, TorrentFileNameMatchClauseSnapshot{
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
