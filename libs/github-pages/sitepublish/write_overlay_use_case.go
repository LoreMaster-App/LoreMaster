package sitepublish

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"lore-master/libs/github-pages/siterender"
)

// OverlayMarkerFileName lists, one path per line, the files WriteOverlay wrote last time.
// It is how the next run knows which files are its own to replace or remove.
const OverlayMarkerFileName = ".lore-master-wiki"

// WriteOverlay lays the files over dir without owning the folder: it removes only the files
// the previous run listed in the marker, writes the new ones and rewrites the marker. Pages
// somebody else added are never touched, which is what a wiki needs, since its first page is
// created by hand on GitHub and people keep editing it there.
func WriteOverlay(dir string, files []siterender.SiteFile) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	markerPath := filepath.Join(dir, OverlayMarkerFileName)
	previous, err := os.ReadFile(markerPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, name := range strings.Split(string(previous), "\n") {
		if safe := safeRelativePath(name); safe != "" {
			if err := os.Remove(filepath.Join(dir, filepath.FromSlash(safe))); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	written := make([]string, 0, len(files))
	for _, file := range files {
		safe := safeRelativePath(file.Path)
		if safe == "" || safe == OverlayMarkerFileName {
			continue
		}
		target := filepath.Join(dir, filepath.FromSlash(safe))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, file.Content, 0o644); err != nil {
			return err
		}
		written = append(written, safe)
	}

	return os.WriteFile(markerPath, []byte(strings.Join(written, "\n")+"\n"), 0o644)
}

// safeRelativePath is name cleaned, or "" when it is empty, absolute or climbs out of the
// folder, so a hand-edited marker can never make a run delete outside it.
func safeRelativePath(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return ""
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") {
		return ""
	}

	return cleaned
}
