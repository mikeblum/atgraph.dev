package bsky

import (
	"strconv"

	"github.com/mikeblum/atproto-graph-viz/conf"
)

type Conf struct {
	conf.EnvConf
}

func NewConf() *Conf {
	return &Conf{conf.NewEnvConf()}
}

func (c *Conf) host() string {
	return c.GetEnv(BSKY_APP_VIEW_URL, BSKY_ENTRYWAY_URL)
}

func (c *Conf) identifier() string {
	return c.GetEnv(ENV_BSKY_IDENTIFIER, "")
}

func (c *Conf) password() string {
	return c.GetEnv(ENV_BSKY_PASSWORD, "")
}

func (c *Conf) PageSize() int {
	var pageSize int
	var err error
	if pageSize, err = strconv.Atoi(c.GetEnv(ENV_BSKY_PAGE_SIZE, strconv.Itoa(DEFAULT_PAGE_SIZE))); err != nil {
		pageSize = DEFAULT_PAGE_SIZE
	}
	return pageSize
}

func (c *Conf) WorkerCount() int {
	var pageSize int
	var err error
	if pageSize, err = strconv.Atoi(c.GetEnv(ENV_BSKY_WORKER_COUNT, strconv.Itoa(DEFAULT_WORKER_COUNT))); err != nil {
		pageSize = DEFAULT_WORKER_COUNT
	}
	return pageSize
}

func (c *Conf) MaxRetries() int {
	var maxRetries int
	var err error
	if maxRetries, err = strconv.Atoi(c.GetEnv(ENV_BSKY_MAX_RETRY_COUNT, strconv.Itoa(DEFAULT_MAX_RETRIES))); err != nil {
		maxRetries = DEFAULT_MAX_RETRIES
	}
	return maxRetries
}
