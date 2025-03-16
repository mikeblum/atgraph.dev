package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/mikeblum/atproto-graph-viz/conf"
	"github.com/mikeblum/atproto-graph-viz/graph"
	"github.com/mikeblum/atproto-graph-viz/version"
)

type Engine struct {
	conf *Conf
	db   *sql.DB
	log  *conf.Log
}

func NewEngine(ctx context.Context) (graph.Engine, error) {
	var build *debug.Module
	var ok bool
	if build, ok = version.BuildVersion(); !ok {

	}
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "atgraph",
			Username: "default",
			Password: "",
		},
		// TLS: &tls.Config{
		// 	InsecureSkipVerify: true,
		// },
		Settings: clickhouse.Settings{
			"max_execution_time": time.Second * 60,
		},
		DialTimeout: time.Second * 30,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Debug:                true,
		BlockBufferSize:      10,
		MaxCompressionBuffer: 10240,
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "atgraph", Version: build.Version},
			},
		},
	})
	conn.SetMaxIdleConns(5)
	conn.SetMaxOpenConns(10)
	// conn.SetConnMaxLifetime(time.Hour)

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	engine := &Engine{
		conf: NewConf(),
		db:   conn,
		log:  conf.NewLog(),
	}
	return engine, engine.LoadSchema(ctx)
}

func (e *Engine) LoadSchema(ctx context.Context) error {
	schemaBytes, err := os.ReadFile("./sql/schema-clickhouse.sql")
	if err != nil {
		return fmt.Errorf("failed to read ClickHouse schema file: %w", err)
	}

	// Execute the schema
	_, err = e.db.Exec(string(schemaBytes))
	if err != nil {
		return fmt.Errorf("failed to initialize ClickHouse schema: %w", err)
	}

	return nil
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
	return e.db.Close()
}

func (e *Engine) Ingest(ctx context.Context, workerID int, item bsky.RepoItem) error {
	return fmt.Errorf("ingest not supported - use IngestEngine")
}

// validate graph.Engine interface is implemented
var _ graph.Engine = &Engine{}
