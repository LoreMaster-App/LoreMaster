package markdowndocs

import (
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
)

var (
	markdownImage = regexp.MustCompile(`!\[[^\]]*\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	htmlImage     = regexp.MustCompile(`(?i)<img\b[^>]*?\bsrc\s*=\s*["']([^"']+)["']`)
	schemeOrRoot  = regexp.MustCompile(`^(?:[a-zA-Z][a-zA-Z0-9+.-]*:|/|#)`)
	fenceOpener   = regexp.MustCompile("^(```|~~~)")
)

// imageExtensions are the files copied along with the pages: what a page can show.
var imageExtensions = []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp"}

// relativeImages lists the images a Markdown page shows through relative paths, as paths
// relative to the page's own folder, in order of appearance without repeats. Images in fenced
// code are examples, not images; web addresses and absolute paths are not files of the project.
func relativeImages(markdown string) []string {
	var seen []string
	inFence := false
	for _, line := range strings.Split(markdown, "\n") {
		if fenceOpener.MatchString(strings.TrimSpace(line)) {
			inFence = !inFence

			continue
		}
		if inFence {
			continue
		}
		for _, pattern := range []*regexp.Regexp{markdownImage, htmlImage} {
			for _, match := range pattern.FindAllStringSubmatch(line, -1) {
				if reference, ok := cleanReference(match[1]); ok && !slices.Contains(seen, reference) {
					seen = append(seen, reference)
				}
			}
		}
	}

	return seen
}

// cleanReference drops the query and fragment, decodes the path and refuses what is not a
// relative path to an image file.
func cleanReference(raw string) (string, bool) {
	if schemeOrRoot.MatchString(raw) {
		return "", false
	}
	if cut := strings.IndexAny(raw, "?#"); cut >= 0 {
		raw = raw[:cut]
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil || decoded == "" {
		return "", false
	}
	if !slices.Contains(imageExtensions, strings.ToLower(path.Ext(decoded))) {
		return "", false
	}

	return decoded, true
}
