package bsky

import (
	"context"
	"fmt"
	"sync"

	"github.com/bluesky-social/indigo/api/atproto"
	"golang.org/x/sync/errgroup"
)

func (c *Client) BackfillRepos(ctx context.Context, pool *WorkerPool) error {
	var (
		page int
		next *string
		err  error
		wg   sync.WaitGroup
		errs = make(chan error, 1)
	)

	g, ctx := errgroup.WithContext(ctx)

	done := make(chan bool)

	// await repos to ingest
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-pool.results:
				if err != nil {
					return err
				}
				pool.metrics.jobsInflight.Add(ctx, -1)
				if pool.jobsInflight.Add(-1) == 0 {
					defer close(done)
					return nil
				}
			}
		}
	})

	for {
		wg.Add(1)
		page++

		// Capture loop variables to prevent closure issues
		currentPage := page
		currentNext := next

		go func() {
			defer wg.Done()

			if next, err = c.listRepos(ctx, currentNext, currentPage, pool); err != nil {
				select {
				case errs <- err:
				default:
				}
				return
			}

			if next == nil || *next == "" {
				select {
				case errs <- nil:
				default:
				}
			}
		}()

		select {
		case err = <-errs:
			wg.Wait()
			return g.Wait()
		}
	}
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

	for _, repo := range repos.Repos {
		go func(repo *atproto.SyncListRepos_Repo) {
			if filterRepo(repo) {
				return
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
		}(repo)
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
