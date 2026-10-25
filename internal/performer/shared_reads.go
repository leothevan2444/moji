package performer

import (
	"context"
	"time"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

// Only concurrent reads are shared. Later page visits still read fresh Stash data.
// A departing browser cannot cancel a read still needed by another request.
func (s *Service) sharedRead(ctx context.Context, key string, read func(context.Context) (any, error)) (any, error) {
	result := s.reads.DoChan(key, func() (any, error) {
		bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		return read(bounded)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case value := <-result:
		return value.Val, value.Err
	}
}

func (s *Service) findPerformer(ctx context.Context, id string) (*stashgraphql.PerformerFragment, error) {
	value, err := s.sharedRead(ctx, "performer:"+id, func(ctx context.Context) (any, error) { return s.stash.FindPerformerByID(ctx, id) })
	if err != nil {
		return nil, err
	}
	return value.(*stashgraphql.PerformerFragment), nil
}

func (s *Service) sharedStashScenes(ctx context.Context, id string) ([]*stashgraphql.SceneFragment, error) {
	value, err := s.sharedRead(ctx, "scenes:"+id, func(ctx context.Context) (any, error) { return s.fetchStashScenesUnshared(ctx, id) })
	if err != nil {
		return nil, err
	}
	return value.([]*stashgraphql.SceneFragment), nil
}
