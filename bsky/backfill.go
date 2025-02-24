package bsky

import (
	"context"

	"github.com/bluesky-social/indigo/api/atproto"
	"golang.org/x/sync/errgroup"
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
				}
				pool.metrics.jobsInflight.Add(ctx, -1)
				pool.jobsInflight.Add(-1)
			}
		}
	})

	var cursor *string
	page := 1

	for {
		next, err := c.listRepos(ctx, cursor, page, pool, g)
		if err != nil {
			c.log.WithError(err).Error("Error listing repo", "cursor", cursor, "page", page)
			return err
		}

		if next == nil || *next == "" {
			break
		}

		cursor = next
		page++
	}

	return g.Wait()
}

func (c *Client) listRepos(ctx context.Context, next *string, page int, pool *WorkerPool, g *errgroup.Group) (*string, error) {
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

	g.Go(func() error {
		for _, repo := range repos.Repos {
			if filterRepo(repo) {
				continue
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
		}
		return nil
	})

	return repos.Cursor, nil
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
