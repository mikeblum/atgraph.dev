package bsky

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/repo"
	"github.com/ipfs/go-cid"
)

type RepoJob struct {
	repo *atproto.SyncListRepos_Repo
}

type RepoItem struct {
	repo    *repo.Repo
	Data    any                `json:"data"`
	DID     syntax.DID         `json:"did"`
	Err     error              `json:"err"`
	Rev     string             `json:"rev"`
	Sig     string             `json:"sig"`
	Ident   *identity.Identity `json:"ident"`
	NSID    syntax.NSID        `json:"nsid"`
	Version int64              `json:"version"`
}

type LexiconError struct {
	nsid syntax.NSID
}

func NewLexiconError(nsid syntax.NSID) *LexiconError {
	return &LexiconError{
		nsid: nsid,
	}
}

func (e *LexiconError) Error() string {
	return fmt.Sprintf("TODO: unsupported lexicon: %s", e.nsid.String())
}

func resolveLexicon(ctx context.Context, ident *identity.Identity, r *repo.Repo, items chan RepoItem) error {
	// extract DID from repo commit
	var did syntax.DID
	var err error
	sc := r.SignedCommit()
	if did, err = syntax.ParseDID(sc.Did); err != nil {
		return err
	}
	err = r.ForEach(ctx, "", func(k string, v cid.Cid) error {
		var data any
		var ok bool
		var rec repo.CborMarshaler
		if _, rec, err = r.GetRecord(ctx, k); err != nil {
			return err
		}
		nsid := syntax.NSID(strings.SplitN(k, "/", 2)[0]).Normalize()

		var lexiconErr *LexiconError
		switch nsid {
		case ITEM_FEED_POST:
			if data, ok = rec.(*bsky.FeedPost); !ok {
				return fmt.Errorf("found wrong type in feed post location in tree: %s", did)
			}
			lexiconErr = NewLexiconError(ITEM_FEED_POST)
		case ITEM_ACTOR_PROFILE:
			if data, ok = rec.(*bsky.ActorProfile); !ok {
				return fmt.Errorf("found wrong type in actor location in tree: %s", did)
			}
		case ITEM_GRAPH_FOLLOW:
			if data, ok = rec.(*bsky.GraphFollow); !ok {
				return fmt.Errorf("found wrong type in follow location in tree: %s", did)
			}
		case ITEM_FEED_REPOST:
			if data, ok = rec.(*bsky.FeedRepost); !ok {
				return fmt.Errorf("found wrong type in repost location in tree: %s", did)
			}
			lexiconErr = NewLexiconError(ITEM_FEED_REPOST)
		case ITEM_FEED_LIKE:
			if data, ok = rec.(*bsky.FeedLike); !ok {
				return fmt.Errorf("found wrong type in like location in tree: %s", did)
			}
			lexiconErr = NewLexiconError(ITEM_FEED_LIKE)
		case ITEM_GRAPH_BLOCK:
			if data, ok = rec.(*bsky.GraphBlock); !ok {
				return fmt.Errorf("found wrong type in block location in tree: %s", did)
			}
			lexiconErr = NewLexiconError(ITEM_GRAPH_BLOCK)
		case ITEM_GRAPH_LIST_BLOCK:
			if data, ok = rec.(*bsky.GraphListblock); !ok {
				return fmt.Errorf("found wrong type in listblock location in tree: %s", did)
			}
			lexiconErr = NewLexiconError(ITEM_GRAPH_LIST_BLOCK)
		case ITEM_GRAPH_LIST:
			if data, ok = rec.(*bsky.GraphList); !ok {
				return fmt.Errorf("found wrong type in list location in tree: %s", did)
			}
			lexiconErr = NewLexiconError(ITEM_GRAPH_LIST)
		case ITEM_GRAPH_LIST_ITEM:
			if data, ok = rec.(*bsky.GraphListitem); !ok {
				return fmt.Errorf("found wrong type in listitem location in tree: %s", did)
			}
			lexiconErr = NewLexiconError(ITEM_GRAPH_LIST_ITEM)
		default:
			return NewLexiconError(nsid)
		}

		if lexiconErr != nil {
			return lexiconErr
		}

		items <- RepoItem{
			repo:    r,
			Data:    data,
			Rev:     sc.Rev,
			Sig:     base64.StdEncoding.EncodeToString(sc.Sig),
			DID:     did,
			Ident:   ident,
			NSID:    nsid,
			Version: sc.Version,
		}

		return nil
	})

	return err
}
