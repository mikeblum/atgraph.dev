package bsky

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/repo"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/ipfs/go-cid"
	"github.com/mikeblum/atproto-graph-viz/conf"
)

type Item struct {
	Data  any                `json:"data"`
	DID   syntax.DID         `json:"did"`
	Ident *identity.Identity `json:"ident"`
	NSID  syntax.NSID        `json:"nsid"`
}

type repoContext struct {
	Did string `json:"did"` // DID persistent, long-term identifiers for every account.
	Rev string `json:"rev"` // REV revision number of the repo.
}

func (c *Client) listRepos(ctx context.Context, next string, stream func(items []*Item) error) (string, chan error) {
	var repos *atproto.SyncListRepos_Output
	var err error
	errs := make(chan error)

	pageSize := int64(pageSize())
	if repos, err = atproto.SyncListRepos(ctx, c.client, next, pageSize); err != nil {
		errs <- err
		return next, errs
	}

	c.log.With("action", "list-repos", "next", next, "page-size", pageSize, "repos", len(repos.Repos)).Info("Fetching Bluesky repo")

	for _, repo := range repos.Repos {
		go func() {
			active := repo.Active != nil && *repo.Active
			if !active {
				return
			}
			var items []*Item
			if items, err = c.getRepo(ctx, repoContext{Did: repo.Did, Rev: repo.Rev}); err != nil {
				c.log.WithError(err, "Error fetching Bluesky repo", "did", repo.Did, "head", repo.Head, "rev", repo.Rev)
			}
			// enqueue bsky items for ingestion
			c.log.With("action", "get-repo", "did", repo.Did, "items", len(items)).Info("Fetching Bluesky items")
			stream(items)
		}()
	}

	return *repos.Cursor, errs
}

func (c *Client) getRepo(ctx context.Context, repoCtx repoContext) ([]*Item, error) {
	var repoData []byte
	var err error
	var ident *identity.Identity
	var atid *syntax.AtIdentifier
	if atid, err = syntax.ParseAtIdentifier(repoCtx.Did); err != nil {
		return nil, err
	}
	if ident, err = identity.DefaultDirectory().Lookup(ctx, *atid); err != nil {
		return nil, err
	}
	xrpcc := xrpc.Client{
		Host: ident.PDSEndpoint(),
	}
	if xrpcc.Host == "" {
		return nil, fmt.Errorf("no PDS endpoint for identity: %s", atid)
	}
	if repoData, err = atproto.SyncGetRepo(ctx, &xrpcc, ident.DID.String(), ""); err != nil {
		c.log.WithError(err, "Error fetching bsky repo")
		return nil, err
	}
	var r *repo.Repo
	if r, err = repo.ReadRepoFromCar(context.Background(), bytes.NewReader(repoData)); err != nil {
		c.log.WithError(err, "Error reading bsky repo")
		return nil, err
	}
	return c.resolveLexicon(ctx, ident, r)
}

func (c *Client) resolveLexicon(ctx context.Context, ident *identity.Identity, r *repo.Repo) ([]*Item, error) {
	items := make([]*Item, 0, pageSize())
	// extract DID from repo commit
	var did syntax.DID
	var err error
	sc := r.SignedCommit()
	if did, err = syntax.ParseDID(sc.Did); err != nil {
		c.log.With("err", err).Warn("unrecognized did")
	}
	err = r.ForEach(ctx, "", func(k string, v cid.Cid) error {
		var data any
		var ok bool
		var rec repo.CborMarshaler
		if _, rec, err = r.GetRecord(ctx, k); err != nil {
			c.log.With("err", err, "did", did, "key", k).Warn("unrecognized lexicon type")
			return nil
		}
		nsid := syntax.NSID(strings.SplitN(k, "/", 2)[0]).Normalize()

		switch nsid {
		case ITEM_FEED_POST:
			if data, ok = rec.(*bsky.FeedPost); !ok {
				return fmt.Errorf("found wrong type in feed post location in tree: %s", did)
			}
			err = unsupportedLexicon(ITEM_FEED_POST)
		case ITEM_ACTOR_PROFILE:
			if data, ok = rec.(*bsky.ActorProfile); !ok {
				return fmt.Errorf("found wrong type in actor location in tree: %s", did)
			}
		case ITEM_GRAPH_FOLLOW:
			if data, ok = rec.(*bsky.GraphFollow); !ok {
				return fmt.Errorf("found wrong type in follow location in tree: %s", did)
			}
			err = unsupportedLexicon(ITEM_GRAPH_FOLLOW)
		case ITEM_FEED_REPOST:
			if data, ok = rec.(*bsky.FeedRepost); !ok {
				return fmt.Errorf("found wrong type in repost location in tree: %s", did)
			}
			err = unsupportedLexicon(ITEM_FEED_REPOST)
		case ITEM_FEED_LIKE:
			if data, ok = rec.(*bsky.FeedLike); !ok {
				return fmt.Errorf("found wrong type in like location in tree: %s", did)
			}
			err = unsupportedLexicon(ITEM_FEED_LIKE)
		case ITEM_GRAPH_BLOCK:
			if data, ok = rec.(*bsky.GraphBlock); !ok {
				return fmt.Errorf("found wrong type in block location in tree: %s", did)
			}
			err = unsupportedLexicon(ITEM_GRAPH_BLOCK)
		case ITEM_GRAPH_LIST_BLOCK:
			if data, ok = rec.(*bsky.GraphListblock); !ok {
				return fmt.Errorf("found wrong type in listblock location in tree: %s", did)
			}
			err = unsupportedLexicon(ITEM_GRAPH_LIST_BLOCK)
		case ITEM_GRAPH_LIST:
			if data, ok = rec.(*bsky.GraphList); !ok {
				return fmt.Errorf("found wrong type in list location in tree: %s", did)
			}
			err = unsupportedLexicon(ITEM_GRAPH_LIST)
		case ITEM_GRAPH_LIST_ITEM:
			if data, ok = rec.(*bsky.GraphListitem); !ok {
				return fmt.Errorf("found wrong type in listitem location in tree: %s", did)
			}
			err = unsupportedLexicon(ITEM_GRAPH_LIST_ITEM)
		default:
			err = fmt.Errorf("found unknown type in tree")
			c.log.With("err", err, "did", did, "nsid", nsid).Warn("unrecognized lexicon type")
		}
		if err != nil {
			return err
		}

		items = append(items, &Item{
			Data:  data,
			DID:   did,
			Ident: ident,
			NSID:  nsid,
		})

		return nil
	})

	return items, err
}

func pageSize() int {
	var pageSize int
	var err error
	if pageSize, err = strconv.Atoi(conf.GetEnv(ENV_BSKY_PAGE_SIZE, strconv.Itoa(DEFAULT_PAGE_SIZE))); err != nil {
		pageSize = DEFAULT_PAGE_SIZE
	}
	return pageSize
}

func unsupportedLexicon(nsid syntax.NSID) error {
	return fmt.Errorf("TODO: unsupported lexicon: %s", nsid.String())
}
