package bsky

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/joho/godotenv"
	"github.com/mikeblum/atproto-graph-viz/conf"
)

type Client struct {
	client  *xrpc.Client
	session *atproto.ServerCreateSession_Output
	log     *conf.Log

	ingest func(ctx context.Context, stream <-chan *Item) error
}

func New() (*Client, error) {
	var err error
	log := conf.NewLog()
	if err = godotenv.Load(); err != nil {
		log.WithError(err, "Error loading .env file")
	}
	host := conf.GetEnv(ENV_BSKY_PDS_URL, BSKY_SOCIAL_URL)
	client := &xrpc.Client{
		Host: host,
	}

	identifier := conf.GetEnv(ENV_BSKY_IDENTIFIER, "")
	password := conf.GetEnv(ENV_BSKY_PASSWORD, "")
	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(password) == "" {
		err := fmt.Errorf("missing identifier or password")
		log.WithError(err, "Error authenticating bsky client")
		return nil, err
	}

	var session *atproto.ServerCreateSession_Output
	if session, err = atproto.ServerCreateSession(context.Background(), client, &atproto.ServerCreateSession_Input{
		Identifier: identifier,
		Password:   password,
	}); err != nil {
		log.WithError(err, "Error authenticating atproto client", "host", host, "did", identifier)
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
		client:  client,
		session: session,
		log:     log,
	}, nil
}

func (c *Client) Ingest(ingest func(ctx context.Context, stream <-chan *Item) error) *Client {
	c.ingest = ingest
	return c
}

func (c *Client) Profile() (*Profile, error) {
	if c.session == nil {
		err := fmt.Errorf("missing session")
		c.log.WithError(err, "Missing atproto session")
		return nil, err
	}
	ctx := context.Background()
	var profiles *bsky.ActorGetProfiles_Output
	var err error
	if profiles, err = bsky.ActorGetProfiles(ctx, c.client, []string{
		c.session.Did,
	}); err != nil {
		c.log.WithError(err, "Error fetching profile")
		return nil, err
	}
	return &Profile{
		profiles,
	}, nil
}
