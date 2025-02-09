package graph

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	indexes = `CREATE INDEX idx_profile_id FOR (n:Profile) ON (n.id);`
)

func (e *Engine) CreateIndexes(ctx context.Context) error {
	var err error
	_, err = neo4j.ExecuteQuery(ctx, e.driver,
		indexes,
		nil, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(database()))
	return err
}
