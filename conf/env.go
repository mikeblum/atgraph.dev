package conf

import (
	"os"
)

const (
	ENV_LOG_LEVEL = "LOG_LEVEL"

	// defaults
	LOG_LEVEL_INFO = "info"
)

func GetEnv(env, fallback string) string {
	if value, ok := os.LookupEnv(env); ok {
		return value
	}
	return fallback
}
