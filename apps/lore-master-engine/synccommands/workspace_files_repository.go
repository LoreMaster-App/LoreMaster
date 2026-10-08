package synccommands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lore-master/libs/documentation-sync/syncexecution"
	"lore-master/libs/markdown-workspace/documentdiscovery"
)

// fileReader reads workspace files by their workspace-relative path, and never outside
// the workspace, whatever a path says.
func fileReader(root string) syncexecution.FileReader {
	return func(path documentdiscovery.DocumentPath) ([]byte, error) {
		full := filepath.Join(root, filepath.FromSlash(string(path)))
		relative, err := filepath.Rel(root, full)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%s is outside the workspace", path)
		}

		return os.ReadFile(full)
	}
}
