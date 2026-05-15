package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Server    string `yaml:"server"`
	Token     string `yaml:"token"`
	ExpiresAt string `yaml:"expires_at"`
}

func NewConfig(server, token string, expiresAt string) *Config {
	return &Config{
		Server:    server,
		Token:     token,
		ExpiresAt: expiresAt,
	}
}

// DefaultConfigPath returns ~/.gcctl/config.yaml using os.UserHomeDir.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	if home == "" {
		return "", errors.New("HOME is empty; cannot resolve config path")
	}
	return filepath.Join(home, ".gcctl", "config.yaml"), nil
}

// IsLoggedIn checks that server and token are set and that the token is not expired.
func (c *Config) IsLoggedIn() bool {
	if c.Server == "" || c.Token == "" {
		return false
	}
	if c.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, c.ExpiresAt)
		if err != nil {
			return false
		}
		if time.Now().After(expiresAt) {
			return false
		}
	}
	return true
}
