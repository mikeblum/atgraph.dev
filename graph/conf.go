package graph

import (
	"time"

	"github.com/mikeblum/atproto-graph-viz/conf"
)

type Conf struct {
	conf.EnvConf
}

func NewConf() *Conf {
	return &Conf{conf.NewEnvConf()}
}

func (c *Conf) uri() string {
	return c.GetEnv(ENV_NEO4J_URI, NEO4J_URI)
}

func (c *Conf) database() string {
	return c.GetEnv(ENV_NEO4J_DATABASE, NEO4J_DATABASE)
}

func (c *Conf) timeout() time.Duration {
	var timeout time.Duration
	var err error
	if timeout, err = time.ParseDuration(c.GetEnv(ENV_NEO4J_TIMEOUT, NEO4J_TIMEOUT.String())); err != nil {
		return NEO4J_TIMEOUT
	}
	return timeout
}
