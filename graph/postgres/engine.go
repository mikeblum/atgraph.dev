package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/mikeblum/atproto-graph-viz/conf"
	"github.com/mikeblum/atproto-graph-viz/graph"
)

const (
	connString = "dbname=atproto_graph_viz sslmode=none"
)

type Engine struct {
	conf    *Conf
	session *pgx.Conn
	log     *conf.Log
}

func NewEngine(ctx context.Context) (graph.Engine, error) {
	var conn *pgx.Conn
	var err error
	if conn, err = pgx.Connect(ctx, connString); err != nil {
		return nil, err
	}
	cfg := NewConf()
	return &Engine{
		conf:    cfg,
		session: conn,
		log:     conf.NewLog(),
	}, nil
}

func (e *Engine) CreateIndexes(ctx context.Context) error {
	// no-op as sqlc generates the schema
	return nil
}

func (e *Engine) CreateConstraints(ctx context.Context) error {
	// no-op as sqlc generates the schema
	return nil
}

func (e *Engine) Close(ctx context.Context) error {
	return e.session.Close(ctx)
}

// validate graph.Engine interface is implemented
var _ graph.Engine = &Engine{}
