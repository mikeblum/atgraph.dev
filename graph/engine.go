package graph

import (
	"context"

	"github.com/mikeblum/atgraph.dev/bsky"
)

type Engine interface {
	Ingest(ctx context.Context, workerID int, item bsky.RepoItem) error
	LoadSchema(ctx context.Context) error
	CreateIndexes(ctx context.Context) error
	CreateConstraints(ctx context.Context) error
	Close(ctx context.Context) error
}
