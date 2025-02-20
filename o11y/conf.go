package o11y

import "github.com/mikeblum/atproto-graph-viz/conf"

type Conf struct {
	conf.EnvConf
}

func NewConf() *Conf {
	return &Conf{conf.NewEnvConf()}
}

func (c *Conf) o11yEndpoint() string {
	return c.GetEnv(ENV_OTEL_EXPORTER_OTLP_ENDPOINT, DEFAULT_OTEL_OTLP_ENDPOINT)
}

func (c *Conf) env() string {
	return c.GetEnv(ENV, DEFAULT_ENV)
}
