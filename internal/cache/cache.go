package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/subme-app/subme/internal/collector"
)

type Entry struct {
	YAML      []byte            `json:"yaml"`
	LastFetch time.Time         `json:"last_fetch"`
	UserInfo  *collector.UserInfo `json:"user_info,omitempty"`
}

type Cache struct {
	dir string
}

func New(cacheDir string) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &Cache{dir: cacheDir}, nil
}

func (c *Cache) Get(key string) (*Entry, error) {
	path := filepath.Join(c.dir, safeKey(key)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cache: %w", err)
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("parse cache: %w", err)
	}
	return &e, nil
}

func (c *Cache) Set(key string, data []byte, userInfo *collector.UserInfo) error {
	e := Entry{
		YAML:      data,
		LastFetch: time.Now(),
		UserInfo:  userInfo,
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	path := filepath.Join(c.dir, safeKey(key)+".json")
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	return nil
}

func (c *Cache) List() ([]string, error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return nil, fmt.Errorf("list cache: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}
	return names, nil
}

func (c *Cache) Delete(key string) {
	path := filepath.Join(c.dir, safeKey(key)+".json")
	os.Remove(path)
}

func safeKey(key string) string {
	return strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"..", "_",
	).Replace(key)
}
