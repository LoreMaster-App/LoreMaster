package configcatalog

import "lore-master/libs/documentation-sync/templatereference"

// repoPattern is "owner/name", a clone URL or an scp-style git address.
const repoPattern = `^(?:[\w.-]+/[\w.-]+|\S+://\S+|git@\S+)$`

const repoPatternHelp = "Use owner/name or a clone URL, or leave it empty for this workspace's own origin."

func githubPagesPlatform() Platform {
	return Platform{
		ID: "github-pages", Label: "GitHub Pages", Icon: "site", Delivery: DeliverySite,
		Description:  "A static site published to a branch of a GitHub repository.",
		NameTemplate: "GitHub Pages · {repo|this repository}",
		Fields: []Field{
			{
				Key: "repo", Label: "Repository", Type: FieldText, Placeholder: "owner/name",
				Help:    "\"owner/name\" or a clone URL. Empty means this workspace's own origin.",
				Pattern: repoPattern, PatternHelp: repoPatternHelp,
			},
			{
				Key: "branch", Label: "Branch", Type: FieldText, Default: "gh-pages", Advanced: true,
				Help: "The branch published to.",
			},
			{
				Key: "path", Label: "Folder in the branch", Type: FieldText, Advanced: true,
				Help: "The folder inside the branch the site goes to, so another site can share the branch. Empty owns the whole branch.",
			},
			{
				Key: "template", Label: "Template", Type: FieldText, Advanced: true,
				Help:        "A built-in layout, or a folder holding template.html. Write ./name for a folder named like a built-in.",
				Suggestions: templatereference.BuiltinNames("github-pages"),
			},
		},
	}
}

func githubWikiPlatform() Platform {
	return Platform{
		ID: "github-wiki", Label: "GitHub Wiki", Icon: "wiki", Delivery: DeliverySite,
		Description:  "The repository's GitHub wiki.",
		NameTemplate: "GitHub Wiki · {repo|this repository}",
		Fields: []Field{
			{
				Key: "repo", Label: "Repository", Type: FieldText, Placeholder: "owner/name",
				Help:    "\"owner/name\" or a clone URL. Empty means this workspace's own origin.",
				Pattern: repoPattern, PatternHelp: repoPatternHelp,
			},
			{
				Key: "branch", Label: "Branch", Type: FieldText, Advanced: true,
				Help: "The wiki's branch. Empty means its default branch.",
			},
			{
				Key: "template", Label: "Template", Type: FieldText, Advanced: true,
				Help:        "A Markdown file with {{content}} where the page goes. Empty is the default layout.",
				Suggestions: templatereference.BuiltinNames("github-wiki"),
			},
		},
	}
}
