package bsky

import (
	"context"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/events"
	"github.com/mikeblum/atproto-graph-viz/conf"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"

	"github.com/gorilla/websocket"
)

const (
	BSKY_WSS_URL    = "wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos"
	SCHEDULER_IDENT = "firehose"
)

type Firehose struct {
	log *conf.Log
}

func NewFirehose() *Firehose {
	return &Firehose{
		log: conf.NewLog(),
	}
}

func (f *Firehose) Stream() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	rsc := &events.RepoStreamCallbacks{
		RepoCommit: func(evt *atproto.SyncSubscribeRepos_Commit) error {
			f.log.Info("Event", "repo", evt.Repo)
			for _, op := range evt.Ops {
				f.log.Info("record", "action", op.Action, "path", op.Path)
			}
			return nil
		},
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, BSKY_WSS_URL, http.Header{})
	if err != nil {
		f.log.WithError(err, "Error connecting to bsky firehose", "url", BSKY_WSS_URL)
		return err
	}
	defer conn.Close()
	sched := sequential.NewScheduler(SCHEDULER_IDENT, rsc.EventHandler)
	return events.HandleRepoStream(ctx, conn, sched, f.log.Logger)
}
