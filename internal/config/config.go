package config

import (
	"fmt"
	"os"
)

type Config struct {
	JWTSecret            string
	ServerPort           string
	GoffEndpoint         string
	Environment          string
	KeycloakURL          string
	KeycloakRealm        string
	KeycloakClientID     string
	KeycloakClientSecret string
}

func Load() (*Config, error) {
	config := &Config{
		JWTSecret:            getEnv("JWT_SECRET", ""),
		ServerPort:           getEnv("SERVER_PORT", "1324"),
		GoffEndpoint:         getEnv("GOFF_ENDPOINT", "http://localhost:1031"),
		Environment:          getEnv("ENVIRONMENT", "development"),
		KeycloakURL:          getEnv("KEYCLOAK_URL", ""),
		KeycloakRealm:        getEnv("KEYCLOAK_REALM", ""),
		KeycloakClientID:     getEnv("KEYCLOAK_CLIENT_ID", ""),
		KeycloakClientSecret: getEnv("KEYCLOAK_CLIENT_SECRET", ""),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.ServerPort == "" {
		return fmt.Errorf("SERVER_PORT cannot be empty")
	}

	if c.GoffEndpoint == "" {
		return fmt.Errorf("GOFF_ENDPOINT cannot be empty")
	}

	if c.KeycloakURL == "" {
		return fmt.Errorf("KEYCLOAK_URL is required")
	}

	if c.KeycloakRealm == "" {
		return fmt.Errorf("KEYCLOAK_REALM is required")
	}

	if c.KeycloakClientID == "" {
		return fmt.Errorf("KEYCLOAK_CLIENT_ID is required")
	}

	if c.KeycloakClientSecret == "" {
		return fmt.Errorf("KEYCLOAK_CLIENT_SECRET is required")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
