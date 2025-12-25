package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultTTL is the default cache time-to-live (24 hours)
	DefaultTTL = 24 * time.Hour

	// CacheDir is the cache directory name
	CacheDir = "agent-linear-cli"
)

// Entry represents a cached item with timestamp
type Entry[T any] struct {
	Data      T         `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// Manager handles cache operations
type Manager struct {
	dir string
	ttl time.Duration
}

// NewManager creates a new cache manager
func NewManager() (*Manager, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, err
	}

	return &Manager{
		dir: cacheDir,
		ttl: DefaultTTL,
	}, nil
}

// getCacheDir returns the cache directory path
func getCacheDir() (string, error) {
	// Use XDG_CACHE_HOME if set, otherwise ~/.cache
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheHome = filepath.Join(home, ".cache")
	}

	return filepath.Join(cacheHome, CacheDir), nil
}

// ensureDir creates the cache directory if it doesn't exist
func (m *Manager) ensureDir() error {
	return os.MkdirAll(m.dir, 0755)
}

// keyPath returns the file path for a cache key
func (m *Manager) keyPath(key string) string {
	return filepath.Join(m.dir, key+".json")
}

// Read retrieves a cached item, returns nil if not found or expired
func Read[T any](m *Manager, key string) (*T, error) {
	path := m.keyPath(key)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss, not an error
		}
		return nil, err
	}

	var entry Entry[T]
	if err := json.Unmarshal(data, &entry); err != nil {
		// Invalid cache file, treat as miss
		return nil, nil
	}

	// Check if expired
	if time.Since(entry.Timestamp) > m.ttl {
		// Expired, clean up
		os.Remove(path)
		return nil, nil
	}

	return &entry.Data, nil
}

// Write stores an item in the cache
func Write[T any](m *Manager, key string, data T) error {
	if err := m.ensureDir(); err != nil {
		return err
	}

	entry := Entry[T]{
		Data:      data,
		Timestamp: time.Now(),
	}

	bytes, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.keyPath(key), bytes, 0644)
}

// Clear removes a specific cache entry
func (m *Manager) Clear(key string) error {
	err := os.Remove(m.keyPath(key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ClearAll removes all cache entries
func (m *Manager) ClearAll() error {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			os.Remove(filepath.Join(m.dir, entry.Name()))
		}
	}

	return nil
}

// Has checks if a cache key exists and is not expired
func Has[T any](m *Manager, key string) bool {
	data, _ := Read[T](m, key)
	return data != nil
}

// GetOrFetch retrieves from cache or calls fetch function if not cached
func GetOrFetch[T any](m *Manager, key string, fetch func() (T, error)) (T, error) {
	// Try cache first
	cached, err := Read[T](m, key)
	if err != nil {
		var zero T
		return zero, err
	}
	if cached != nil {
		return *cached, nil
	}

	// Fetch fresh data
	data, err := fetch()
	if err != nil {
		return data, err
	}

	// Store in cache (ignore write errors)
	Write(m, key, data)

	return data, nil
}

// Key helpers for consistent cache key naming

// TeamKey returns the cache key for team-scoped data
func TeamKey(resource, teamID string) string {
	return resource + "-team-" + teamID
}

// WorkspaceKey returns the cache key for workspace-scoped data
func WorkspaceKey(resource string) string {
	return resource + "-workspace"
}
