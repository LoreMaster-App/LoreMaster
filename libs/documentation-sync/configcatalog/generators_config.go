package configcatalog

// generatorFields are the settings every generator has.
func generatorFields() []Field {
	return []Field{
		{
			Key: "type", Label: "Type", Type: FieldChoice,
			Help: "What the generator reads. Empty recognises the type from input.",
		},
		{
			Key: "output", Label: "Output folder", Type: FieldPath, PathKind: "folder", Required: true,
			Help: "The folder, inside the workspace, the pages are written to. Include it in an output's content roots for the pages to sync.",
		},
		{
			Key: "input", Label: "Input", Type: FieldList,
			Help: "Gitignore-style patterns selecting what to read. Empty means the type's own default.",
		},
		{
			Key: "title", Label: "Index page title", Type: FieldText, Advanced: true,
			Help: "Heads the generated index page. Empty means the type's own default.",
		},
		{
			Key: "prepare", Label: "Prepare command", Type: FieldText, Advanced: true,
			Help: "A shell command that produces what the generator reads (a build, a test run). It runs only when a run asks for it, never on its own.",
		},
	}
}

func generatorTypes() []GeneratorType {
	return []GeneratorType{
		{ID: "test-results", Label: "Test results", Description: "JUnit XML test reports."},
		{ID: "coverage", Label: "Coverage", Description: "Code coverage reports."},
		{ID: "markdown-docs", Label: "Markdown documents", Description: "Existing documents brought in as pages."},
		{ID: "go-docs", Label: "Go documentation", Description: "Go package documentation."},
		{ID: "openapi-docs", Label: "OpenAPI", Description: "OpenAPI 3 descriptions."},
		{ID: "ts-docs", Label: "TypeScript documentation", Description: "TypeScript and JavaScript source documentation, through TypeDoc."},
		{ID: "python-docs", Label: "Python documentation", Description: "Python source documentation."},
		{ID: "csharp-docs", Label: "C# documentation", Description: "C# source documentation."},
		{ID: "dart-docs", Label: "Dart documentation", Description: "Dart source documentation."},
		{
			ID: "storybook-docs", Label: "Storybook", Description: "The components of a Storybook.",
			Fields: []Field{{
				Key: "source", Label: "Storybook", Type: FieldText, Required: true,
				Placeholder: "storybook-static",
				Help:        "The URL of a Storybook, or the workspace folder of its build. Input then filters the stories by title (Components/, !Internal/).",
			}},
		},
	}
}
