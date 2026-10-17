package configcatalog

// Exported formats the export settings offer; the engine's export/formats call is the live list.
var exportFormatIDs = []string{"html", "pdf", "png", "docx", "odt", "txt", "jira-wiki", "jira-adf", "mediawiki"}

func intPointer(value int) *int { return &value }

// vscodeOnly marks a setting only the VS Code extension acts on.
var vscodeOnly = []string{"vscode"}

func choicesOf(values ...string) []Choice {
	choices := make([]Choice, len(values))
	for i, value := range values {
		choices[i] = Choice{Value: value, Label: value}
	}

	return choices
}

// sections are the settings that are not an output or a generator, grouped for a settings page.
// A key is the setting's name after "loreMaster.", the same in every editor that keeps it.
func sections() []Section {
	return []Section{
		{
			ID: "project", Label: "Project", Scope: ScopeProject,
			Settings: []Field{
				{
					Key: "ignore", Label: "Ignore", Type: FieldList,
					Help: "Gitignore-style patterns every output leaves out of the scan, for example CLAUDE.md or internal/.",
				},
				{
					Key: "skipGitignored", Label: "Skip git-ignored files", Type: FieldBool, Default: "true",
					Help: "Leave out Markdown that the workspace's .gitignore files ignore.",
				},
				{
					Key: "confluence.baseUrl", Label: "Confluence address", Type: FieldText, Placeholder: "https://example.atlassian.net/wiki",
					Help: "The Confluence address every output uses unless it sets its own. Set it once in the user's settings; a project overrides it. The token is never a setting.",
				},
				{
					Key: "defaults.direction", Label: "Default direction", Type: FieldChoice, Advanced: true,
					Help:    "The direction of an output that does not set one. Empty means files to platform.",
					Choices: choicesOf("to-platform", "two-way"),
				},
				{
					Key: "defaults.mermaidMode", Label: "Default Mermaid mode", Type: FieldChoice, Advanced: true,
					Help:    "How Mermaid diagrams are stored in an output that does not set it. Empty means image.",
					Choices: choicesOf("image", "code"),
				},
				{
					Key: "defaults.linkMode", Label: "Default link mode", Type: FieldChoice, Advanced: true,
					Help:    "How links between pages are written in an output that does not set it. Empty means title.",
					Choices: choicesOf("title", "id"),
				},
				{
					Key: "defaults.titleCollision", Label: "Default title collision", Type: FieldChoice, Advanced: true,
					Help:    "What to do when an unsynced file's title already exists, in an output that does not set it. Empty means fail.",
					Choices: choicesOf("fail", "adopt"),
				},
				{
					Key: "defaults.titlePrefix", Label: "Default title prefix", Type: FieldText, Advanced: true,
					Help: "The title prefix of an output that does not set one. Empty means the first sync asks for one.",
				},
			},
		},
		{
			ID: "sync", Label: "Sync and watch", Scope: ScopeUser,
			Settings: []Field{
				{
					Key: "generateBeforeSync", Shells: vscodeOnly, Label: "Generate before every sync", Type: FieldBool, Default: "false",
					Help: "Run the generators (test results, API documentation) before every sync, so the pages they write are as fresh as the code.",
				},
				{
					Key: "runPrepareSteps", Shells: vscodeOnly, Label: "Run prepare commands", Type: FieldChoice, Default: "ask",
					Help: "Whether a generator's prepare command runs when you generate. Ask shows the exact commands first.",
					Choices: []Choice{
						{Value: "ask", Label: "Ask first"},
						{Value: "always", Label: "Always"},
						{Value: "never", Label: "Never"},
					},
				},
				{
					Key: "watchDebounceSeconds", Label: "Watch mode wait", Type: FieldNumber, Default: "2", Min: intPointer(1), Max: intPointer(300),
					Help: "In watch mode, how many seconds nothing may change before the changes are synced, so a burst of saves is one sync.",
				},
				{
					Key: "credentials.useKeychain", Shells: vscodeOnly, Label: "Use the system keychain", Type: FieldBool, Default: "false",
					Help: "Remember connections in the operating system's keychain, so every editor and the command line share them and a token is entered once.",
				},
			},
		},
		{
			ID: "export", Label: "Export", Scope: ScopeUser,
			Settings: []Field{
				{
					Key: "export.formats", Shells: vscodeOnly, Label: "Default formats", Type: FieldList,
					Help:        "The formats ticked by default when you export a Markdown file.",
					Suggestions: exportFormatIDs,
				},
				{
					Key: "export.folder", Label: "Export folder", Type: FieldPath, PathKind: "folder",
					Help: "Where exported files go. Empty writes each next to its Markdown file. A folder receives the exports with the document's own folders mirrored below it.",
				},
				{
					Key: "export.pageSize", Label: "PDF page size", Type: FieldChoice, Default: "A4",
					Help:    "The paper a PDF export is printed on.",
					Choices: choicesOf("A4", "A3", "A5", "Letter", "Legal"),
				},
				{
					Key: "export.pngWidth", Label: "PNG width", Type: FieldNumber, Default: "1000", Min: intPointer(320), Max: intPointer(4000),
					Help: "The width in pixels of a PNG export. The image is as tall as the page.",
				},
				{
					Key: "export.docxTemplate", Label: "Word template", Type: FieldPath, PathKind: "file", Advanced: true,
					Help: "A Word file used as the company template for DOCX exports: styles, numbering, headers, footers and page setup are taken from it. Relative to the project, or absolute.",
				},
				{
					Key: "export.odtTemplate", Label: "OpenDocument template", Type: FieldPath, PathKind: "file", Advanced: true,
					Help: "An OpenDocument file used as the company template for ODT exports. Relative to the project, or absolute.",
				},
				{
					Key: "export.browserPath", Label: "Browser for PDF and PNG", Type: FieldPath, PathKind: "file", Advanced: true,
					Help: "The Chrome, Edge or Chromium that prints PDF and PNG exports. Empty uses the one installed in the usual place.",
				},
			},
		},
	}
}
