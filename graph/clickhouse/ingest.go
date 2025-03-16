package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	bskyItem "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/mikeblum/atproto-graph-viz/graph/clickhouse/internal/db"
	"golang.org/x/sync/errgroup"
)

const (
	APP_INGEST = "atproto-graph-viz:ingest"
)

func (e *IngestEngine) Ingest(ctx context.Context, workerID int, item bsky.RepoItem) error {
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

func (e *IngestEngine) ingestItem(ctx context.Context, item *bsky.RepoItem) (chan *db.AtgraphProfile, error) {
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

func (e *IngestEngine) ingestionErr(item *bsky.RepoItem) error {
	err := fmt.Errorf("ingest: error mapping %s", item.NSID)
	e.log.WithErrorMsg(err, "Error ingesting bsky item", "action", "ingest", "engine", "clickhouse")
	return err
}

func (e *IngestEngine) ingestProfile(ctx context.Context, item *bsky.RepoItem, actor *bskyItem.ActorProfile) (chan *db.AtgraphProfile, error) {
	var conn *ch.Client
	var err error
	records := make(chan *db.AtgraphProfile, 1)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {

		defer close(records)

		// open ch-db conn
		if conn, err = newConn(ctx); err != nil {
			return err
		}
		defer conn.Close()

		// atgraph.profiles headers
		var (
			did       proto.ColStr
			lexicon   = proto.NewLowCardinality(new(proto.ColStr))
			handle    proto.ColStr
			createdTs = new(proto.ColDateTime64).WithPrecision(proto.PrecisionNano)
			rev       proto.ColStr
			sig       proto.ColStr
			version   proto.ColUInt8
		)

		// load data
		did.Append(item.DID.String())
		lexicon.Append(actor.LexiconTypeID)
		handle.Append(item.Ident.Handle.String())

		if ts, err := datetimeMust(item.DID, actor.CreatedAt); ts != nil {
			createdTs.Append(*ts)
		} else {
			e.log.WithError(err).Error("Missing created timestamp", "did", item.DID, "lexicon", actor.LexiconTypeID)
		}

		rev.Append(item.Rev)
		sig.Append(item.Sig)
		version.Append(uint8(item.Version))

		input := proto.Input{
			{Name: "did", Data: did},
			{Name: "lexicon", Data: lexicon},
			{Name: "handle", Data: handle},
			{Name: "created", Data: createdTs},
			{Name: "rev", Data: rev},
			{Name: "sig", Data: sig},
			{Name: "version", Data: version},
		}

		if err = conn.Do(ctx, ch.Query{
			Body:  input.Into("profiles"), // helper that generates INSERT INTO query with all columns
			Input: input,
		}); err != nil {
			e.log.WithErrorMsg(err, fmt.Sprintf("Error ingesting %s", actor.LexiconTypeID), "id", item.DID.String(), "action", "ingest", "lexicon", actor.LexiconTypeID)
			return err
		}
		records <- &db.AtgraphProfile{
			Did: item.DID.String(),
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
