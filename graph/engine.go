package graph

import (
	"context"
	"time"

	"github.com/mikeblum/atproto-graph-viz/conf"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Engine struct {
	driver neo4j.DriverWithContext
	config neo4j.SessionConfig
	log    *conf.Log
}

func Bootstrap(ctx context.Context) (*Engine, error) {
	var driver neo4j.DriverWithContext
	var err error
	uri := conf.GetEnv(ENV_NEO4J_URI, NEO4J_URI)
	if driver, err = neo4j.NewDriverWithContext(
		uri,
		neo4j.NoAuth(),
	); err != nil {
		return nil, err
	}
	log := conf.NewLog()
	return &Engine{
		driver: driver,
		config: neo4j.SessionConfig{
			DatabaseName: database(),
			BoltLogger:   neo4jLogBridge(log),
		},
		log: log,
	}, driver.VerifyConnectivity(ctx)
}

func (e *Engine) Close(ctx context.Context) error {
	return e.driver.Close(ctx)
}

func database() string {
	return conf.GetEnv(ENV_NEO4J_DATABASE, NEO4J_DATABASE)
}

func timeout() time.Duration {
	var timeout time.Duration
	var err error
	if timeout, err = time.ParseDuration(conf.GetEnv(ENV_NEO4J_TIMEOUT, NEO4J_TIMEOUT.String())); err != nil {
		return NEO4J_TIMEOUT
	}
	return timeout
}
