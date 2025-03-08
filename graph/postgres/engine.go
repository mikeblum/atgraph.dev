package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/mikeblum/atproto-graph-viz/conf"
)

type Engine struct {
	conf    *Conf
	session *pgx.Conn
	log     *conf.Log
}

func (e *Engine) Bootstrap(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, "dbname=atproto_graph_viz sslmode=none")
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	return nil
}

func (e *Engine) Ingest(ctx context.Context, workerID int, item bsky.RepoItem) error {
	return nil
}
