package resourceselection

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Historical policy snapshots stored positive and negative references in both
// global lists. Decode them into equivalent direct lists without changing a
// saved decision. Configuration uses strict YAML and is migrated only manually.
func (g *Group) UnmarshalJSON(data []byte) error {
	var raw struct {
		Fallback bool            `json:"fallback"`
		ID       string          `json:"id"`
		Name     string          `json:"name"`
		Require  json.RawMessage `json:"require"`
		Exclude  json.RawMessage `json:"exclude"`
		Tiers    []Tier          `json:"tiers"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	required, notRequired, err := snapshotGlobalRefs(raw.Require)
	if err != nil {
		return err
	}
	excluded, notExcluded, err := snapshotGlobalRefs(raw.Exclude)
	if err != nil {
		return err
	}
	*g = Group{Fallback: raw.Fallback, ID: raw.ID, Name: raw.Name, Require: uniqueIDs(required, notExcluded), Exclude: uniqueIDs(excluded, notRequired), Tiers: raw.Tiers}
	return nil
}
func snapshotGlobalRefs(data json.RawMessage) ([]string, []string, error) {
	if len(data) == 0 {
		return nil, nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, nil, err
	}
	direct, inverse := []string{}, []string{}
	for _, item := range items {
		var id string
		if err := json.Unmarshal(item, &id); err == nil && id != "" {
			direct = append(direct, id)
			continue
		}
		var old struct {
			RuleID string `json:"ruleID"`
			Negate bool   `json:"negate"`
		}
		if err := json.Unmarshal(item, &old); err != nil {
			return nil, nil, err
		}
		if old.RuleID == "" {
			return nil, nil, fmt.Errorf("invalid global rule snapshot reference")
		}
		if old.Negate {
			inverse = append(inverse, old.RuleID)
		} else {
			direct = append(direct, old.RuleID)
		}
	}
	return direct, inverse, nil
}
func uniqueIDs(first, second []string) []string {
	out := []string{}
	for _, items := range [][]string{first, second} {
		for _, id := range items {
			if !slices.Contains(out, id) {
				out = append(out, id)
			}
		}
	}
	return out
}

// Only historical report policies accept the old object references. Their saved
// evaluations remain authoritative; v7 never evaluates a legacy policy.
func historicalTierIDs(data []byte) ([]byte, error) {
	var policy map[string]json.RawMessage
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}
	var version int
	if len(policy["version"]) == 0 {
		return data, nil
	}
	if err := json.Unmarshal(policy["version"], &version); err != nil {
		return nil, err
	}
	if version >= Version || len(policy["groups"]) == 0 {
		return data, nil
	}
	var groups []map[string]json.RawMessage
	if err := json.Unmarshal(policy["groups"], &groups); err != nil {
		return nil, err
	}
	for _, g := range groups {
		var tiers []map[string]json.RawMessage
		if len(g["tiers"]) == 0 {
			continue
		}
		if err := json.Unmarshal(g["tiers"], &tiers); err != nil {
			return nil, err
		}
		for _, tier := range tiers {
			if len(tier["rules"]) == 0 {
				continue
			}
			var refs []json.RawMessage
			if err := json.Unmarshal(tier["rules"], &refs); err != nil {
				return nil, err
			}
			ids := []string{}
			for _, raw := range refs {
				var id string
				if err := json.Unmarshal(raw, &id); err != nil {
					var old struct {
						RuleID string `json:"ruleID"`
					}
					if err := json.Unmarshal(raw, &old); err != nil {
						return nil, err
					}
					id = old.RuleID
				}
				if id == "" {
					return nil, fmt.Errorf("invalid historical tier reference")
				}
				ids = append(ids, id)
			}
			tier["rules"], _ = json.Marshal(uniqueIDs(ids, nil))
		}
		g["tiers"], _ = json.Marshal(tiers)
	}
	policy["groups"], _ = json.Marshal(groups)
	return json.Marshal(policy)
}
