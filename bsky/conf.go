package bsky

import (
	"strconv"

	"github.com/mikeblum/atproto-graph-viz/conf"
)

type Conf struct{}

func NewConf() *Conf {
	return &Conf{}
}

func (c *Conf) PageSize() int {
	var pageSize int
	var err error
	if pageSize, err = strconv.Atoi(conf.GetEnv(ENV_BSKY_PAGE_SIZE, strconv.Itoa(DEFAULT_PAGE_SIZE))); err != nil {
		pageSize = DEFAULT_PAGE_SIZE
	}
	return pageSize
}

func (c *Conf) WorkerCount() int {
	var pageSize int
	var err error
	if pageSize, err = strconv.Atoi(conf.GetEnv(ENV_BSKY_REPO_WORKER_COUNT, strconv.Itoa(DEFAULT_WORKER_COUNT))); err != nil {
		pageSize = DEFAULT_WORKER_COUNT
	}
	return pageSize
}

func (c *Conf) MaxRetries() int {
	var maxRetries int
	var err error
	if maxRetries, err = strconv.Atoi(conf.GetEnv(ENV_BSKY_MAX_RETRY_COUNT, strconv.Itoa(DEFAULT_MAX_RETRIES))); err != nil {
		maxRetries = DEFAULT_MAX_RETRIES
	}
	return maxRetries
}
