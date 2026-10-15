package editorsettings

import (
	"errors"
	"io/fs"
	"os"

	"lore-master/libs/documentation-sync/workspacesettings"
)

// workspaceFileSettingsKey is where a .code-workspace file keeps its settings.
const workspaceFileSettingsKey = "settings"

// ConfigurationTarget is where a save goes.
type ConfigurationTarget struct {
	// Path is the file written.
	Path string
	// InWorkspaceFile is true when it is a .code-workspace file, whose settings sit under
	// "settings"; false for a folder's .vscode/settings.json.
	InWorkspaceFile bool
}

// TargetFor says where the configuration of a workspace folder is written: the .code-workspace
// file the editor has open (LORE_MASTER_WORKSPACE_FILE), and only when there is none the folder's
// .vscode/settings.json. A folder that already holds LoreMaster's own keys keeps them: they are
// what wins over the workspace file, so writing anywhere else would be shadowed by them.
func TargetFor(workspaceRoot string) (ConfigurationTarget, error) {
	folder := ConfigurationTarget{Path: FolderSettingsPath(workspaceRoot)}
	workspaceFile := os.Getenv(WorkspaceFileVariable)
	if workspaceFile == "" {
		return folder, nil
	}
	if _, err := os.Stat(workspaceFile); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return folder, nil
		}

		return ConfigurationTarget{}, err
	}
	holds, err := folderHoldsConfiguration(workspaceRoot)
	if err != nil {
		return ConfigurationTarget{}, err
	}
	if holds {
		return folder, nil
	}

	return ConfigurationTarget{Path: workspaceFile, InWorkspaceFile: true}, nil
}

// folderHoldsConfiguration is whether the folder's settings.json sets any key LoreMaster reads.
func folderHoldsConfiguration(workspaceRoot string) (bool, error) {
	layer, err := ReadLayer(FolderSettingsPath(workspaceRoot))
	if err != nil {
		return false, err
	}
	for key := range layer.Values {
		if configKeys[key] {
			return true, nil
		}
	}

	return false, nil
}

// SaveConfiguration writes the settings where the user keeps their configuration (TargetFor):
// into the open .code-workspace file, so a workspace that is configured there never grows a
// competing .vscode folder, and into .vscode/settings.json only when there is no workspace file.
// Comments and every other setting stay where they were.
func SaveConfiguration(workspaceRoot string, settings workspacesettings.Settings) (ConfigurationTarget, error) {
	target, err := TargetFor(workspaceRoot)
	if err != nil {
		return ConfigurationTarget{}, err
	}

	return target, saveInto(target.Path, target.InWorkspaceFile, workspaceRoot, settings)
}
