package bsky

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/bluesky-social/indigo/api/atproto"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func (c *Client) BackfillRepos(ctx context.Context, pool *WorkerPool) error {
	g, ctx := errgroup.WithContext(ctx)

	// Process results
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				c.log.WithError(ctx.Err()).Error("Context done - exiting...")
				return ctx.Err()
			case err, ok := <-pool.results:
				if !ok {
					// results closed out
					return nil
				}
				if err != nil {
					c.log.WithError(err).Error("Error processing results - exiting...")
					continue
				}
				pool.metrics.jobsInflight.Add(ctx, -1)
				pool.jobsInflight.Add(-1)
			}
		}
	})

	// Limit comcurrent page fetches to worker count
	sem := semaphore.NewWeighted(int64(c.conf.WorkerCount()))
	cursor := atomic.Value{}
	cursor.Store((*string)(nil))
	done := atomic.Bool{}

	for page := 1; !done.Load(); page++ {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		currentPage := page
		currentCursor := cursor.Load().(*string)

		// submit pages throttled by worker pool size
		g.Go(func() error {
			defer sem.Release(1)

			var nextCursor *string
			var err error
			if nextCursor, err = c.listRepos(ctx, currentCursor, currentPage, pool); err != nil {
				return err
			}

			cursor.Store(nextCursor)
			// short-circuit if no more pages to process
			if nextCursor == nil || *nextCursor == "" {
				done.Store(true)
				c.log.With("total-pages", page).Info("Submitted repos for ingestion")
			}
			return nil
		})

	}

	return g.Wait()
}

func (c *Client) listRepos(ctx context.Context, next *string, page int, pool *WorkerPool) (*string, error) {
	var repos *atproto.SyncListRepos_Output
	var err error

	pageSize := int64(c.conf.PageSize())
	var cursor string
	if next != nil {
		cursor = *next
	}
	if repos, err = atproto.SyncListRepos(ctx, c.atproto, cursor, pageSize); err != nil {
		if !suppressATProtoErr(err) {
			c.log.WithErrorMsg(err, "Error fetching bsky repo", "next", next)
		}
		return next, err
	}

	c.log.With("action", "list-repos", "next", next, "page", page, "page-size", pageSize, "repos", len(repos.Repos)).Info("Fetching Bluesky repos")

	// limit conncurrent processing of repos to worker pool
	g, ctx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(int64(c.conf.WorkerCount()))

	for _, repo := range repos.Repos {
		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}

		g.Go(func() error {
			defer sem.Release(1)

			if filterRepo(repo) {
				return nil
			}

			// Increment count before submitting
			pool.jobsInflight.Add(1)
			pool.metrics.jobsInflight.Add(ctx, 1)

			if err = pool.Submit(ctx, RepoJob{
				repo: repo,
			}); err != nil {
				// Decrement count on submission failure
				pool.jobsInflight.Add(-1)
				pool.metrics.jobsInflight.Add(ctx, -1)
				c.log.WithErrorMsg(err, "Error submitting bsky repo for ingestion", "did", repo.Did)
			}
			return nil
		})
	}

	prev := c.cursor
	if prev == repos.Cursor {
		return nil, fmt.Errorf("!!listRepos: cursor not advancing: %s", *prev)
	}
	c.cursor = repos.Cursor
	return c.cursor, nil
}

func filterRepo(repo *atproto.SyncListRepos_Repo) bool {
	if repo == nil {
		return true
	}
	if !*repo.Active {
		return true
	}
	return false
}
