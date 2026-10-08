package markdownnormalising

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

// Options say how to treat one tool's output.
type Options struct {
	// IndexTitle is the title of the output's index page (README.md), which is written when
	// the tool did not write one.
	IndexTitle string
	// DropLinePrefixes removes every line that starts with one of them, after leading space:
	// the lines a tool stamps in that change without the code changing, such as "Defined in:"
	// source links carrying a line number or a commit.
	DropLinePrefixes []string
}

var (
	blankRuns    = regexp.MustCompile(`\n{3,}`)
	frontMatter  = regexp.MustCompile(`(?s)\A---\n.*?\n---\n`)
	topHeading   = regexp.MustCompile(`(?m)^# +(.+?) *$`)
	fencedOpener = regexp.MustCompile("^(```|~~~)")
)

// NormalisePages cleans each page, gives every page a title no other page has, and adds an
// index page when the tool wrote none. Pages come back sorted by path; the warnings say what
// had to be repaired (a page without a heading, a title made unique).
func NormalisePages(pages []generatedfile.File, options Options) ([]generatedfile.File, []string) {
	var warnings []string
	sorted := slices.Clone(pages)
	slices.SortFunc(sorted, func(a, b generatedfile.File) int { return strings.Compare(a.Path, b.Path) })

	cleaned := make([]generatedfile.File, len(sorted))
	for i, page := range sorted {
		body, note := cleanPage(page, options)
		if note != "" {
			warnings = append(warnings, note)
		}
		cleaned[i] = generatedfile.File{Path: page.Path, Body: []byte(body)}
	}

	titles := map[string]bool{strings.ToLower(options.IndexTitle): options.IndexTitle != ""}
	for i, page := range cleaned {
		body, changed := uniqueTitle(page, titles, options.IndexTitle)
		if changed != "" {
			warnings = append(warnings, changed)
		}
		cleaned[i].Body = []byte(body)
	}

	if !slices.ContainsFunc(cleaned, func(page generatedfile.File) bool { return page.Path == "README.md" }) {
		cleaned = append([]generatedfile.File{{Path: "README.md", Body: []byte(indexPage(options.IndexTitle, cleaned))}}, cleaned...)
	}

	return cleaned, warnings
}

// cleanPage fixes line endings, removes front matter and stamped lines, tidies whitespace, and
// makes sure the page opens with a heading.
func cleanPage(page generatedfile.File, options Options) (string, string) {
	text := strings.ReplaceAll(string(page.Body), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\ufeff")
	text = frontMatter.ReplaceAllString(text, "")

	var kept []string
	inFence := false
	for _, line := range strings.Split(text, "\n") {
		if fencedOpener.MatchString(strings.TrimSpace(line)) {
			inFence = !inFence
		}
		if !inFence && startsWithAny(strings.TrimSpace(line), options.DropLinePrefixes) {
			continue
		}
		kept = append(kept, strings.TrimRight(line, " \t"))
	}
	text = blankRuns.ReplaceAllString(strings.Join(kept, "\n"), "\n\n")
	text = strings.TrimSpace(text)

	note := ""
	if !strings.HasPrefix(text, "# ") {
		note = fmt.Sprintf("%s had no top heading; one was added from its file name", page.Path)
		text = "# " + titleFromPath(page.Path) + "\n\n" + text
	}

	return strings.TrimRight(text, "\n") + "\n", note
}

func startsWithAny(line string, prefixes []string) bool {
	return slices.ContainsFunc(prefixes, func(prefix string) bool { return prefix != "" && strings.HasPrefix(line, prefix) })
}

// titleFromPath names a page after its file: README.md after its folder.
func titleFromPath(file string) string {
	name := strings.TrimSuffix(path.Base(file), path.Ext(file))
	if strings.EqualFold(name, "readme") && path.Dir(file) != "." {
		name = path.Base(path.Dir(file))
	}

	return name
}

// uniqueTitle makes the page's title one nobody has used: a repeated title gets the page's
// folder in brackets, then a number. It records the title it ended with.
func uniqueTitle(page generatedfile.File, taken map[string]bool, indexTitle string) (string, string) {
	body := string(page.Body)
	match := topHeading.FindStringSubmatchIndex(body)
	title := strings.TrimSpace(body[match[2]:match[3]])

	// The index page is the one page entitled to the index title.
	if page.Path == "README.md" && indexTitle != "" && strings.EqualFold(title, indexTitle) {
		return body, ""
	}
	unique := title
	if taken[strings.ToLower(unique)] {
		unique = fmt.Sprintf("%s (%s)", title, scopeOf(page.Path))
	}
	for n := 2; taken[strings.ToLower(unique)]; n++ {
		unique = fmt.Sprintf("%s (%s) #%d", title, scopeOf(page.Path), n)
	}
	taken[strings.ToLower(unique)] = true
	if unique == title {
		return body, ""
	}

	return body[:match[2]] + unique + body[match[3]:], fmt.Sprintf("%s: the title %q was already used; it is now %q", page.Path, title, unique)
}

// scopeOf is what tells two pages with the same title apart: the folder, or the file name for
// a page at the top.
func scopeOf(file string) string {
	if dir := path.Dir(file); dir != "." {
		return dir
	}

	return strings.TrimSuffix(path.Base(file), path.Ext(file))
}

// indexPage lists the pages for a tool that wrote no index of its own.
func indexPage(title string, pages []generatedfile.File) string {
	if title == "" {
		title = "API reference"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	for _, page := range pages {
		heading := topHeading.FindStringSubmatch(string(page.Body))
		fmt.Fprintf(&out, "- [%s](%s)\n", strings.NewReplacer("[", `\[`, "]", `\]`).Replace(heading[1]), page.Path)
	}

	return out.String()
}
