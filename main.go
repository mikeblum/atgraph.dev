package main

import (
	"os"

	"github.com/mikeblum/atgraph.dev/bsky"
	"github.com/mikeblum/atgraph.dev/conf"
)

func main() {
	log := conf.NewLog()
	var client *bsky.Client
	var err error
	if client, err = bsky.NewSyncClient(); err != nil {
		log.WithErrorMsg(err, "Error creating bsky client")
		exit()
	}
	var profile *bsky.Profile
	if profile, err = client.Profile(); err != nil {
		log.WithErrorMsg(err, "Error fetching bsky profile")
		exit()
	}
	log.With("profile", profile).Info("Fetched bsky profiles")
}

func exit() {
	os.Exit(1)
}
