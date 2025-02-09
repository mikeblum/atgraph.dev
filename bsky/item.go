package bsky

import "github.com/bluesky-social/indigo/atproto/syntax"

const (
	ITEM_ACTOR_PROFILE      = syntax.NSID("app.bsky.actor.profile")
	ITEM_ACTOR_PROFILE_SELF = syntax.NSID("app.bsky.actor.profile/self")
	ITEM_FEED_LIKE          = syntax.NSID("app.bsky.feed.like")
	ITEM_FEED_POST          = syntax.NSID("app.bsky.feed.post")
	ITEM_FEED_REPOST        = syntax.NSID("app.bsky.feed.repost")
	ITEM_GRAPH_FOLLOW       = syntax.NSID("app.bsky.graph.follow")
	ITEM_GRAPH_BLOCK        = syntax.NSID("app.bsky.graph.block")
	ITEM_GRAPH_LIST         = syntax.NSID("app.bsky.graph.list")
	ITEM_GRAPH_LIST_BLOCK   = syntax.NSID("app.bsky.graph.listblock")
	ITEM_GRAPH_LIST_ITEM    = syntax.NSID("app.bsky.graph.listitem")
)
