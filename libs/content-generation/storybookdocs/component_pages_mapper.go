package storybookdocs

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

// DefaultTitle heads the index page when the generator is not given one.
const DefaultTitle = "Component library"

// maxProps bounds the props table: a component with hundreds of inherited props is not
// documented by listing them all.
const maxProps = 50

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug turns a name into a folder or file name.
func slug(name string) string {
	slugged := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slugged == "" {
		return "page"
	}

	return slugged
}

// pageFolders gives every title, ancestors included, its folder below the output: the slug of
// each of its parts, made unique among siblings. Titles must be sorted, so a parent comes first.
func pageFolders(titles []string) map[string]string {
	folders := make(map[string]string, len(titles))
	claimed := map[string]bool{}
	for _, title := range titles {
		parent, name := "", title
		if cut := strings.LastIndex(title, "/"); cut >= 0 {
			parent, name = title[:cut], title[cut+1:]
		}
		base := path.Join(folders[parent], slug(name))
		folder := base
		for n := 2; claimed[folder]; n++ {
			folder = fmt.Sprintf("%s-%d", base, n)
		}
		claimed[folder] = true
		folders[title] = folder
	}

	return folders
}

// withAncestors lists every title and every folder above it, sorted.
func withAncestors(titles []string) []string {
	seen := map[string]bool{}
	for _, title := range titles {
		parts := strings.Split(title, "/")
		for i := 1; i <= len(parts); i++ {
			seen[strings.Join(parts[:i], "/")] = true
		}
	}
	all := make([]string, 0, len(seen))
	for title := range seen {
		all = append(all, title)
	}
	sort.Strings(all)

	return all
}

// ComponentPages builds the pages of the rendered stories: one per component, one per folder
// of the sidebar listing what is in it, and an index; the screenshots are assets next to the
// page that shows them. Failures and console errors go to the warnings, not into the pages, so
// a page changes only when the component does. stories must be ordered by title.
func ComponentPages(stories []renderedStory, indexTitle string, source string) ([]generatedfile.File, []string) {
	byTitle := map[string][]renderedStory{}
	var titles []string
	for _, story := range stories {
		if _, known := byTitle[story.Story.Title]; !known {
			titles = append(titles, story.Story.Title)
		}
		byTitle[story.Story.Title] = append(byTitle[story.Story.Title], story)
	}
	all := withAncestors(titles)
	folders := pageFolders(all)

	var files []generatedfile.File
	var warnings []string
	children := map[string][]string{}
	var roots []string
	for _, title := range all {
		if cut := strings.LastIndex(title, "/"); cut >= 0 {
			children[title[:cut]] = append(children[title[:cut]], title)
		} else {
			roots = append(roots, title)
		}
	}

	for _, title := range all {
		page, images, notes := componentPage(title, byTitle[title], children[title], folders)
		files = append(files, generatedfile.File{Path: path.Join(folders[title], "README.md"), Body: []byte(page)})
		for name, body := range images {
			files = append(files, generatedfile.File{Path: path.Join(folders[title], "images", name), Body: body, Asset: true})
		}
		warnings = append(warnings, notes...)
	}

	var index strings.Builder
	fmt.Fprintf(&index, "# %s\n\nComponent documentation generated from %s: every example was rendered in a real browser and its screenshot is included.\n\n", indexTitle, sourceLink(source))
	for _, root := range roots {
		fmt.Fprintf(&index, "- [%s](./%s/README.md)\n", root, folders[root])
	}
	files = append(files, generatedfile.File{Path: "README.md", Body: []byte(index.String())})

	return files, warnings
}

// componentPage writes one title's page and returns its images by file name.
func componentPage(title string, stories []renderedStory, kids []string, folders map[string]string) (string, map[string][]byte, []string) {
	var page strings.Builder
	var warnings []string
	images := map[string][]byte{}
	fmt.Fprintf(&page, "# %s\n\n", strings.ReplaceAll(title, "/", " / "))
	if len(stories) == 0 {
		page.WriteString("This section groups the pages below.\n\n")
	}
	if description := componentDescription(stories); description != "" {
		page.WriteString(description + "\n\n")
	}
	page.WriteString(propsTable(stories))

	if len(stories) > 0 {
		page.WriteString("## Examples\n\n")
	}
	taken := map[string]bool{}
	for _, story := range stories {
		fmt.Fprintf(&page, "### %s\n\n", story.Story.Name)
		if story.Failure != "" {
			page.WriteString("> This example could not be rendered; see the generator's warnings.\n\n")
			warnings = append(warnings, fmt.Sprintf("%s / %s: %s", title, story.Story.Name, story.Failure))

			continue
		}
		file := uniqueName(slug(story.Story.Name), taken) + ".png"
		images[file] = story.PNG
		fmt.Fprintf(&page, "![%s](./images/%s)\n\n", escapeAlt(story.Story.Name), file)
		if text := nested(story.Meta.Docs, "description", "story"); text != "" {
			page.WriteString(text + "\n\n")
		}
		if code := storySource(story.Meta.Docs); code != "" {
			fmt.Fprintf(&page, "```tsx\n%s\n```\n\n", code)
		}
		if len(story.Errors) > 0 {
			warnings = append(warnings, fmt.Sprintf("%s / %s: %d console error(s), the first: %s", title, story.Story.Name, len(story.Errors), story.Errors[0]))
		}
	}

	if len(kids) > 0 {
		page.WriteString("## In this section\n\n")
		for _, kid := range kids {
			relative := strings.TrimPrefix(folders[kid], folders[title]+"/")
			fmt.Fprintf(&page, "- [%s](./%s/README.md)\n", kid[strings.LastIndex(kid, "/")+1:], relative)
		}
		page.WriteString("\n")
	}
	if len(stories) > 0 && stories[0].Story.ImportPath != "" {
		fmt.Fprintf(&page, "<sub>Story source: `%s`</sub>\n", stories[0].Story.ImportPath)
	}

	return page.String(), images, warnings
}

// uniqueName appends a number to a name already taken on the page.
func uniqueName(name string, taken map[string]bool) string {
	unique := name
	for n := 2; taken[unique]; n++ {
		unique = fmt.Sprintf("%s-%d", name, n)
	}
	taken[unique] = true

	return unique
}

func escapeAlt(text string) string {
	return strings.NewReplacer("[", `\[`, "]", `\]`).Replace(text)
}

// componentDescription is the text the component's docs parameters give it.
func componentDescription(stories []renderedStory) string {
	for _, story := range stories {
		if text := nested(story.Meta.Docs, "description", "component"); text != "" {
			return text
		}
	}

	return ""
}

// storySource is the code a story's docs parameters show for it.
func storySource(docs map[string]any) string {
	code := nested(docs, "source", "originalSource")
	if code == "" {
		code = nested(docs, "source", "code")
	}

	return strings.TrimSpace(code)
}

// nested reads a string two levels down in a decoded JSON object; empty when it is absent.
func nested(object map[string]any, first string, second string) string {
	inner, ok := object[first].(map[string]any)
	if !ok {
		return ""
	}
	text, _ := inner[second].(string)

	return text
}

// propsTable lists the props of the first story that declares any.
func propsTable(stories []renderedStory) string {
	for _, story := range stories {
		if len(story.Meta.ArgTypes) == 0 {
			continue
		}
		var names []string
		for name, argType := range story.Meta.ArgTypes {
			if table, ok := argType["table"].(map[string]any); ok && table["disable"] == true {
				continue
			}
			names = append(names, name)
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)

		var out strings.Builder
		out.WriteString("## Props\n\n| Name | Type | Default | Description |\n|---|---|---|---|\n")
		for i, name := range names {
			if i >= maxProps {
				fmt.Fprintf(&out, "\n%d more props are not listed.\n", len(names)-maxProps)

				break
			}
			argType := story.Meta.ArgTypes[name]
			kind, fallback := "", ""
			if table, ok := argType["table"].(map[string]any); ok {
				kind = summary(table, "type")
				fallback = summary(table, "defaultValue")
			}
			if kind == "" {
				if declared, ok := argType["type"].(map[string]any); ok {
					kind, _ = declared["name"].(string)
				}
			}
			description, _ := argType["description"].(string)
			fmt.Fprintf(&out, "| `%s` | %s | %s | %s |\n", name, cell(kind), cell(fallback), cell(description))
		}
		out.WriteString("\n")

		return out.String()
	}

	return ""
}

func summary(table map[string]any, key string) string {
	inner, ok := table[key].(map[string]any)
	if !ok {
		return ""
	}
	text, _ := inner["summary"].(string)

	return text
}

// cell makes text safe for one cell of a Markdown table.
func cell(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "|", `\|`)
	if runes := []rune(text); len(runes) > 160 {
		text = string(runes[:160]) + "…"
	}

	return text
}

// sourceLink names where the pages came from: a link for a URL, the folder's name otherwise,
// since a folder of the machine means nothing to a reader.
func sourceLink(source string) string {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return "[" + source + "](" + source + ")"
	}

	return "`" + strings.Trim(source, "/") + "`"
}
