package main

import (
	"context"
	"os"

	"github.com/mikeblum/atgraph.dev/bsky"
	"github.com/mikeblum/atgraph.dev/conf"
	"github.com/mikeblum/atgraph.dev/graph"
	"github.com/mikeblum/atgraph.dev/graph/clickhouse"
	"github.com/mikeblum/atgraph.dev/o11y"
)

func main() {
	log := conf.NewLog()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var client *bsky.Client
	var err error

	// configure o11y
	if _, err = o11y.NewO11y(ctx, log); err != nil {
		log.WithErrorMsg(err, "Error bootstrapping OTEL o11y")
		exit()
	}
	defer o11y.Cleanup(ctx)

	var engine graph.Engine
	if engine, err = clickhouse.NewIngestEngine(ctx); err != nil {
		log.WithErrorMsg(err, "Error bootstrapping clickhouse driver")
		exit()
	}

	defer engine.Close(ctx)

	// create indexes
	if err = engine.CreateIndexes(ctx); err != nil {
		log.WithErrorMsg(err, "Error creating indexes")
		exit()
	}

	// create constraints
	if err = engine.CreateConstraints(ctx); err != nil {
		log.WithErrorMsg(err, "Error creating constraints")
		exit()
	}

	// init authenticated bsky client
	if client, err = bsky.NewSyncClient(); err != nil {
		log.WithErrorMsg(err, "Error creating bsky sync client")
		exit()
	}

	// bootstrap worker pool
	var pool *bsky.WorkerPool
	if pool, err = bsky.NewWorkerPool(ctx, client, bsky.NewConf()); err != nil {
		log.WithErrorMsg(err, "Error initing worker pool")
		exit()
	}
	pool.StartMonitor(ctx).WithIngest(engine.Ingest)
	go func() {
		if err = pool.Start(ctx); err != nil {
			log.WithErrorMsg(err, "Error starting bsky worker pool")
			cancel() // cancel context if worker pool fails to start
		}
	}()

	// Wait for pool to be ready
	select {
	case <-pool.PoolReady():
		log.Info("Worker pool ready")
	case <-ctx.Done():
		log.WithErrorMsg(ctx.Err(), "Context cancelled before pool was ready")
		return
	}

	// Wait for ingest to be ready
	select {
	case <-pool.IngestReady():
		log.Info("Repo ingest ready")
	case <-ctx.Done():
		log.WithErrorMsg(ctx.Err(), "Context cancelled before ingest was ready")
		return
	}

	// Channel to signal backfill completion
	done := make(chan bool)

	// Start backfill in the background
	go func() {
		defer close(done)
		if err := client.BackfillRepos(ctx, pool); err != nil {
			log.WithErrorMsg(err, "Error backfilling bsky repos")
			cancel()
			return
		}
	}()

	// Await backfill to complete or be cancelled
	select {
	case <-done:
		log.Info("Bsky backfill successful ✅")
	case <-ctx.Done():
		log.WithErrorMsg(ctx.Err(), "Error backfilling bsky repos ❌")
	}
}

func exit() {
	os.Exit(1)
}
