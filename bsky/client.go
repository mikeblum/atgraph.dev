package bsky

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/mikeblum/atproto-graph-viz/conf"
)

type Client struct {
	atproto *xrpc.Client
	conf    *Conf
	session *atproto.ServerCreateSession_Output
	log     *conf.Log
	cursor  *string
}

func NewClient() (*Client, error) {
	var err error
	log := conf.NewLog()

	cfg := NewConf()

	client := &xrpc.Client{
		Host: cfg.host(),
	}

	if strings.TrimSpace(cfg.identifier()) == "" || strings.TrimSpace(cfg.password()) == "" {
		err := fmt.Errorf("missing identifier or password")
		log.WithErrorMsg(err, "Error authenticating bsky client")
		return nil, err
	}

	var session *atproto.ServerCreateSession_Output
	if session, err = atproto.ServerCreateSession(context.Background(), client, &atproto.ServerCreateSession_Input{
		Identifier: cfg.identifier(),
		Password:   cfg.password(),
	}); err != nil {
		log.WithErrorMsg(err, "Error authenticating atproto client", "host", cfg.host(), "did", cfg.identifier())
		return nil, err
	}

	// set auth context
	client.Auth = &xrpc.AuthInfo{
		AccessJwt:  session.AccessJwt,
		RefreshJwt: session.RefreshJwt,
		Handle:     session.Handle,
		Did:        session.Did,
	}

	return &Client{
		atproto: client,
		conf:    NewConf(),
		session: session,
		log:     log,
	}, nil
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
