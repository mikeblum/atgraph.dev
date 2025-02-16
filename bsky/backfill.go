package bsky

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/api/atproto"
)

func (c *Client) BackfillRepos(ctx context.Context, pool *WorkerPool) error {
	var page int
	var next *string
	var err error

	for {
		page = page + 1
		if next, err = c.listRepos(ctx, next, page+1, pool); err != nil {
			return err
		}
		if next == nil || *next == "" {
			break
		}
	}

	return nil
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

	go func(ctx context.Context) {
		for _, repo := range repos.Repos {
			if filterRepo(repo) {
				continue
			}
			if err = pool.Submit(ctx, RepoJob{
				repo: repo,
			}); err != nil {
				c.log.WithErrorMsg(err, "Error submitting bsky repo for ingestion", "did", repo.Did)
			}
		}
	}(ctx)

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
