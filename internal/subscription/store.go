package subscription

import (
	"context"
	"time"
)

type Store interface {
	GetCheckStates(context.Context, []string) ([]*PerformerState, error)
	GetCheckState(context.Context, string) (*PerformerState, error)
	SaveActivity(context.Context, string, ActivityState) error
	SaveCheckOutcome(context.Context, *PerformerState) error
	InitializeCheckSchedule(context.Context, string, time.Time) error
	ListCheckSchedules(context.Context) ([]CheckSchedule, error)
	ListDueCheckSchedules(context.Context, time.Time, *CheckSchedule, int) ([]CheckSchedule, error)

	Get(ctx context.Context, performerID string) (*PerformerState, error)
	Put(ctx context.Context, state *PerformerState) error
	Delete(ctx context.Context, performerID string) error
	List(ctx context.Context) ([]*PerformerState, error)
}

// CheckSchedule is a hot projection: it never loads scoring evidence or releases.
type CheckSchedule struct {
	PerformerID   string
	LastCheckedAt *time.Time
	NextCheckAt   *time.Time
}
