package factory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store persists work items as one JSON file per item in a backlog directory.
type Store struct {
	Dir string
}

// NewStore returns a Store rooted at dir (normally <repo>/.factory/backlog).
func NewStore(dir string) Store { return Store{Dir: dir} }

func (s Store) path(id string) string {
	return filepath.Join(s.Dir, id+".json")
}

// Load reads one work item by ID.
func (s Store) Load(id string) (*WorkItem, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("work item %s not found in %s", id, s.Dir)
		}
		return nil, fmt.Errorf("read %s: %w", id, err)
	}
	var item WorkItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.path(id), err)
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	return &item, nil
}

// Save writes the item atomically (temp file + rename in the same directory).
func (s Store) Save(item *WorkItem) error {
	if err := item.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", s.Dir, err)
	}
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", item.ID, err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(s.Dir, ".tmp-"+item.ID+"-*.json")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", item.ID, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", item.ID, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync %s: %w", item.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", item.ID, err)
	}
	if err := os.Rename(tmpName, s.path(item.ID)); err != nil {
		return fmt.Errorf("rename %s: %w", item.ID, err)
	}
	return nil
}

// Exists reports whether an item file is present.
func (s Store) Exists(id string) bool {
	_, err := os.Stat(s.path(id))
	return err == nil
}

// List returns every valid item sorted by Order then ID. Files that fail to
// parse are reported as an error naming the offending file.
func (s Store) List() ([]*WorkItem, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backlog %s: %w", s.Dir, err)
	}
	var items []*WorkItem
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, ".tmp-") {
			continue
		}
		item, err := s.Load(strings.TrimSuffix(name, ".json"))
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	SortItems(items)
	return items, nil
}

// SortItems orders items by Order then ID.
func SortItems(items []*WorkItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		return items[i].ID < items[j].ID
	})
}

// Next returns the lowest-ordered item still in state queued.
func (s Store) Next() (*WorkItem, error) {
	items, err := s.List()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.State == StateQueued {
			return item, nil
		}
	}
	return nil, nil
}
