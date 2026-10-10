package workspacesettings

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
)

// reservedPlatforms are the platforms the format names but has no adapter for yet; each
// points at the issue that tracks it (the epic #166).
var reservedPlatforms = map[string]string{
	"sharepoint": "#166", "google-sites": "#166", "notion": "#166",
	"microsoft-loop": "#166", "monday": "#166", "clickup": "#166",
	"zoho-wiki": "#166", "basecamp": "#166", "asana": "#166",
}

// Validate reports every problem in settings at once, each naming where it is and what
// to write instead. Values the format reserves for features not built yet are refused
// with the issue that tracks them, so a file written for a later version fails loudly
// instead of being half-understood.
func Validate(settings Settings) error {
	var problems []error
	add := func(format string, args ...any) { problems = append(problems, fmt.Errorf(format, args...)) }

	if settings.Version != CurrentVersion {
		add("version %d is not supported; this version of LoreMaster reads version %d", settings.Version, CurrentVersion)
	}
	if len(settings.Outputs) == 0 {
		add("outputs is empty; add at least one output")
	}
	validateIgnore(add, settings.Ignore)
	validateGenerators(add, settings.Generators)
	confluenceSeen := map[string]int{}
	pagesSeen := map[string]int{}
	for i, output := range settings.Outputs {
		where := fmt.Sprintf("outputs[%d]", i)
		choose(add, where+".platform", output.Platform, []string{"confluence", "github-pages", "github-wiki"}, reservedPlatforms)
		switch output.Platform {
		case "confluence":
			validateConfluenceOutput(add, where, output, confluenceSeen, i)
		case "github-pages", "github-wiki":
			validateGitHubPagesOutput(add, where, output, pagesSeen, i)
		}
		validateContent(add, where, output)
	}

	return errors.Join(problems...)
}

// validateConfluenceOutput checks the fields a Confluence output uses, and that no two
// outputs would overwrite each other's pages.
func validateConfluenceOutput(add func(string, ...any), where string, output Output, seen map[string]int, index int) {
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
		seen[key] = index
	}
}

// validateGitHubPagesOutput checks a github-pages or github-wiki output: it is one-way, it uses a repo
// and branch rather than Confluence's targeting fields, and no two outputs publish to the
// same place.
func validateGitHubPagesOutput(add func(string, ...any), where string, output Output, seen map[string]int, index int) {
	choose(add, where+".direction", output.Direction, []string{"to-platform"}, nil)

	for _, field := range []struct{ name, value string }{
		{"space", output.Space}, {"parentPageId", output.ParentPageID},
		{"titlePrefix", output.TitlePrefix}, {"baseUrl", output.BaseURL},
	} {
		if field.value != "" {
			add("%s.%s is not used by a %s output; remove it", where, field.name, output.Platform)
		}
	}
	if output.Branch != "" && (strings.ContainsAny(output.Branch, " \t") || strings.HasPrefix(output.Branch, "/") || strings.HasSuffix(output.Branch, "/")) {
		add("%s.branch %q is not a valid branch name", where, output.Branch)
	}
	if strings.ContainsAny(output.Repo, " \t") {
		add("%s.repo %q must not contain spaces", where, output.Repo)
	}

	if output.Platform == "github-wiki" && output.Path != "" {
		add("%s.path is not used by a github-wiki output; a wiki is a flat folder of pages", where)
	} else if reason := invalidSitePath(output.Path); reason != "" {
		add("%s.path %q %s", where, output.Path, reason)
	}

	branch := output.Branch
	if branch == "" {
		branch = "gh-pages"
	}
	key := output.Platform + "\x00" + output.Repo + "\x00" + branch + "\x00" + strings.Trim(output.Path, "/")
	if first, duplicate := seen[key]; duplicate {
		add("%s publishes to the same repo and branch as outputs[%d]; two outputs would overwrite each other", where, first)
	}
	seen[key] = index
}

// invalidSitePath says why a github-pages path cannot be used, or "" when it can: it names a
// folder inside the branch, so it is relative and never climbs out.
func invalidSitePath(path string) string {
	switch {
	case path == "":
		return ""
	case strings.ContainsAny(path, " \t\\"):
		return "must not contain spaces or backslashes; use forward slashes"
	case strings.HasPrefix(path, "/"):
		return "must be relative to the branch root, without a leading slash"
	}
	for _, part := range strings.Split(strings.TrimSuffix(path, "/"), "/") {
		if part == ".." || part == "." || part == "" || part == ".git" {
			return "must be a plain folder path inside the branch, without \"..\", \".\", \".git\" or empty segments"
		}
	}

	return ""
}

// validateContent checks the content sources an output feeds from, which every platform
// shares.
func validateContent(add func(string, ...any), where string, output Output) {
	validateSelection(add, where, output)
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

// validateSelection checks an output's include and exclude lists: entries stay inside the
// workspace, and the same entry is not both included and excluded, which would say nothing.
func validateSelection(add func(string, ...any), where string, output Output) {
	for _, list := range []struct {
		name    string
		entries []string
	}{{"include", output.Include}, {"exclude", output.Exclude}} {
		for i, entry := range list.entries {
			clean := path.Clean(strings.TrimPrefix(entry, "!"))
			if strings.TrimSpace(entry) == "" || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
				add("%s.%s[%d] %q must be a non-empty path or pattern inside the workspace", where, list.name, i, entry)
			}
		}
	}
	for _, entry := range output.Include {
		if slices.Contains(output.Exclude, entry) {
			add("%s lists %q in both include and exclude; keep it in one", where, entry)
		}
	}
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

// validateIgnore checks the top-level ignore list: patterns stay inside the workspace.
func validateIgnore(add func(string, ...any), patterns []string) {
	for i, pattern := range patterns {
		clean := path.Clean(strings.TrimPrefix(pattern, "!"))
		if strings.TrimSpace(pattern) == "" || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
			add("ignore[%d] %q must be a non-empty pattern inside the workspace", i, pattern)
		}
	}
}

// BuiltGeneratorTypes are the generator types that exist; the rest the format names are
// reserved until they are built.
var BuiltGeneratorTypes = []string{"test-results", "go-docs", "openapi-docs", "python-docs", "csharp-docs", "dart-docs", "ts-docs"}

var reservedGenerators = map[string]string{}

// validateGenerators checks the generators list: a known type, an output folder inside the
// workspace that no other generator writes into, and input patterns that stay inside it.
func validateGenerators(add func(string, ...any), generators []Generator) {
	outputs := make([]string, len(generators))
	for i, generator := range generators {
		at := fmt.Sprintf("generators[%d]", i)
		choose(add, at+".type", generator.Type, BuiltGeneratorTypes, reservedGenerators)

		output := path.Clean(generator.Output)
		switch {
		case generator.Output == "":
			add("%s.output is empty; name the folder the pages are written to", at)
		case path.IsAbs(generator.Output) || output == "." || output == ".." || strings.HasPrefix(output, "../"):
			add("%s.output %q must be a folder inside the workspace", at, generator.Output)
		default:
			outputs[i] = output
		}
		for j, pattern := range generator.Input {
			clean := path.Clean(strings.TrimPrefix(pattern, "!"))
			if strings.TrimSpace(pattern) == "" || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
				add("%s.input[%d] %q must be a non-empty pattern inside the workspace", at, j, pattern)
			}
		}
	}
	for i := range generators {
		for j := i + 1; j < len(generators); j++ {
			if outputs[i] != "" && outputs[j] != "" && (outputs[i] == outputs[j] || strings.HasPrefix(outputs[j], outputs[i]+"/") || strings.HasPrefix(outputs[i], outputs[j]+"/")) {
				add("generators[%d] and generators[%d] write into overlapping folders (%s, %s); give each its own", i, j, outputs[i], outputs[j])
			}
		}
	}
}
