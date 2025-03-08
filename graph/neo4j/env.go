package neo4j

import "time"

const (
	ENV_NEO4J_DATABASE = "NEO4J_DATABASE"
	ENV_NEO4J_PASSWORD = "NEO4J_PASSWORD"
	ENV_NEO4J_TIMEOUT  = "NEO4J_TIMEOUT"
	ENV_NEO4J_URI      = "NEO4J_URI"
	ENV_NEO4J_USERNAME = "NEO4J_USERNAME"

	// defaults
	NEO4J_CONNECTION_POOL_SIZE = 3
	NEO4J_DATABASE             = "bluesky"
	NEO4J_TIMEOUT              = time.Second * 10
	NEO4J_URI                  = "neo4j://localhost:7687"
	NEO4J_USERNAME             = "neo4j"
)
