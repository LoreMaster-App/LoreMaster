package editorsettings

import (
	"os"
	"runtime"

	"lore-master/libs/documentation-sync/workspacesettings"
)

// CurrentUserSettingsPath is where the user's settings.json is on this machine.
func CurrentUserSettingsPath() string {
	home, _ := os.UserHomeDir()

	return UserSettingsPath(runtime.GOOS, os.Getenv, home)
}

// LoadConfiguration is what every consumer of the configuration calls: the editor's
// settings layers when they hold loreMaster.outputs, otherwise the deprecated
// .lore-master.yaml, otherwise the defaults of a workspace not set up yet.
func LoadConfiguration(workspaceRoot string) (workspacesettings.Loaded, error) {
	result, err := LoadEditorSettings(workspaceRoot, CurrentUserSettingsPath())
	if err != nil {
		return workspacesettings.Loaded{}, err
	}
	if result.Configured {
		return workspacesettings.LoadedFromSettings(result.Settings, result.FirstSync), nil
	}

	return workspacesettings.LoadSettings(workspaceRoot)
}
