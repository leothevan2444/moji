package subscription

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/apptime"
	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/releasedate"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

// ActivityState is a confirmed catalog observation and its durable schedule.
// Task creation outcomes never determine catalog activity or retry cadence.
type ActivityState struct {
	EvaluationReason ActivityEvaluationStatus `json:"evaluation_reason,omitempty"`
	Value            *float64                 `json:"value,omitempty"`
	Rank             *int                     `json:"rank,omitempty"`
	LowerBound       bool                     `json:"lower_bound,omitempty"`
	ObservedAt       *time.Time               `json:"observed_at,omitempty"`
	LastSuccessAt    *time.Time               `json:"last_success_at,omitempty"`
	NextCheckAt      *time.Time               `json:"next_check_at,omitempty"`
	Failures         int                      `json:"failures,omitempty"`
	Days             []ActivityDay            `json:"days,omitempty"`
}

// Evaluation status describes missing evidence, independently of an active job.
type ActivityEvaluationStatus string

const (
	ActivityReady             ActivityEvaluationStatus = "READY"
	ActivityNotChecked        ActivityEvaluationStatus = "NOT_CHECKED"
	ActivityCheckFailed       ActivityEvaluationStatus = "CHECK_FAILED"
	ActivityIncompleteCatalog ActivityEvaluationStatus = "INCOMPLETE_CATALOG"
	ActivityInvalidDates      ActivityEvaluationStatus = "INVALID_DATES"
	ActivityInvalidMembership ActivityEvaluationStatus = "INVALID_MEMBERSHIP"
)

func (a ActivityState) EvaluationStatus() ActivityEvaluationStatus {
	if a.Value != nil && a.Rank != nil {
		return ActivityReady
	}
	if a.ObservedAt == nil {
		if a.Failures > 0 {
			return ActivityCheckFailed
		}
		return ActivityNotChecked
	}
	switch a.EvaluationReason {
	case ActivityInvalidDates, ActivityInvalidMembership:
		return a.EvaluationReason
	default:
		return ActivityIncompleteCatalog
	}
}

type ActivityDay struct {
	Date   string  `json:"date"`
	Weight float64 `json:"weight"`
}

func activityInterval(rank *int) time.Duration {
	if rank == nil {
		return 48 * time.Hour
	}
	days := [...]int{30, 14, 7, 3, 1}
	if *rank < 1 || *rank > 5 {
		return 48 * time.Hour
	}
	return time.Duration(days[*rank-1]) * 24 * time.Hour
}

func activityRank(value float64) int {
	for i, threshold := range []float64{.25, 1, 2.5, 6} {
		if value < threshold {
			return i + 1
		}
	}
	return 5
}

// Calendar days are counted in the configured release timezone. Date-only
// metadata retains its source calendar date, including across DST boundaries.
func activityToday(now time.Time, zone string) time.Time {
	day, _ := time.Parse("2006-01-02", now.In(apptime.Location(zone)).Format("2006-01-02"))
	return day
}

func activityWeight(scene *stashboxgraphql.SceneFragment) float64 {
	title := strings.ToLower(strings.TrimSpace(stringValue(scene.Title)))
	fields := []string{title}
	for _, tag := range scene.Tags {
		if tag != nil {
			fields = append(fields, strings.ToLower(strings.TrimSpace(tag.Name)))
		}
	}
	for _, field := range fields {
		for _, marker := range []string{"総集編", "総集", "ベスト", "compilation", "anthology", "omnibus", "再発売", "re-release"} {
			if strings.Contains(field, marker) {
				return .2
			}
		}
		for _, word := range strings.FieldsFunc(field, func(r rune) bool { return r < 'a' || r > 'z' }) {
			if word == "best" {
				return .2
			}
		}
	}
	if title == "" {
		return .5
	}
	return 1
}

type activitySample struct {
	records           map[string]ActivityDay
	seen              map[string]bool
	invalidMembership bool
	invalidDate       bool
	covered           bool
	today             time.Time
}

func newActivitySample(now time.Time, zone string) *activitySample {
	return &activitySample{records: map[string]ActivityDay{}, seen: map[string]bool{}, today: activityToday(now, zone)}
}

// Observe raw catalog scenes before policy, library, code or task filtering.
func (a *activitySample) observe(scenes []*stashboxgraphql.SceneFragment, complete bool) {
	if complete {
		a.covered = true
	}
	for _, scene := range scenes {
		if scene == nil || a.seen[scene.ID] {
			continue
		}
		a.seen[scene.ID] = true
		date, ok := releasedate.Parse(stringValue(scene.Date))
		if !ok {
			a.invalidDate = true
			continue
		}
		day, _ := time.Parse("2006-01-02", date)
		age := int(a.today.Sub(day).Hours() / 24)
		if age < 0 {
			continue
		}
		if age > 365 {
			a.covered = true
			continue
		}
		key := "scene:" + scene.ID
		if code := film.Code(stringValue(scene.Code)); code != "" {
			key = "code:" + code
		}
		dayRecord := ActivityDay{Date: date, Weight: activityWeight(scene)}
		existing, exists := a.records[key]
		// A code repeated across dates represents one work. Prefer the latest
		// reported release date, and the strongest type evidence on that date.
		if !exists || dayRecord.Date > existing.Date || (dayRecord.Date == existing.Date && dayRecord.Weight > existing.Weight) {
			a.records[key] = dayRecord
		}
	}
}

func (a *activitySample) snapshot(now time.Time) ActivityState {
	out := ActivityState{ObservedAt: &now, LastSuccessAt: &now, EvaluationReason: ActivityIncompleteCatalog}
	if a.invalidDate {
		out.EvaluationReason = ActivityInvalidDates
	}
	if a.invalidMembership {
		out.EvaluationReason = ActivityInvalidMembership
	}
	value := 0.0
	days := map[string]float64{}
	for _, record := range a.records {
		if record.Weight > days[record.Date] {
			days[record.Date] = record.Weight
		}
	}
	dates := make([]string, 0, len(days))
	for date := range days {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	for _, date := range dates {
		weight := days[date]
		day, _ := time.Parse("2006-01-02", date)
		value += weight * math.Exp2(-a.today.Sub(day).Hours()/24/60)
		out.Days = append(out.Days, ActivityDay{Date: date, Weight: weight})
	}
	sort.Slice(out.Days, func(i, j int) bool { return out.Days[i].Date > out.Days[j].Date })
	// An incomplete catalog can prove the highest tier, never a lower tier.
	if (a.covered && !a.invalidDate && !a.invalidMembership) || value >= 6 {
		rank := activityRank(value)
		out.Value, out.Rank = &value, &rank
		out.LowerBound = !a.covered || a.invalidDate || a.invalidMembership
		out.EvaluationReason = ""
	}
	return out
}

func (a *activitySample) sufficient(now time.Time) bool { return a.snapshot(now).Rank != nil }

// PerformerActivity reads persisted observations without querying StashBox.
func (s *Service) PerformerActivity(ctx context.Context, id string) (ActivityState, error) {
	state, err := s.store.GetCheckState(ctx, id)
	if err != nil || state == nil {
		return ActivityState{}, err
	}
	s.checks.mu.Lock()
	enabled := s.checks.enabled
	s.checks.mu.Unlock()
	activity := state.Activity
	if !enabled {
		activity.NextCheckAt = nil
	}
	return activity, nil
}

// PerformerActivities reads at most one visible page of saved observations.
// It includes manually checked actors regardless of subscription status.
func (s *Service) PerformerActivities(ctx context.Context, ids []string) (map[string]ActivityState, error) {
	out := map[string]ActivityState{}
	if len(ids) == 0 {
		return out, nil
	}
	ids, err := normalizePerformerBatchIDs(ids)
	if err != nil {
		return nil, err
	}
	states, err := s.store.GetCheckStates(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = ActivityState{}
	}
	for _, state := range states {
		out[state.PerformerID] = state.Activity
	}
	return out, nil
}
