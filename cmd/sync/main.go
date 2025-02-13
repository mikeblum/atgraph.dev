package main

import (
	"context"
	"os"

	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/mikeblum/atproto-graph-viz/conf"
	"github.com/mikeblum/atproto-graph-viz/graph"
)

func main() {
	log := conf.NewLog()
	ctx := context.Background()
	var client *bsky.Client
	var err error

	var engine *graph.Engine
	if engine, err = graph.Bootstrap(ctx); err != nil {
		log.WithError(err, "Error bootstrapping neo4j driver")
		exit()
	}

	defer engine.Close(ctx)

	// create indexes
	if err = engine.CreateIndexes(ctx); err != nil {
		log.WithError(err, "Error creating indexes")
		exit()
	}

	// create constraints
	if err = engine.CreateConstraints(ctx); err != nil {
		log.WithError(err, "Error creating constraints")
		exit()
	}

	if client, err = bsky.New(); err != nil {
		log.WithError(err, "Error creating bsky client")
		exit()
	}

	if err = client.BackfillRepos(ctx, engine.Ingest); err != nil {
		log.WithError(err, "Error performing bsky backfill")
		exit()
	}
}

func exit() {
	os.Exit(1)
}
