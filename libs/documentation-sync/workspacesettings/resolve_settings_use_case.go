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
