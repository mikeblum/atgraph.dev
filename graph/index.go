package graph

import (
	"context"
	"errors"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	idx_profile_created = `CREATE INDEX idx_profile_created IF NOT EXISTS FOR (n:Profile) ON (n.created);`
	idx_profile_updated = `CREATE INDEX idx_profile_updated IF NOT EXISTS FOR (n:Profile) ON (n.updated);`
)

func (e *Engine) CreateIndexes(ctx context.Context) error {
	var err error
	indexes := []string{
		idx_profile_created,
		idx_profile_updated,
	}
	for _, idx := range indexes {
		next := idx
		_, err := neo4j.ExecuteQuery(ctx, e.driver,
			next,
			nil, neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(database()))
		err = errors.Join(err)
	}
	return err
}
