package workspacesettings

// ResolveSettings completes settings assembled somewhere other than .lore-master.yaml (the
// editor's settings layers, #308): empty fields take their defaults and the result is
// validated, exactly as for a file. It also reports whether an output still lacks its
// first-sync answers.
func ResolveSettings(settings Settings) (Settings, bool, error) {
	applyDefaults(&settings)
	if err := Validate(settings); err != nil {
		return Settings{}, false, err
	}

	return settings, needsFirstSync(settings), nil
}

// ResolveEditorSettings is ResolveSettings for settings read from the editor's layers, where an
// empty loreMaster.outputs is meaningful: the user removed every storage. It is configured as
// having none, and never falls back to a leftover .lore-master.yaml.
func ResolveEditorSettings(settings Settings) (Settings, bool, error) {
	applyDefaults(&settings)
	if err := ValidateAllowingNoOutputs(settings); err != nil {
		return Settings{}, false, err
	}
	if len(settings.Outputs) == 0 {
		return settings, true, nil
	}

	return settings, needsFirstSync(settings), nil
}
