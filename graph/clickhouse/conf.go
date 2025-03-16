package clickhouse

import "github.com/mikeblum/atgraph.dev/conf"

type Conf struct {
	conf.EnvConf
}

func NewConf() *Conf {
	return &Conf{conf.NewEnvConf()}
}
