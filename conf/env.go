package conf

import (
	"os"

	"github.com/joho/godotenv"
)

type EnvConf interface {
	// Loads env variables from .env files and/or OS environment
	Load() error
	// Resolve env variables or fallback
	GetEnv(env, fallback string) string
}

func NewEnvConf(files ...string) EnvConf {
	return &envConf{
		files: files,
	}
}

type envConf struct {
	loaded bool
	files  []string
}

func (e *envConf) Load() error {
	return godotenv.Load()
}

func (e *envConf) GetEnv(env, fallback string) string {
	if !e.loaded {
		e.Load()
		e.loaded = true
	}
	if value, ok := os.LookupEnv(env); ok {
		return value
	}
	return fallback
}
