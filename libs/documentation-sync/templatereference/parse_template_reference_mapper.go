package templatereference

import (
	"path"
	"regexp"
	"strings"
)

var (
	digits     = regexp.MustCompile(`^\d+$`)
	builtin    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	pageInURL  = regexp.MustCompile(`/pages/(\d+)(?:[/?#]|$)`)
	pageInForm = regexp.MustCompile(`[?&]pageId=(\d+)`)
)

// Parse reads a template value. An empty value is the default. A value that is all digits,
// `page:<id>` or a Confluence page address is a page; a lowercase word is a built-in's name;
// anything else is a workspace path. It reports ok false for a path that leaves the
// workspace.
func Parse(value string) (Reference, bool) {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return Reference{Kind: Builtin, Name: DefaultName}, true
	case digits.MatchString(value):
		return Reference{Kind: Page, PageID: value}, true
	case strings.HasPrefix(value, "page:") && digits.MatchString(strings.TrimSpace(value[len("page:"):])):
		return Reference{Kind: Page, PageID: strings.TrimSpace(value[len("page:"):])}, true
	case strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://"):
		if match := pageInURL.FindStringSubmatch(value); match != nil {
			return Reference{Kind: Page, PageID: match[1]}, true
		}
		if match := pageInForm.FindStringSubmatch(value); match != nil {
			return Reference{Kind: Page, PageID: match[1]}, true
		}

		return Reference{}, false
	case strings.HasPrefix(value, "./"):
		// "./brand" is the folder brand, which a bare "brand" would read as a built-in's name.
		value = value[2:]
	case builtin.MatchString(value):
		return Reference{Kind: Builtin, Name: value}, true
	}
	slashed := strings.ReplaceAll(value, "\\", "/")
	clean := path.Clean(slashed)
	if path.IsAbs(slashed) || (len(slashed) > 1 && slashed[1] == ':') || clean == ".." || strings.HasPrefix(clean, "../") || clean == "." {
		return Reference{}, false
	}

	return Reference{Kind: File, Path: clean}, true
}
