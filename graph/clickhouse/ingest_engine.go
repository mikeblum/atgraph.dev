package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ClickHouse/ch-go"
	"github.com/mikeblum/atgraph.dev/conf"
	"github.com/mikeblum/atgraph.dev/graph"
)

const (
	// Clickhouse database schema (incompat with sqlc)
	schemaCreateDb              = `CREATE DATABASE IF NOT EXISTS atgraph COMMENT 'atgraph.dev ClickHouse database';`
	schemaCreateDbO11yMetrics   = `CREATE DATABASE IF NOT EXISTS atgraph_o11y_metrics COMMENT 'metrics.o11y.atgraph.dev ClickHouse metrics database';`
	schemaCreateDbO11yTraces    = `CREATE DATABASE IF NOT EXISTS atgraph_o11y_traces COMMENT 'traces.o11y.atgraph.dev ClickHouse traces database';`
	schemaCreateDbO11yLogs      = `CREATE DATABASE IF NOT EXISTS atgraph_o11y_logs COMMENT 'logs.o11y.atgraph.dev ClickHouse logs database';`
	schemaCreateDbSignozMetrics = `CREATE DATABASE IF NOT EXISTS signoz_metrics COMMENT 'signoz metrics database';`
	schemaCreateDbSignozTraces  = `CREATE DATABASE IF NOT EXISTS signoz_traces COMMENT 'signoz traces database';`
	schemaCreateDbSignozLogs    = `CREATE DATABASE IF NOT EXISTS signoz_logs COMMENT 'signoz logs database';`
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
	return errors.Join(e.loadSchemas(ctx, schemaCreateDb, schemaCreateDbO11yMetrics,
		schemaCreateDbO11yTraces, schemaCreateDbO11yLogs), e.loadSchemas(ctx, schemaCreateDbSignozMetrics, schemaCreateDbSignozTraces, schemaCreateDbSignozLogs))
}

func (e *IngestEngine) loadSchemas(ctx context.Context, schemas ...string) error {
	var conn *ch.Client
	var err error
	if conn, err = newConn(ctx); err != nil {
		return err
	}
	defer conn.Close()
	for _, schema := range schemas {
		if err = conn.Do(ctx, ch.Query{
			Body: schema,
		}); err != nil {
			e.log.WithError(err).Error("Error creating ClickHouse db", "schema", schema)
			return err
		}
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
