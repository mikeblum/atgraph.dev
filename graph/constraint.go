package graph

import (
	"context"
	"errors"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	uniq_profile_id        = `CREATE CONSTRAINT uniq_profile_id IF NOT EXISTS FOR (n:Profile) REQUIRE (n.id) IS UNIQUE;`
	uniq_profile_handle_id = `CREATE CONSTRAINT uniq_profile_handle_id IF NOT EXISTS FOR (n:Profile) REQUIRE (n.handle, n.id) IS UNIQUE;`
)

func (e *Engine) CreateConstraints(ctx context.Context) error {
	var err error
	constraints := []string{
		uniq_profile_id,
		uniq_profile_handle_id,
	}
	for _, constraint := range constraints {
		next := constraint
		_, err := neo4j.ExecuteQuery(ctx, e.driver,
			next,
			nil, neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(database()))
		err = errors.Join(err)
	}
	return err
}
