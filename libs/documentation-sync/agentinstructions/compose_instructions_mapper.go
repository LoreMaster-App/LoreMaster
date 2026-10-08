package agentinstructions

import (
	"fmt"
	"slices"
	"strings"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documenttree"
)

// Compose writes the instructions as Markdown, without a top-level heading, so each tool's
// file can frame them its own way. exists says whether the workspace has a
// .lore-master.yaml; without one the workspace section says so instead of guessing. The
// result depends only on its arguments.
func Compose(settings workspacesettings.Settings, exists bool, conventions []documenttree.Convention) string {
	var out strings.Builder

	out.WriteString("You write and edit this project's documentation as Markdown files. LoreMaster syncs those files " +
		"to a documentation platform and turns the file tree into the page tree, so where a file lives, what it is " +
		"called and how it starts decide where its page ends up. Follow the rules below exactly.\n\n")

	out.WriteString("## How LoreMaster organises pages\n\n")
	out.WriteString("The nesting rules are tried in this order; the first that applies decides a page's parent. Titles follow.\n\n")
	for i, convention := range conventions {
		fmt.Fprintf(&out, "%d. **%s** — %s\n", i+1, convention.Key, convention.Summary)
	}
	out.WriteString("\n")

	writeWorkspace(&out, settings, exists)
	writeWorkflow(&out)
	writeStyle(&out)

	return strings.TrimRight(out.String(), "\n") + "\n"
}

func writeWorkspace(out *strings.Builder, settings workspacesettings.Settings, exists bool) {
	out.WriteString("## This workspace\n\n")
	if !exists {
		out.WriteString("There is no `.lore-master.yaml` yet; the first sync creates it and asks where the pages go. Until then, " +
			"write pages anywhere under the workspace following the rules above.\n\n")

		return
	}

	for _, output := range settings.Outputs {
		fmt.Fprintf(out, "- %s\n", describeOutput(output))
	}
	out.WriteString("\n")

	roots := syncedRoots(settings)
	if len(roots) > 0 {
		fmt.Fprintf(out, "Only Markdown under %s is synced; a file anywhere else never reaches the platform, so put pages there.\n\n", codeList(roots))
	}
	if left := leftOut(settings); len(left) > 0 {
		fmt.Fprintf(out, "Left out of the sync: %s. Do not put pages that should be published there.\n\n", codeList(left))
	}
	if settings.SkipGitignored != nil && !*settings.SkipGitignored {
		out.WriteString("Files ignored by `.gitignore` are still synced here (`skipGitignored: false`).\n\n")
	} else {
		out.WriteString("Markdown that a `.gitignore` ignores is not synced.\n\n")
	}

	if len(settings.Generators) > 0 {
		out.WriteString("**Generated pages.** These folders are written by LoreMaster generators from other project artifacts. " +
			"Never edit a file in them by hand: your change is overwritten the next time the generator runs. Change the source " +
			"instead, then ask the user to run **LoreMaster: Run generators**.\n\n")
		for _, generator := range settings.Generators {
			fmt.Fprintf(out, "- `%s` — written by the `%s` generator.\n", strings.TrimRight(generator.Output, "/"), generator.Type)
		}
		out.WriteString("\n")
	}
}

// describeOutput is one storage in a sentence, with what matters to a writer.
func describeOutput(output workspacesettings.Output) string {
	switch output.Platform {
	case "confluence":
		text := fmt.Sprintf("Confluence space `%s` at %s", output.Space, output.BaseURL)
		if prefix := strings.TrimSpace(output.TitlePrefix); prefix != "" {
			text += fmt.Sprintf("; page titles are `%s: <the page's first heading>`", prefix)
		}
		if output.Direction == "two-way" {
			text += ". Sync is two-way: edits made on the platform are pulled back into these files, so the files can change after a sync and conflicting edits are reported rather than merged"
		}
		switch output.MermaidMode {
		case "code":
			text += ". Mermaid diagrams stay code blocks"
		default:
			text += ". Mermaid diagrams in ```mermaid blocks are rendered to images"
		}

		return text + "."
	case "github-pages":
		return "A static site on GitHub Pages: the Markdown is published as written and rendered in the browser."
	default:
		return fmt.Sprintf("A `%s` storage.", output.Platform)
	}
}

// syncedRoots are the folders the outputs read, in the order first met, without repeats; the
// whole workspace is not listed as a folder.
func syncedRoots(settings workspacesettings.Settings) []string {
	var roots []string
	for _, output := range settings.Outputs {
		for _, content := range output.Content {
			if content.Type != "markdown" {
				continue
			}
			for _, root := range content.Roots {
				if root != "." && root != "" && !slices.Contains(roots, root) {
					roots = append(roots, root)
				}
			}
		}
	}

	return roots
}

// leftOut are the patterns the sync skips: the top-level ignore list and every content
// entry's excludes, without repeats.
func leftOut(settings workspacesettings.Settings) []string {
	left := slices.Clone(settings.Ignore)
	for _, output := range settings.Outputs {
		for _, content := range output.Content {
			for _, pattern := range content.Excludes {
				if !slices.Contains(left, pattern) {
					left = append(left, pattern)
				}
			}
		}
	}

	return left
}

func codeList(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = "`" + value + "`"
	}

	return strings.Join(quoted, ", ")
}

func writeWorkflow(out *strings.Builder) {
	out.WriteString("## How to work\n\n")
	out.WriteString("1. **Look before you write.** If the LoreMaster tools are available, call `loremaster_nesting_rules` once and `preview_tree` " +
		"to see the pages that exist and how they nest. Otherwise read the existing `.md` files and follow the rules above.\n")
	out.WriteString("2. **Place the page.** Use `place_document` with the page's title and, if you know it, its parent: it returns the file path " +
		"and, when a dotted name cannot do it, the `<!-- lore-master ... -->` block with a `parent:` line to put at the very top of the file. " +
		"Without the tool, name the file so a rule puts it there (a dotted name such as `readme.setup.md` under `readme.md`, or a file in " +
		"the folder whose `README.md` is its parent), or add the `parent:` line yourself.\n")
	out.WriteString("3. **Write the page.** Start with one `# Title` heading; it becomes the page title and must be unique among the pages. " +
		"Link to other pages with relative `.md` paths, and to images and files with relative paths: LoreMaster resolves them. " +
		"Keep one topic per page and nest by meaning, not by habit.\n")
	out.WriteString("4. **Check it.** Call `validate_document` on every file you wrote or moved and fix what it reports: a title that clashes " +
		"with another page, a missing heading, a parent that does not exist, a file the sync would not include.\n")
	out.WriteString("5. **Leave the sync alone.** In a file's `<!-- lore-master ... -->` block you may write only `parent:` and `title:`; the other " +
		"keys (page id, version, hashes, `generated`) belong to the sync, so never write or change them. YAML front matter does not " +
		"set a parent or a title. Do not run the sync yourself unless you are asked to; tell the user to run **LoreMaster: Sync** " +
		"when the pages are ready.\n\n")
}

func writeStyle(out *strings.Builder) {
	out.WriteString("## Style\n\n")
	out.WriteString("- Write for the reader who opens the page cold: say what it is for in the first sentence.\n")
	out.WriteString("- Prefer short pages that link to each other over one long page; use headings, lists and tables, and code blocks with a language.\n")
	out.WriteString("- Change existing pages rather than adding near-duplicates, and keep a page's title stable: renaming it renames the page on the platform.\n")
	out.WriteString("- Say what is true now, not what was true; date a statement only when the date matters.\n")
}
