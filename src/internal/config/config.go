package config

import (
	"fmt"
	"os"
)

type Config struct {
	JWTSecret    string
	ServerPort   string
	GoffEndpoint string
	Environment  string
}

func Load() (*Config, error) {
	config := &Config{
		JWTSecret:    getEnv("JWT_SECRET", ""),
		ServerPort:   getEnv("SERVER_PORT", "1324"),
		GoffEndpoint: getEnv("GOFF_ENDPOINT", "http://localhost:1031"),
		Environment:  getEnv("ENVIRONMENT", "development"),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if c.ServerPort == "" {
		return fmt.Errorf("SERVER_PORT cannot be empty")
	}

	if c.GoffEndpoint == "" {
		return fmt.Errorf("GOFF_ENDPOINT cannot be empty")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
