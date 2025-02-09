package bsky

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/repo"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/ipfs/go-cid"
	"golang.org/x/sync/errgroup"
)

const (
	PAGE_SIZE = 10
)

type RepoContext struct {
	Did string `json:"did"` // DID persistent, long-term identifiers for every account.
	Rev string `json:"rev"` // REV revision number of the repo.
}

func (c *Client) ListRepos() error {
	var ctx = context.Background()
	var repos *atproto.SyncListRepos_Output
	var err error
	var cursor string
	if repos, err = atproto.SyncListRepos(ctx, c.client, cursor, PAGE_SIZE); err != nil {
		return err
	}
	errs, ctx := errgroup.WithContext(ctx)
	for _, repo := range repos.Repos {
		active := repo.Active != nil && *repo.Active
		if !active {
			continue
		}
		c.log.With("did", repo.Did, "head", repo.Head, "rev", repo.Rev).Debug("bsky repo")
		// TODO: worker pool
		errs.Go(func() error {
			return c.GetRepo(context.TODO(), RepoContext{Did: repo.Did, Rev: repo.Rev})
		})
	}
	return errs.Wait()
}

func (c *Client) GetRepo(ctx context.Context, repoCtx RepoContext) error {
	var repoData []byte
	var err error
	var atid *syntax.AtIdentifier
	if atid, err = syntax.ParseAtIdentifier(repoCtx.Did); err != nil {
		return err
	}
	ident, err := identity.DefaultDirectory().Lookup(ctx, *atid)
	if err != nil {
		return err
	}
	xrpcc := xrpc.Client{
		Host: ident.PDSEndpoint(),
	}
	if xrpcc.Host == "" {
		return fmt.Errorf("no PDS endpoint for identity: %s", atid)
	}
	if repoData, err = atproto.SyncGetRepo(ctx, &xrpcc, ident.DID.String(), ""); err != nil {
		c.log.WithError(err, "Error fetching bsky repo")
		return err
	}
	// TODO: worker pool
	var r *repo.Repo
	if r, err = repo.ReadRepoFromCar(context.Background(), bytes.NewReader(repoData)); err != nil {
		c.log.WithError(err, "Error reading bsky repo")
		return err
	}
	if err = c.resolveLexicon(ctx, r); err != nil {
		c.log.WithError(err, "Error parsing bsky repo")
		return err
	}
	return nil
}

func (c *Client) resolveLexicon(ctx context.Context, r *repo.Repo) error {
	// extract DID from repo commit
	var did syntax.DID
	var err error
	sc := r.SignedCommit()
	if did, err = syntax.ParseDID(sc.Did); err != nil {
		c.log.With("err", err).Warn("unrecognized did")
	}
	return r.ForEach(ctx, "", func(k string, v cid.Cid) error {
		var rec repo.CborMarshaler
		if _, rec, err = r.GetRecord(ctx, k); err != nil {
			c.log.With("err", err, "did", did).Warn("unrecognized lexicon type: %s")
			return nil
		}
		nsid := strings.SplitN(k, "/", 2)[0]

		switch nsid {
		case "app.bsky.feed.post":
			if _, ok := rec.(*bsky.FeedPost); !ok {
				return fmt.Errorf("found wrong type in feed post location in tree: %s", did)
			}
		case "app.bsky.actor.profile":
			if _, ok := rec.(*bsky.ActorProfile); !ok {
				return fmt.Errorf("found wrong type in actor location in tree: %s", did)
			}
		case "app.bsky.graph.follow":
			if _, ok := rec.(*bsky.GraphFollow); !ok {
				return fmt.Errorf("found wrong type in follow location in tree: %s", did)
			}
		case "app.bsky.feed.repost":
			if _, ok := rec.(*bsky.FeedRepost); !ok {
				return fmt.Errorf("found wrong type in repost location in tree: %s", did)
			}
		case "app.bsky.feed.like":
			if _, ok := rec.(*bsky.FeedLike); !ok {
				return fmt.Errorf("found wrong type in like location in tree: %s", did)
			}
		case "app.bsky.graph.block":
			if _, ok := rec.(*bsky.GraphBlock); !ok {
				return fmt.Errorf("found wrong type in block location in tree: %s", did)
			}
		case "app.bsky.graph.listblock":
			if _, ok := rec.(*bsky.GraphListblock); !ok {
				return fmt.Errorf("found wrong type in listblock location in tree: %s", did)
			}
		case "app.bsky.graph.list":
			if _, ok := rec.(*bsky.GraphList); !ok {
				return fmt.Errorf("found wrong type in list location in tree: %s", did)
			}
		case "app.bsky.graph.listitem":
			if _, ok := rec.(*bsky.GraphListitem); !ok {
				return fmt.Errorf("found wrong type in listitem location in tree: %s", did)
			}
		default:
			err := fmt.Errorf("found unknown type in tree: %s", did)
			c.log.With("err", err).Warn("unrecognized lexicon type")
		}
		return nil
	})
}
