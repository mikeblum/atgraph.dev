package bsky

import (
	"context"
	"time"
)

func (c *Client) BackfillRepos(ctx context.Context, stream func(context.Context, chan *Item) error) error {
	var next string
	var err error
	for {
		if next, err = c.listRepos(ctx, next, stream); err != nil {
			c.log.WithError(err, "Error fetching bsky repos")
			return err
		}
		if next == "" {
			break
		}
	}
	time.Sleep(time.Second)
	return nil
}
