package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	bskyItem "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/mikeblum/atproto-graph-viz/graph/clickhouse/internal/db"
	"golang.org/x/sync/errgroup"
)

const (
	APP_INGEST = "atproto-graph-viz:ingest"
)

func (e *Engine) Ingest(ctx context.Context, workerID int, item bsky.RepoItem) error {
	var err error

	var records chan *db.AtgraphProfile
	if records, err = e.ingestItem(ctx, &item); records == nil || err != nil {
		err = errors.Join(err, e.ingestionErr(&item))
		return err
	}

	for {
		select {
		case record, ok := <-records:
			if !ok {
				e.log.With(
					"action", "ingest",
					"engine", "clickhouse",
					"type", item.NSID,
					"did", item.DID,
					"worker-id", workerID,
				).Debug("records channel closed, stopping ingest...")
				return err
			}
			e.log.With(
				"action", "ingest",
				"engine", "clickhouse",
				"record", record,
				"type", item.NSID,
				"did", item.DID.String(),
				"worker-id", workerID,
			).Info("Ingested bsky item")

		case <-ctx.Done():
			return errors.Join(err, ctx.Err())
		}
	}
}

func (e *Engine) ingestItem(ctx context.Context, item *bsky.RepoItem) (chan *db.AtgraphProfile, error) {
	e.log.With("nsid", item.NSID.String(), "did", item.DID.String(), "action", "ingest", "engine", "clickhouse").Info("Ingesting bsky item")
	records := make(chan *db.AtgraphProfile, 1)
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

func (e *Engine) ingestProfile(ctx context.Context, item *bsky.RepoItem, actor *bskyItem.ActorProfile) (chan *db.AtgraphProfile, error) {
	records := make(chan *db.AtgraphProfile, 1)
	group, ctx := errgroup.WithContext(ctx)
	session := db.New(e.db)
	createdTs, err := datetimeMust(item.DID, actor.CreatedAt)
	if err != nil {
		return nil, err
	}
	group.Go(func() error {
		defer close(records)
		session.InsertProfile(ctx, db.InsertProfileParams{
			Did:     item.DID.String(),
			Type:    actor.LexiconTypeID,
			Handle:  sql.NullString{String: item.Ident.Handle.String(), Valid: true},
			Created: createdTs,
			Rev:     sql.NullString{String: item.Rev, Valid: true},
			Sig:     sql.NullString{String: item.Sig, Valid: true},
			Version: item.Version,
		})

		if err != nil {
			e.log.WithErrorMsg(err, "Error ingesting app.bsky.actor.profile", "id", item.DID.String(), "action", "ingest", "type", "app.bsky.actor.profile")
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
