package subscription

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/performer"
)

type HomeSummary struct {
	Total, Attention, Unchecked int
	LastCheckedAt               *time.Time
	AttentionPerformerIDs       []string
}

// HomeSummary joins current Stash subscription flags with local check outcomes.
// It never fetches actor metadata, scenes, releases, or StashBox catalogues.
func (s *Service) HomeSummary(ctx context.Context) (HomeSummary, error) {
	flags, err := s.stash.PerformerSubscriptionFlags(ctx)
	if err != nil {
		return HomeSummary{}, err
	}
	states := map[string]*PerformerState{}
	var ids []string
	for _, flag := range flags {
		if flag != nil && performer.IsSubscribed(flag.CustomFields, s.customFieldKey) {
			if _, exists := states[flag.ID]; !exists {
				ids = append(ids, flag.ID)
				states[flag.ID] = &PerformerState{PerformerID: flag.ID}
			}
		}
	}
	for offset := 0; offset < len(ids); offset += 500 {
		page, err := s.store.GetCheckStates(ctx, ids[offset:min(offset+500, len(ids))])
		if err != nil {
			return HomeSummary{}, err
		}
		for _, state := range page {
			states[state.PerformerID] = state
		}
	}
	s.applyLatestCheckOutcomes(states)
	result := HomeSummary{Total: len(ids)}
	for _, state := range states {
		if strings.TrimSpace(state.LastError) != "" {
			result.Attention++
			result.AttentionPerformerIDs = append(result.AttentionPerformerIDs, state.PerformerID)
		}
		if state.LastCheckedAt == nil {
			result.Unchecked++
		} else if result.LastCheckedAt == nil || state.LastCheckedAt.After(*result.LastCheckedAt) {
			result.LastCheckedAt = copyTime(state.LastCheckedAt)
		}
	}
	sort.Strings(result.AttentionPerformerIDs)
	result.AttentionPerformerIDs = result.AttentionPerformerIDs[:min(3, len(result.AttentionPerformerIDs))]
	return result, nil
}
