package syncstate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Read loads lore-master.json from workspaceRoot. A missing file is an empty state, not an
// error. One that cannot be trusted (not JSON, or written by a newer LoreMaster) is: writing
// over it would lose what it knows.
func Read(workspaceRoot string) (File, error) {
	content, err := os.ReadFile(filepath.Join(workspaceRoot, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return Empty(), nil
	}
	if err != nil {
		return File{}, fmt.Errorf("%s: %w", FileName, err)
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return Empty(), nil
	}
	var file File
	if err := json.Unmarshal(content, &file); err != nil {
		return File{}, fmt.Errorf("%s is not valid JSON (%v); fix or delete it, the next sync adopts the pages by title", FileName, err)
	}
	if file.Version > CurrentVersion {
		return File{}, fmt.Errorf("%s was written by a newer LoreMaster (layout %d, this one reads %d); update the extension", FileName, file.Version, CurrentVersion)
	}
	if file.Storages == nil {
		file.Storages = map[string]Storage{}
	}
	file.Version = CurrentVersion

	return file, nil
}

// Write saves the state and reports whether the file changed. An unchanged state touches
// nothing, and an empty state is never created, so a workspace that syncs nothing gets no file.
func Write(workspaceRoot string, file File) (bool, error) {
	path := filepath.Join(workspaceRoot, FileName)
	existing, readErr := os.ReadFile(path)
	if len(file.Storages) == 0 && readErr != nil {
		return false, nil
	}
	file.Version = CurrentVersion
	encoded, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return false, err
	}
	encoded = append(encoded, '\n')
	if readErr == nil && bytes.Equal(normaliseEndings(existing), encoded) {
		return false, nil
	}
	temporary, err := os.CreateTemp(workspaceRoot, "."+FileName+".*")
	if err != nil {
		return false, err
	}
	defer func() { _ = os.Remove(temporary.Name()) }()
	if _, err := temporary.Write(encoded); err != nil {
		_ = temporary.Close()

		return false, err
	}
	if err := temporary.Close(); err != nil {
		return false, err
	}
	if err := os.Chmod(temporary.Name(), 0o644); err != nil {
		return false, err
	}

	return true, os.Rename(temporary.Name(), path)
}

// normaliseEndings lets a file that git checked out with CRLF compare equal to what is written.
func normaliseEndings(content []byte) []byte {
	return bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
}
