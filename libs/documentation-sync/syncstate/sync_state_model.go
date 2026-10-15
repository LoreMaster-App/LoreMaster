package syncstate

import "slices"

// FileName is the state file, at the workspace root.
const FileName = "lore-master.json"

// CurrentVersion is the layout this build reads and writes.
const CurrentVersion = 1

// File is the whole state file: one Storage per configured output, by its name.
type File struct {
	Version  int                `json:"version"`
	Storages map[string]Storage `json:"storages"`
}

// Storage is what a sync recorded for one output: where it goes and, per workspace-relative
// Markdown file, the page it became.
type Storage struct {
	Platform string          `json:"platform,omitempty"`
	BaseURL  string          `json:"baseUrl,omitempty"`
	Space    string          `json:"space,omitempty"`
	Pages    map[string]Page `json:"pages"`
}

// Page is one file's page.
type Page struct {
	PageID      string `json:"pageId"`
	ParentID    string `json:"parentId,omitempty"`
	Version     int    `json:"version,omitempty"`
	ContentHash string `json:"contentHash,omitempty"`
	// RenderHash is the hash of the page body as it was converted.
	RenderHash string `json:"renderHash,omitempty"`
	// Attachments maps an attachment file name to the hash it was uploaded with.
	Attachments map[string]string `json:"attachments,omitempty"`
	// SyncedAt is RFC 3339, UTC.
	SyncedAt string `json:"syncedAt,omitempty"`
}

// Empty is a file with no storages.
func Empty() File {
	return File{Version: CurrentVersion, Storages: map[string]Storage{}}
}

// StorageNamed returns the storage recorded under key, or an empty one.
func (file File) StorageNamed(key string) Storage {
	storage, ok := file.Storages[key]
	if !ok || storage.Pages == nil {
		storage.Pages = map[string]Page{}
	}

	return storage
}

// Keys are the storage names in a stable order.
func (file File) Keys() []string {
	keys := make([]string, 0, len(file.Storages))
	for key := range file.Storages {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	return keys
}
