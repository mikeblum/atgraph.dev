package bsky

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/api/atproto"
	"golang.org/x/sync/errgroup"
)

func (c *Client) BackfillRepos(ctx context.Context, pool *WorkerPool) error {
	var page int
	var next *string
	var err error
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
				if pool.jobCount.Add(-1) == 0 {
					defer close(done)
					return nil
				}
			}
		}
	})

	// submit repos
	for {
		page = page + 1
		if next, err = c.listRepos(ctx, next, page+1, pool); err != nil {
			return err
		}
		if next == nil || *next == "" {
			break
		}
	}

	go func() {
		<-ctx.Done()
		close(done)
	}()

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

	for _, repo := range repos.Repos {
		if filterRepo(repo) {
			continue
		}

		// Increment count before submitting
		pool.jobCount.Add(1)

		if err = pool.Submit(ctx, RepoJob{
			repo: repo,
		}); err != nil {
			// Decrement count on submission failure
			pool.jobCount.Add(-1)
			c.log.WithErrorMsg(err, "Error submitting bsky repo for ingestion", "did", repo.Did)
		}
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
