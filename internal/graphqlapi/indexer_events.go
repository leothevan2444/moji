package graphqlapi

import (
	"context"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
)

type IndexerEventsService interface {
	Subscribe() (<-chan indexer.Event, func(), error)
}

func (r *Resolver) indexerEvents(ctx context.Context) (<-chan *model.IndexerEvent, error) {
	if r.IndexerEventSource == nil {
		return nil, indexer.ErrNotConfigured
	}
	events, unsubscribe, err := r.IndexerEventSource.Subscribe()
	if err != nil {
		return nil, err
	}
	out := make(chan *model.IndexerEvent, 1)
	go func() {
		defer close(out)
		defer unsubscribe()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				item := &model.IndexerEvent{StreamID: event.StreamID, Sequence: event.Sequence, Kind: model.IndexerEventKind(event.Kind)}
				// Only this producer writes out. Coalesce instead of blocking
				// shutdown behind a client that stopped consuming notifications.
				select {
				case <-out:
				default:
				}
				out <- item
			}
		}
	}()
	return out, nil
}
