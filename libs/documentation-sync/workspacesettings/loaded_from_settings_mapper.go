package workspacesettings

// LoadedFromSettings wraps settings that came from the editor's settings layers (#308)
// rather than from .lore-master.yaml, so every consumer keeps one type. It carries no
// file to merge into.
func LoadedFromSettings(settings Settings, firstSync bool) Loaded {
	return Loaded{Settings: settings, FirstSync: firstSync, Exists: true}
}
