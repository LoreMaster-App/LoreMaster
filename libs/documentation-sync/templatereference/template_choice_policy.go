package templatereference

import (
	"fmt"
	"slices"
	"strings"
)

// builtins are the templates LoreMaster ships, per platform.
var builtins = map[string][]string{
	"github-pages": {DefaultName, "minimal", "handbook", "headless"},
	"github-wiki":  {DefaultName},
	"confluence":   {DefaultName},
}

// BuiltinNames are the built-in templates a platform offers, the default first.
func BuiltinNames(platform string) []string {
	return slices.Clone(builtins[platform])
}

// Check parses value for platform and says why it cannot be used, or "" when it can.
func Check(platform string, value string) (Reference, string) {
	reference, ok := Parse(value)
	if !ok {
		return Reference{}, fmt.Sprintf("%q is not a built-in name, a page id or address, or a path inside the workspace", value)
	}
	switch reference.Kind {
	case Builtin:
		if !slices.Contains(builtins[platform], reference.Name) {
			return reference, fmt.Sprintf("%q is not a built-in template of %s; use %s, or a path", reference.Name, platform, strings.Join(builtins[platform], ", "))
		}
	case Page:
		if platform != "confluence" {
			return reference, "a Confluence page can only be the template of a confluence output"
		}
	case File:
		if platform == "github-wiki" && !strings.HasSuffix(strings.ToLower(reference.Path), ".md") {
			return reference, fmt.Sprintf("the wiki template %q must be a Markdown file (.md)", reference.Path)
		}
	}

	return reference, ""
}
