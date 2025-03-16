package clickhouse

import (
	"context"
	"fmt"
	"os"

	"github.com/ClickHouse/ch-go"
	"github.com/mikeblum/atgraph.dev/conf"
	"github.com/mikeblum/atgraph.dev/graph"
)

const (
	// Clickhouse database schema (incompat with sqlc)
	schemaCreateDb = `CREATE DATABASE IF NOT EXISTS atgraph ENGINE = postgresql COMMENT 'atgraph.dev ClickHouse database';`
)

type IngestEngine struct {
	conf *Conf
	log  *conf.Log
}

// NewIngestEngine: low-level ch-go impl for bulk inserts
// https://clickhouse.com/docs/integrations/go#choosing-a-client
func NewIngestEngine(ctx context.Context) (graph.Engine, error) {
	engine := &IngestEngine{
		conf: NewConf(),
		log:  conf.NewLog(),
	}
	return engine, engine.LoadSchema(ctx)
}

func newConn(ctx context.Context) (*ch.Client, error) {
	var conn *ch.Client
	var err error
	if conn, err = ch.Dial(ctx, ch.Options{
		ClientName:  "atgraph.dev:ingest",
		Database:    "atgraph",
		Compression: ch.CompressionLZ4,
	}); err != nil {
		return nil, err
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}
	return conn, err
}

func (e *IngestEngine) LoadSchema(ctx context.Context) error {
	var conn *ch.Client
	var err error
	if conn, err = newConn(ctx); err != nil {
		return err
	}
	defer conn.Close()
	if err = conn.Do(ctx, ch.Query{
		Body: schemaCreateDb,
	}); err != nil {
		e.log.WithError(err).Error("Error creating ClickHouse db")
		return err
	}

	var schemaBytes []byte
	if schemaBytes, err = os.ReadFile("./sql/schema-clickhouse.sql"); err != nil {
		e.log.WithError(err).Error("failed to read ClickHouse schema file")
		return err
	}

	return conn.Do(ctx, ch.Query{
		Body: string(schemaBytes),
	})
}

func (e *IngestEngine) CreateIndexes(ctx context.Context) error {
	// no-op as sqlc generates the schema
	return nil
}

func (e *IngestEngine) CreateConstraints(ctx context.Context) error {
	// no-op as sqlc generates the schema
	return nil
}

func (e *IngestEngine) Close(ctx context.Context) error {
	// ch-db client connections should be closed via defer
	return fmt.Errorf("IngestEngine.Close(context.Context) error not supported")
}

// validate graph.Engine interface is implemented
var _ graph.Engine = &IngestEngine{}
