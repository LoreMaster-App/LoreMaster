package workspacesettings

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
)

// Validate reports every problem in settings at once, each naming where it is and what
// to write instead. Values the format reserves for features not built yet are refused
// with the issue that tracks them, so a file written for a later version fails loudly
// instead of being half-understood.
func Validate(settings Settings) error {
	var problems []error
	add := func(format string, args ...any) { problems = append(problems, fmt.Errorf(format, args...)) }

	if settings.Version != CurrentVersion {
		add("version %d is not supported; this version of Lore Master reads version %d", settings.Version, CurrentVersion)
	}
	if len(settings.Outputs) == 0 {
		add("outputs is empty; add at least one output")
	}
	seen := map[string]int{}
	for i, output := range settings.Outputs {
		where := fmt.Sprintf("outputs[%d]", i)
		choose(add, where+".platform", output.Platform, []string{"confluence"}, nil)
		choose(add, where+".direction", output.Direction, []string{"to-platform", "two-way"}, nil)
		choose(add, where+".mermaidMode", output.MermaidMode, []string{"image", "code"}, map[string]string{"html-macro": "#40", "marketplace-macro": "#40"})
		choose(add, where+".titleCollision", output.TitleCollision, []string{"fail", "adopt"}, nil)
		choose(add, where+".linkMode", output.LinkMode, []string{"title", "id"}, nil)
		if output.BaseURL != "" && !strings.HasPrefix(output.BaseURL, "https://") && !strings.HasPrefix(output.BaseURL, "http://localhost") && !strings.HasPrefix(output.BaseURL, "http://127.0.0.1") {
			add("%s.baseUrl %q must be an https address", where, output.BaseURL)
		}
		if output.TitlePrefix != strings.TrimSpace(output.TitlePrefix) {
			add("%s.titlePrefix %q has leading or trailing spaces", where, output.TitlePrefix)
		}
		if output.Space != "" && output.ParentPageID != "" {
			key := output.Platform + " " + output.BaseURL + " " + output.Space + " " + output.ParentPageID
			if first, duplicate := seen[key]; duplicate {
				add("%s syncs to the same space and parent page as outputs[%d]; two outputs would overwrite each other's pages", where, first)
			}
			seen[key] = i
		}
		if len(output.Content) == 0 {
			add("%s.content is empty; add at least one content entry", where)
		}
		for j, content := range output.Content {
			at := fmt.Sprintf("%s.content[%d]", where, j)
			choose(add, at+".type", content.Type, []string{"markdown"}, map[string]string{"test-results": "#96", "code-docs": "#96"})
			if content.Template != "" {
				// An empty template means "unspecified", which is the default; only a non-empty,
				// non-default value is a (reserved) custom template.
				choose(add, at+".template", content.Template, []string{"default"}, nil)
			}
			for _, root := range content.Roots {
				clean := path.Clean(root)
				if root == "" || path.IsAbs(root) || clean == ".." || strings.HasPrefix(clean, "../") {
					add("%s.roots entry %q must be a path inside the workspace", at, root)
				}
			}
		}
	}

	return errors.Join(problems...)
}

// choose checks an enumerated value. A value in reserved is valid in the format but
// not built yet; templates other than default are all reserved (#97).
func choose(add func(string, ...any), field string, value string, allowed []string, reserved map[string]string) {
	switch {
	case slices.Contains(allowed, value):
		return
	case reserved[value] != "":
		add("%s %q is planned but not available yet (%s); use %s", field, value, reserved[value], strings.Join(allowed, " or "))
	case strings.HasSuffix(field, ".template"):
		add("%s %q: custom templates are planned but not available yet (#97); use default", field, value)
	default:
		add("%s %q is not one of %s", field, value, strings.Join(allowed, ", "))
	}
}
