package main

import (
	"os"

	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/mikeblum/atproto-graph-viz/conf"
)

func main() {
	log := conf.NewLog()
	var client *bsky.Client
	var err error
	if client, err = bsky.New(); err != nil {
		log.WithError(err, "Error creating bsky client")
		exit()
	}
	var profile *bsky.Profile
	if profile, err = client.Profile(); err != nil {
		log.WithError(err, "Error fetching profile")
		exit()
	}
	log.With("profile", profile).Info("Fetched profiles")
}

func exit() {
	os.Exit(1)
}
