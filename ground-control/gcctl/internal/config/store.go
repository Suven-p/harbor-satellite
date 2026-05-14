package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Store loads and saves Config to durable storage.
type Store interface {
	Load() (*Config, error)
	Save(*Config) error
	Path() string
}

// FileStore reads and writes Config from a single YAML file.
type FileStore struct {
	path string
}

// NewFileStore returns a Store backed by the given file path.
func NewFileStore(path string) (*FileStore, error) {
	// Create the file if it does not exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create config dir: %w", err)
		}
		if f, err := os.Create(path); err != nil {
			return nil, fmt.Errorf("create config file: %w", err)
		} else {
			f.Close()
		}
	}
	return &FileStore{path: path}, nil
}

// Path returns the resolved file path.
func (s *FileStore) Path() string { return s.path }

// Save writes c to s.path with mode 0600, creating the parent dir 0700 if needed.
// The write is atomic via a sibling temp file + rename.
func (s *FileStore) Save(c *Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, filepath.Base(s.path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // best-effort cleanup if rename fails

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("rename temp config: %w", err)
	}
	return nil
}

// Load reads s.path and returns its parsed Config.
func (s *FileStore) Load(c *Config) error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}
