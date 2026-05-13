package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
