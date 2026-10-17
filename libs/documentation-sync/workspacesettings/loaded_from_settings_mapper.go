package workspacesettings

// LoadedFromSettings wraps settings that came from the editor's settings layers (#308), so every
// consumer keeps one type.
func LoadedFromSettings(settings Settings, firstSync bool) Loaded {
	return Loaded{Settings: settings, FirstSync: firstSync, Exists: true}
}
