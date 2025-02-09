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
	if err = client.ListRepos(); err != nil {
		log.WithError(err, "Error fetching bsky repos")
		exit()
	}
}

func exit() {
	os.Exit(1)
}
