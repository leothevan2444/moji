package subscription

import "context"

type Store interface {
	Get(ctx context.Context, performerID string) (*PerformerState, error)
	Put(ctx context.Context, state *PerformerState) error
	Delete(ctx context.Context, performerID string) error
	List(ctx context.Context) ([]*PerformerState, error)
}
