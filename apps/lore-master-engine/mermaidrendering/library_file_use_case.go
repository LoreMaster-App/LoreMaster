package mermaidrendering

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// LibraryPath is the Mermaid library (mermaid.min.js) a pdf, png or html export draws diagrams with: the one asked for, else the
// one the LORE_MASTER_MERMAID variable names, else the copy embedded in the engine, written once to the temporary folder. The
// result is empty only when that copy cannot be written, in which case diagrams stay code.
func LibraryPath(asked string) string {
	if asked != "" {
		return asked
	}
	// The variable also takes the word "editor" for a sync, so it only counts when it names a file.
	if named := os.Getenv("LORE_MASTER_MERMAID"); named != "" {
		if info, err := os.Stat(named); err == nil && info.Mode().IsRegular() {
			return named
		}
	}
	text, err := bundle()
	if err != nil {
		return ""
	}
	digest := sha256.Sum256([]byte(text))
	path := filepath.Join(os.TempDir(), "lore-master-mermaid-"+hex.EncodeToString(digest[:6])+".js")
	if info, statErr := os.Stat(path); statErr == nil && info.Size() == int64(len(text)) {
		return path
	}
	if os.WriteFile(path, []byte(text), 0o600) != nil {
		return ""
	}

	return path
}
