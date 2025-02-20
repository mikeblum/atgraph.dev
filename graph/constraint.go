package graph

import (
	"context"
	"errors"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	uidx_profile_id        = `CREATE CONSTRAINT uidx_profile_id IF NOT EXISTS FOR (n:Profile) REQUIRE (n.id) IS UNIQUE;`
	uidx_profile_handle_id = `CREATE CONSTRAINT uidx_profile_handle_id IF NOT EXISTS FOR (n:Profile) REQUIRE (n.handle, n.id) IS UNIQUE;`
)

func (e *Engine) CreateConstraints(ctx context.Context) error {
	var err error
	constraints := []string{
		uidx_profile_id,
		uidx_profile_handle_id,
	}
	for _, constraint := range constraints {
		next := constraint
		_, uidxErr := neo4j.ExecuteQuery(ctx, e.driver,
			next,
			nil, neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(e.conf.database()))
		err = errors.Join(err, uidxErr)
	}
	return err
}
