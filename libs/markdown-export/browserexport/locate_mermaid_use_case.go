package browserexport

import (
	"os"
	"strings"
)

// MermaidVariable names the environment variable that points at the Mermaid library (mermaid.min.js).
const MermaidVariable = "LORE_MASTER_MERMAID"

// LocateMermaid returns the Mermaid library to draw diagrams with: the configured file, else the one the
// LORE_MASTER_MERMAID variable names, and an empty string when there is none or it is not a file. The editor extension
// ships the library; a command line without it leaves diagrams as code, and says nothing, because that is how every
// other format shows them.
func LocateMermaid(configured string) string {
	for _, candidate := range []string{strings.TrimSpace(configured), strings.TrimSpace(os.Getenv(MermaidVariable))} {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return ""
}
