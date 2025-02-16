package main

import (
	"os"

	"github.com/mikeblum/atproto-graph-viz/bsky"
	"github.com/mikeblum/atproto-graph-viz/conf"
)

func main() {
	log := conf.NewLog()
	firehose := bsky.NewFirehose()
	var err error

	if err = firehose.Stream(); err != nil {
		log.WithErrorMsg(err, "Error slurping from bsky firehose")
		exit()
	}
}

func exit() {
	os.Exit(1)
}
