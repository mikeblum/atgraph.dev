package bsky

const (
	ENV_BSKY_IDENTIFIER      = "BSKY_IDENTIFIER"
	ENV_BSKY_MAX_RETRY_COUNT = "BSKY_MAX_RETRY_COUNT"
	ENV_BSKY_PASSWORD        = "BSKY_PASSWORD"
	ENV_BSKY_PDS_URL         = "BSKY_PDS_URL"
	ENV_BSKY_PAGE_SIZE       = "BSKY_PAGE_SIZE"
	ENV_BSKY_WORKER_COUNT    = "BSKY_WORKER_COUNT"

	// defaults
	// https://docs.bsky.app/docs/advanced-guides/api-directory#bluesky-services
	// App View URL
	BSKY_APP_VIEW_URL = "https://api.bsky.app"
	// If you were making an authenticated client, you would
	// use the PDS URL here instead - the main one is bsky.social
	BSKY_ENTRYWAY_URL    = "https://bsky.social"
	BSKY_RELAY_URL       = "https://bsky.network"
	DEFAULT_PAGE_SIZE    = 1000
	DEFAULT_WORKER_COUNT = 5
	DEFAULT_MAX_RETRIES  = 3
	ITEMS_BUFFER         = 100
)
