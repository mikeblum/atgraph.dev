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
	"golang.org/x/sync/errgroup"
)

const (
	APP_INGEST = "atproto-graph-viz:ingest"
)

func (e *Engine) Ingest(ctx context.Context, items chan bsky.RepoItem) error {
	var err error
	for {
		select {
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())
		case item, ok := <-items:
			if !ok {
				// Channel is closed, return any accumulated errors
				return err
			}

			var records chan *neo4j.Record
			if records, err = e.ingestItem(ctx, &item); records == nil || err != nil {
				err = errors.Join(err, e.ingestionErr(&item))
				continue
			}
			// Process records with context awareness
		recordLoop:
			for {
				select {
				case record, ok := <-records:
					if !ok {
						break recordLoop
					}
					e.log.With(
						"record", record.AsMap(),
						"nsid", item.NSID.String(),
						"did", item.DID.String(),
						"action", "ingest",
						"engine", "neo4j",
					).Info("Ingested bsky item")

				case <-ctx.Done():
					return errors.Join(err, ctx.Err())
				}
			}
		}
		e.log.With(
			"action", "ingest",
			"engine", "neo4j",
		).Debug("Awaiting items to ingest")
	}
}

func (e *Engine) ingestItem(ctx context.Context, item *bsky.RepoItem) (chan *neo4j.Record, error) {
	e.log.With("nsid", item.NSID.String(), "did", item.DID.String(), "action", "ingest", "engine", "neo4j").Info("Ingesting bsky item")
	records := make(chan *neo4j.Record, 1)
	defer close(records)
	var err error
	var ok bool
	switch item.NSID {
	case bsky.ITEM_FEED_POST:
		if _, ok := item.Data.(*bskyItem.FeedPost); !ok {
			e.ingestionErr(item)
			return records, nil
		}
	case bsky.ITEM_ACTOR_PROFILE:
		var data *bskyItem.ActorProfile
		if data, ok = item.Data.(*bskyItem.ActorProfile); !ok {
			e.ingestionErr(item)
			return records, nil
		}
		return e.ingestProfile(ctx, item, data)
	case bsky.ITEM_GRAPH_FOLLOW:
		if _, ok = item.Data.(*bskyItem.GraphFollow); !ok {
			e.ingestionErr(item)
			return records, nil
		}
	case bsky.ITEM_FEED_REPOST:
		if _, ok = item.Data.(*bskyItem.FeedRepost); !ok {
			e.ingestionErr(item)
			return records, nil
		}
	case bsky.ITEM_FEED_LIKE:
		if _, ok = item.Data.(*bskyItem.FeedLike); !ok {
			e.ingestionErr(item)
			return records, nil
		}
	case bsky.ITEM_GRAPH_BLOCK:
		if _, ok = item.Data.(*bskyItem.GraphBlock); !ok {
			e.ingestionErr(item)
			return records, nil
		}
	case bsky.ITEM_GRAPH_LIST_BLOCK:
		if _, ok = item.Data.(*bskyItem.GraphListblock); !ok {
			e.ingestionErr(item)
			return records, nil
		}
	case bsky.ITEM_GRAPH_LIST:
		if _, ok := item.Data.(*bskyItem.GraphList); !ok {
			e.ingestionErr(item)
			return records, nil
		}
	case bsky.ITEM_GRAPH_LIST_ITEM:
		if _, ok = item.Data.(*bskyItem.GraphListitem); !ok {
			e.ingestionErr(item)
			return records, nil
		}
	default:
		err = fmt.Errorf("found unknown type in tree")
		e.log.With("err", err, "nsid", item.NSID).Debug("unrecognized lexicon type")
	}
	return records, err
}

func (e *Engine) ingestionErr(item *bsky.RepoItem) error {
	err := fmt.Errorf("ingest: error mapping %s", item.NSID)
	e.log.WithErrorMsg(err, "Error ingesting bsky item", "action", "ingest", "engine", "neo4j")
	return err
}

func (e *Engine) ingestProfile(ctx context.Context, item *bsky.RepoItem, actor *bskyItem.ActorProfile) (chan *neo4j.Record, error) {
	records := make(chan *neo4j.Record, 1)
	session := e.driver.NewSession(ctx, e.config)
	defer session.Close(ctx)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		_, err := session.ExecuteWrite(ctx,
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
							p.type		= $type,
							// tracking ingestion lag time
							p.ingested 	= timestamp(),
							p.created 	= $created
					ON MATCH
						SET 
							p.handle	= $handle,
							p.rev		= $rev,
							p.sig 		= $sig,
							p.version 	= $version,
							// tracking firehose lag time
							p.updated 	= timestamp()
					RETURN p.id AS did, p.ingested AS ingested_ts;
					`, map[string]any{
					"id":     item.DID.String(),
					"rev":    item.Rev,
					"sig":    item.Sig,
					"type":   actor.LexiconTypeID,
					"handle": item.Ident.Handle.String(),
					// neo4j (java) expects epoch time in milliseconds
					"created": createdTiemstamp.Unix() * int64(time.Second/time.Millisecond),
					"version": item.Version,
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
		if err != nil {
			e.log.WithErrorMsg(err, "Error ingesting :Profile", "id", item.DID.String(), "action", "ingest")
			return err
		}
		return nil
	})

	return records, group.Wait()
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
