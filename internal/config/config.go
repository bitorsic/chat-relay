package config

import (
	"fmt"
	"os"
)

type Config struct {
	BackendPort string
}

func Load() (*Config, error) {
	backendPort := os.Getenv("BACKEND_PORT")

	if backendPort == "" {
		return nil, fmt.Errorf(".env not complete")
	}

	return &Config{
		BackendPort: backendPort,
	}, nil
}
