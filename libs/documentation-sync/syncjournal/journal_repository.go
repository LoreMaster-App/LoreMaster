package syncjournal

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// keep is how many journals a workspace retains; older ones are dropped on save.
const keep = 20

// Repository stores a workspace's journals as JSON files in one folder.
type Repository struct {
	dir string
}

// NewRepository keeps journals in dir.
func NewRepository(dir string) *Repository {
	return &Repository{dir: dir}
}

// DirFor is the folder holding a workspace's journals under cacheRoot (the user's cache
// folder): outside the repository, so a journal never shows in git.
func DirFor(cacheRoot string, workspaceRoot string) string {
	absolute, err := filepath.Abs(workspaceRoot)
	if err != nil {
		absolute = workspaceRoot
	}
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.ToSlash(absolute))))

	return filepath.Join(cacheRoot, "lore-master", "journals", hex.EncodeToString(sum[:])[:16])
}

// NewID makes an id that sorts by time.
func NewID(now time.Time) string {
	random := make([]byte, 3)
	_, _ = rand.Read(random)

	return now.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(random)
}

// Save writes the journal and drops the oldest beyond the retained few.
func (repository *Repository) Save(journal Journal) error {
	if journal.ID == "" {
		return errors.New("a journal needs an id")
	}
	if err := os.MkdirAll(repository.dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	if err := os.WriteFile(repository.path(journal.ID), data, 0o600); err != nil {
		return err
	}

	journals, err := repository.List()
	if err != nil {
		return nil
	}
	if len(journals) > keep {
		for _, old := range journals[keep:] {
			_ = repository.Remove(old.ID)
		}
	}

	return nil
}

// Load reads one journal.
func (repository *Repository) Load(id string) (Journal, error) {
	data, err := os.ReadFile(repository.path(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Journal{}, fmt.Errorf("there is no sync journal %q", id)
		}

		return Journal{}, err
	}
	var journal Journal
	if err := json.Unmarshal(data, &journal); err != nil {
		return Journal{}, fmt.Errorf("the sync journal %q is unreadable: %w", id, err)
	}

	return journal, nil
}

// List returns the journals, newest first. An unreadable one is skipped.
func (repository *Repository) List() ([]Journal, error) {
	entries, err := os.ReadDir(repository.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var journals []Journal
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		journal, err := repository.Load(strings.TrimSuffix(name, ".json"))
		if err == nil {
			journals = append(journals, journal)
		}
	}
	sort.Slice(journals, func(i, j int) bool { return journals[i].ID > journals[j].ID })

	return journals, nil
}

// Remove deletes a journal; one that is already gone is not an error.
func (repository *Repository) Remove(id string) error {
	err := os.Remove(repository.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

func (repository *Repository) path(id string) string {
	return filepath.Join(repository.dir, filepath.Base(id)+".json")
}
