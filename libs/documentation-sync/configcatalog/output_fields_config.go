package configcatalog

// contentFields are the settings of the Markdown an output reads.
func contentFields() []Field {
	return []Field{
		{
			Key: "roots", Label: "Folders to sync", Type: FieldList, Default: ".",
			Help: "Folders relative to the workspace. \".\" is the whole workspace.",
		},
		{
			Key: "excludes", Label: "Excluded paths", Type: FieldList,
			Help: "Gitignore-style patterns this storage leaves out.",
		},
	}
}

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
