package bsky

import (
	"context"
	"strconv"
	"sync"

	"github.com/mikeblum/atproto-graph-viz/conf"
)

func (c *Client) BackfillRepos(ctx context.Context) error {
	var next string
	var errs chan error
	var wg sync.WaitGroup

	stream := make(chan *Item)

	workerCount := workerCount()
	// spin up backfill workers
	c.log.With("action", "backfill", "worker-count", workerCount).Info("Launching Bluesky Backfill Workers")
	for w := 1; w <= workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.backfillWorker(ctx, w, stream, errs)
		}()
	}
	for {
		next, errs = c.listRepos(ctx, next, func(items []*Item) error {
			for _, item := range items {
				stream <- item
			}
			return nil
		})
		if next == "" {
			break
		}
	}
	close(errs)
	close(stream)

	wg.Wait()
	return nil
}

func (c *Client) backfillWorker(ctx context.Context, id int, items <-chan *Item, errs chan<- error) {
	if err := c.ingest(ctx, items); err != nil {
		c.log.With("action", "backfill", "worker-id", id).Error("Error ingesting Bluesky items")
		errs <- err
	}
}

func workerCount() int {
	var workerCount int
	var err error
	if workerCount, err = strconv.Atoi(conf.GetEnv(ENV_BSKY_REPO_WORKER_COUNT, strconv.Itoa(DEFAULT_WORKER_COUNT))); err != nil {
		workerCount = DEFAULT_WORKER_COUNT
	}
	return workerCount
}
