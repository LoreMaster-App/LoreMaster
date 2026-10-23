package editorsettings

import "lore-master/libs/documentation-sync/workspacesettings"

// LoadDiscoveryScope is which files the workspace's ignore settings (loreMaster.ignore, loreMaster.skipGitignored) leave out of a
// scan. Unlike LoadEditorSettings it needs no storage to be configured: the ignore list is the project's, whether or not anything is
// synced. A workspace with neither setting gets the defaults (nothing listed, .gitignore honoured).
func LoadDiscoveryScope(workspaceRoot, userSettingsPath string) (workspacesettings.DiscoveryScope, error) {
	layers, err := readLayers(workspaceRoot, userSettingsPath)
	if err != nil {
		return workspacesettings.DiscoveryScope{}, err
	}
	values := mergeLayers(layers)
	var settings workspacesettings.Settings
	if err := decodeKey(values, keyIgnore, &settings.Ignore); err != nil {
		return workspacesettings.DiscoveryScope{}, err
	}
	if err := decodeKey(values, keySkipGitignored, &settings.SkipGitignored); err != nil {
		return workspacesettings.DiscoveryScope{}, err
	}

	return settings.DiscoveryScope(), nil
}
