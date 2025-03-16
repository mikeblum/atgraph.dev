package neo4j

import (
	"context"

	"github.com/mikeblum/atproto-graph-viz/conf"
	"github.com/mikeblum/atproto-graph-viz/graph"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Engine struct {
	conf    *Conf
	driver  neo4j.DriverWithContext
	session neo4j.SessionConfig
	log     *conf.Log
}

func NewEngine(ctx context.Context) (graph.Engine, error) {
	var driver neo4j.DriverWithContext
	var err error

	cfg := NewConf()

	if driver, err = neo4j.NewDriverWithContext(
		cfg.uri(),
		neo4j.NoAuth(),
	); err != nil {
		return nil, err
	}
	log := conf.NewLog()

	return &Engine{
		conf:   cfg,
		driver: driver,
		session: neo4j.SessionConfig{
			DatabaseName: cfg.database(),
			BoltLogger:   neo4jLogBridge(log),
		},
		log: log,
	}, driver.VerifyConnectivity(ctx)
}

func (e *Engine) LoadSchema(ctx context.Context) error {
	// no-op since we don't load a schema beforehand
	return nil
}

func (e *Engine) Close(ctx context.Context) error {
	return e.driver.Close(ctx)
}

// validate graph.Engine interface is implemented
var _ graph.Engine = &Engine{}
