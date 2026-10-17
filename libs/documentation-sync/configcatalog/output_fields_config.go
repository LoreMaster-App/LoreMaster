package configcatalog

// outputFields are the settings every storage has, whatever the platform.
func outputFields() []Field {
	return []Field{
		{
			Key: "name", Label: "Name", Type: FieldText,
			Help: "What this output is called in the tree, the sync and the settings, so two of the same platform are told apart. Empty shows a name made from the platform and where it goes.",
		},
		{
			Key: "include", Label: "Include", Type: FieldList, Advanced: true,
			Help: "Gitignore-style patterns this output reads even though an ignore or exclude list leaves them out.",
		},
		{
			Key: "exclude", Label: "Exclude", Type: FieldList, Advanced: true,
			Help: "Gitignore-style patterns this output leaves out, on top of the workspace's ignore list.",
		},
	}
}
