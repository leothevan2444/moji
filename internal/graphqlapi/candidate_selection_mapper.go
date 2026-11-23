package graphqlapi

import (
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/resourceselection"
)

func torrentSelectionSettingsFromModel(input *model.TorrentSelectionSettingsInput) TorrentSelectionSettingsSnapshot {
	if input == nil {
		return TorrentSelectionSettingsSnapshot{}
	}
	p := resourceselection.Policy{Version: input.Version, ActiveGroup: input.ActiveGroup, Rules: []resourceselection.Rule{}, Groups: []resourceselection.Group{}}
	for _, r := range input.Rules {
		if r == nil {
			continue
		}
		v := resourceselection.Rule{ID: r.ID, Name: r.Name, Conditions: []resourceselection.Condition{}}
		for _, c := range r.Conditions {
			if c != nil {
				v.Conditions = append(v.Conditions, resourceselection.Condition{Scope: c.Scope, Value: c.Value, Minimum: c.Minimum, Maximum: c.Maximum})
			}
		}
		p.Rules = append(p.Rules, v)
	}
	for _, g := range input.Groups {
		if g == nil {
			continue
		}
		v := resourceselection.Group{ID: g.ID, Name: g.Name, Fallback: g.Fallback, Require: g.Require, Exclude: g.Exclude, Tiers: []resourceselection.Tier{}}
		for _, t := range g.Tiers {
			if t != nil {
				v.Tiers = append(v.Tiers, resourceselection.Tier{ID: t.ID, Name: t.Name, Rules: append([]string{}, t.Rules...)})
			}
		}
		p.Groups = append(p.Groups, v)
	}
	return p
}
