package syncjournal

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// FileState is a file as it was before a write-back.
type FileState struct {
	Existed bool
	Content []byte
}

// CaptureFiles reads the files a write-back is about to change (workspace-relative paths).
func CaptureFiles(workspaceRoot string, paths []string) map[string]FileState {
	states := make(map[string]FileState, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(path)))
		states[path] = FileState{Existed: err == nil, Content: content}
	}

	return states
}

// ChangedFiles compares the captured files with what is on disk now and returns the ones
// the write-back changed, each with the hash of what it left.
func ChangedFiles(workspaceRoot string, before map[string]FileState) []FileChange {
	var changes []FileChange
	for path, state := range before {
		content, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(path)))
		if err != nil {
			continue
		}
		if state.Existed && string(content) == string(state.Content) {
			continue
		}
		changes = append(changes, FileChange{Path: path, Existed: state.Existed, Before: state.Content, AfterHash: hashOf(content)})
	}
	slices.SortFunc(changes, func(a, b FileChange) int { return strings.Compare(a.Path, b.Path) })

	return changes
}

func hashOf(content []byte) string {
	sum := sha256.Sum256(content)

	return hex.EncodeToString(sum[:])
}
