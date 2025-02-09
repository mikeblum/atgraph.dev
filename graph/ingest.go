package graph

import (
	"context"
	"errors"
	"fmt"
	"sync"

	bskyItem "github.com/bluesky-social/indigo/api/bsky"
	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"golang.org/x/sync/errgroup"
)

const (
	APP_INGEST = "atproto-graph-viz:ingest"
)

func (e *Engine) Ingest(ctx context.Context, stream chan *bsky.Item) error {
	var errs []error
	var mu sync.Mutex

	group, _ := errgroup.WithContext(ctx)
	for item := range stream {
		group.Go(func() error {
			var err error
			if _, err = e.ingestItem(ctx, item); err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, err)
			}
			// allow other goroutines to progress
			return nil
		})
	}

	_ = group.Wait()

	return errors.Join(errs...)
}

func (e *Engine) ingestItem(ctx context.Context, item *bsky.Item) (chan *neo4j.Record, error) {
	e.log.With("nsid", item.NSID.String(), "did", item.DID.String(), "action", "ingest", "engine", "neo4j").Info("ingesting bsky item")
	records := make(chan *neo4j.Record)
	defer close(records)
	var ok bool
	switch item.NSID {
	case bsky.ITEM_FEED_POST:
		if _, ok := item.Data.(*bskyItem.FeedPost); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
	case bsky.ITEM_ACTOR_PROFILE:
		var data *bskyItem.ActorProfile
		if data, ok = item.Data.(*bskyItem.ActorProfile); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
		return e.ingestProfile(ctx, item, data)
	case bsky.ITEM_GRAPH_FOLLOW:
		if _, ok = item.Data.(*bskyItem.GraphFollow); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
	case bsky.ITEM_FEED_REPOST:
		if _, ok = item.Data.(*bskyItem.FeedRepost); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
	case bsky.ITEM_FEED_LIKE:
		if _, ok = item.Data.(*bskyItem.FeedLike); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
	case bsky.ITEM_GRAPH_BLOCK:
		if _, ok = item.Data.(*bskyItem.GraphBlock); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
	case bsky.ITEM_GRAPH_LIST_BLOCK:
		if _, ok = item.Data.(*bskyItem.GraphListblock); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
	case bsky.ITEM_GRAPH_LIST:
		if _, ok := item.Data.(*bskyItem.GraphList); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
	case bsky.ITEM_GRAPH_LIST_ITEM:
		if _, ok = item.Data.(*bskyItem.GraphListitem); !ok {
			return nil, fmt.Errorf("ingest: error mapping %s", item.NSID)
		}
	default:
		err := fmt.Errorf("found unknown type in tree")
		e.log.With("err", err, "nsid", item.NSID).Warn("unrecognized lexicon type")
	}
	return records, nil
}

func (e *Engine) ingestProfile(ctx context.Context, item *bsky.Item, actor *bskyItem.ActorProfile) (chan *neo4j.Record, error) {
	records := make(chan *neo4j.Record)
	defer close(records)
	session := e.driver.NewSession(context.Background(), e.config)
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx,
		func(tx neo4j.ManagedTransaction) (any, error) {
			result, err := tx.Run(ctx, `
				MERGE (p:Profile {id: $id})
				ON CREATE
					SET p.type = $type
  					SET p.ingested = timestamp()
					SET p.created = $created
				ON MATCH
					SET p.handle = $handle
				RETURN p.id, p.ingested;
                `, map[string]any{
				"id":      item.DID.String(),
				"type":    actor.LexiconTypeID,
				"handle":  item.Ident.Handle.String(),
				"created": actor.CreatedAt,
			})
			if err != nil {
				return nil, err
			}
			for result.Next(ctx) {
				record := result.Record()
				records <- record
			}
			return records, nil
		},
		neo4j.WithTxTimeout(timeout()),
		neo4j.WithTxMetadata(map[string]any{"app": APP_INGEST}))
	return records, err
}
