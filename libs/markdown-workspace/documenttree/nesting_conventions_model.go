package documenttree

// Convention is one of LoreMaster's documentation rules, described for a person or an
// authoring agent. Key is a stable identifier (the nesting rules reuse the Rule values);
// Summary explains the rule in one sentence.
type Convention struct {
	Key     string `json:"key"`
	Summary string `json:"summary"`
}

// NestingConventions describes, in the order the policy applies them, how LoreMaster
// decides a page's parent, followed by how it titles pages. It is the single human-facing
// description of these rules; NestingPolicy in this package is what enforces them, so the
// two are meant to be read and changed together. Anything that teaches an author the
// rules (for example the MCP server) builds on this rather than restating them.
func NestingConventions() []Convention {
	return []Convention{
		{
			Key:     string(RuleExplicitParent),
			Summary: "A `parent:` line in the file's `<!-- lore-master ... -->` annotation block wins (not YAML front matter, which is ignored): it names another synced document — a path relative to this file, or absolute from the workspace root with a leading `/` — matched case-insensitively. If it names no synced document, or the document itself, the next rule applies.",
		},
		{
			Key:     string(RuleDottedName),
			Summary: "A dotted file name nests under the file one segment shorter in the same directory: `readme.architecture.md` nests under `readme.md` (case-insensitive, so `readme.setup.md` finds `README.md`). If that direct parent is missing, the nearest existing shorter prefix is used.",
		},
		{
			Key:     string(RuleDirectoryIndex),
			Summary: "A directory's index page — `README.md`, else `index.md` (any case) — is the parent of the other documents in that directory and of its subdirectories' index pages. A directory with no index hands its documents to the nearest ancestor directory that has one.",
		},
		{
			Key:     string(RuleSelectedParent),
			Summary: "A document that no other rule placed nests directly under the page the user selected as the sync target.",
		},
		{
			Key:     "title",
			Summary: "A page's title is its first `# H1`; a `title:` line in the `<!-- lore-master ... -->` annotation block overrides it (not YAML front matter, which is ignored); with no H1, the file name is used.",
		},
		{
			Key:     "title-prefix",
			Summary: "Published page titles are `<titlePrefix>: <title>`. The prefix is chosen once, at the first sync, defaulting to the selected parent page's title.",
		},
		{
			Key:     "title-unique",
			Summary: "Confluence page titles are unique per space; a would-be clash is resolved by the output's title-collision setting (adopt the existing page, or fail).",
		},
	}
}
