package graph

import (
	"context"
	"errors"
	"fmt"
	"time"

	bskyItem "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	APP_INGEST = "atproto-graph-viz:ingest"
)

func (e *Engine) Ingest(ctx context.Context, stream chan *bsky.Item) error {
	var err error
	for item := range stream {
		var records chan *neo4j.Record
		if records = e.ingestItem(ctx, item); records == nil {
			err = errors.Join(e.ingestionErr(item))
			continue
		}
		for record := range records {
			e.log.With("record", record.AsMap(), "nsid", item.NSID.String(), "did", item.DID.String(), "action", "ingest", "engine", "neo4j").Info("ingested bsky item")
		}
	}

	return err
}

func (e *Engine) ingestItem(ctx context.Context, item *bsky.Item) chan *neo4j.Record {
	e.log.With("nsid", item.NSID.String(), "did", item.DID.String(), "action", "ingest", "engine", "neo4j").Info("ingesting bsky item")
	records := make(chan *neo4j.Record)
	defer close(records)
	var ok bool
	switch item.NSID {
	case bsky.ITEM_FEED_POST:
		if _, ok := item.Data.(*bskyItem.FeedPost); !ok {
			e.ingestionErr(item)
			return nil
		}
	case bsky.ITEM_ACTOR_PROFILE:
		var data *bskyItem.ActorProfile
		if data, ok = item.Data.(*bskyItem.ActorProfile); !ok {
			e.ingestionErr(item)
			return nil
		}
		return e.ingestProfile(ctx, item, data)
	case bsky.ITEM_GRAPH_FOLLOW:
		if _, ok = item.Data.(*bskyItem.GraphFollow); !ok {
			e.ingestionErr(item)
			return nil
		}
	case bsky.ITEM_FEED_REPOST:
		if _, ok = item.Data.(*bskyItem.FeedRepost); !ok {
			e.ingestionErr(item)
			return nil
		}
	case bsky.ITEM_FEED_LIKE:
		if _, ok = item.Data.(*bskyItem.FeedLike); !ok {
			e.ingestionErr(item)
			return nil
		}
	case bsky.ITEM_GRAPH_BLOCK:
		if _, ok = item.Data.(*bskyItem.GraphBlock); !ok {
			e.ingestionErr(item)
			return nil
		}
	case bsky.ITEM_GRAPH_LIST_BLOCK:
		if _, ok = item.Data.(*bskyItem.GraphListblock); !ok {
			e.ingestionErr(item)
			return nil
		}
	case bsky.ITEM_GRAPH_LIST:
		if _, ok := item.Data.(*bskyItem.GraphList); !ok {
			e.ingestionErr(item)
			return nil
		}
	case bsky.ITEM_GRAPH_LIST_ITEM:
		if _, ok = item.Data.(*bskyItem.GraphListitem); !ok {
			e.ingestionErr(item)
			return nil
		}
	default:
		err := fmt.Errorf("found unknown type in tree")
		e.log.With("err", err, "nsid", item.NSID).Warn("unrecognized lexicon type")
	}
	return records
}

func (e *Engine) ingestionErr(item *bsky.Item) error {
	err := fmt.Errorf("ingest: error mapping %s", item.NSID)
	e.log.WithError(err, "Error ingesting bsky item", "action", "ingest", "engine", "neo4j")
	return err
}

func (e *Engine) ingestProfile(ctx context.Context, item *bsky.Item, actor *bskyItem.ActorProfile) chan *neo4j.Record {
	records := make(chan *neo4j.Record, 1)
	session := e.driver.NewSession(context.Background(), e.config)
	defer session.Close(ctx)
	go session.ExecuteWrite(ctx,
		func(tx neo4j.ManagedTransaction) (any, error) {
			defer close(records)
			createdTiemstamp, err := datetimeMust(item.DID, actor.CreatedAt)
			if err != nil {
				return nil, err
			}
			result, err := tx.Run(ctx, `
				MERGE (p:Profile {id: $id})
				ON CREATE
					SET
						p.type = $type,
						// tracking ingestion lag time
  						p.ingested = timestamp(),
						p.created = $created
				ON MATCH
					SET 
						p.handle = $handle,
						// tracking firehose lag time
  						p.updated = timestamp()
				RETURN p.id, p.ingested;
                `, map[string]any{
				"id":     item.DID.String(),
				"type":   actor.LexiconTypeID,
				"handle": item.Ident.Handle.String(),
				// neo4j (java) expects epoch time in milliseconds
				"created": createdTiemstamp.Unix() * int64(time.Second/time.Millisecond),
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
		// neo4j.WithTxTimeout(timeout()),
		neo4j.WithTxMetadata(map[string]any{"app": APP_INGEST}))
	return records
}

func datetimeMust(did syntax.DID, datetime *string) (*time.Time, error) {
	var parsedTime time.Time
	var err error
	if datetime == nil {
		return nil, fmt.Errorf("missing datetime: did=%s", did.String())
	}
	if parsedTime, err = time.Parse(time.RFC3339, *datetime); err != nil {
		return nil, err
	}
	return &parsedTime, nil
}
