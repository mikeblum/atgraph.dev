package postgres

import "github.com/mikeblum/atproto-graph-viz/conf"

type Conf struct {
	conf.EnvConf
}

func NewConf() *Conf {
	return &Conf{conf.NewEnvConf()}
}
