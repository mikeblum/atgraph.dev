package bsky

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/mikeblum/atgraph.dev/conf"
)

type Client struct {
	atproto *xrpc.Client
	conf    *Conf
	session *atproto.ServerCreateSession_Output
	log     *conf.Log
}

// bsky/api for public api client usage
// unauthenticated
func NewAPIClient() *Client {
	cfg := NewConf()

	client := &xrpc.Client{
		Client: NewHTTPClient(),
		Host:   cfg.host(),
	}

	return &Client{
		atproto: client,
		conf:    cfg,
		log:     conf.NewLog(),
	}
}

// bsky/sync for server <-> server synchronization
// requires authentication
func NewSyncClient() (*Client, error) {
	client := NewAPIClient()
	return client.authenticated()
}

func (c *Client) authenticated() (*Client, error) {
	var err error
	if strings.TrimSpace(c.conf.identifier()) == "" || strings.TrimSpace(c.conf.password()) == "" {
		err := fmt.Errorf("missing identifier or password")
		c.log.WithErrorMsg(err, "Error authenticating bsky client")
		return nil, err
	}

	if c.session, err = atproto.ServerCreateSession(context.Background(), c.atproto, &atproto.ServerCreateSession_Input{
		Identifier: c.conf.identifier(),
		Password:   c.conf.password(),
	}); err != nil {
		c.log.WithErrorMsg(err, "Error authenticating atproto client", "host", c.conf.host(), "did", c.conf.identifier())
		return nil, err
	}

	// set auth context
	c.atproto.Auth = &xrpc.AuthInfo{
		AccessJwt:  c.session.AccessJwt,
		RefreshJwt: c.session.RefreshJwt,
		Handle:     c.session.Handle,
		Did:        c.session.Did,
	}

	return c, nil
}

func (c *Client) Profile() (*Profile, error) {
	if c.session == nil {
		err := fmt.Errorf("missing session")
		c.log.WithErrorMsg(err, "Missing atproto session")
		return nil, err
	}
	ctx := context.Background()
	var profiles *bsky.ActorGetProfiles_Output
	var err error
	if profiles, err = bsky.ActorGetProfiles(ctx, c.atproto, []string{
		c.session.Did,
	}); err != nil {
		c.log.WithErrorMsg(err, "Error fetching profile")
		return nil, err
	}
	return &Profile{
		profiles,
	}, nil
}
