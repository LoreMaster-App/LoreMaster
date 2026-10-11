package settingscommands

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/editorsettings"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// SaveSettings handles settings/save: validate, refusing with each problem named, then
// write. The configuration goes to the folder's .vscode/settings.json, unless the
// workspace is still configured by a .lore-master.yaml (and has no editor settings yet),
// in which case that file is merged into so the author's comments and key order survive.
func SaveSettings() rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SettingsSaveParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if err := absolute(params.WorkspaceRoot); err != nil {
			return nil, err
		}
		settings := fromWire(params.Settings)

		result, err := editorsettings.LoadEditorSettings(params.WorkspaceRoot, editorsettings.CurrentUserSettingsPath())
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "the existing settings cannot be updated: %s", err.Error())
		}
		if result.Configured || !yamlExists(params.WorkspaceRoot) {
			if err := editorsettings.SaveFolderSettings(params.WorkspaceRoot, settings); err != nil {
				return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
			}

			return nil, nil
		}

		loaded, err := workspacesettings.LoadSettings(params.WorkspaceRoot)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "the existing %s cannot be updated: %s", workspacesettings.FileName, err.Error())
		}
		if err := workspacesettings.SaveSettings(loaded, settings); err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}

		return nil, nil
	}
}

func yamlExists(workspaceRoot string) bool {
	_, err := os.Stat(filepath.Join(workspaceRoot, workspacesettings.FileName))

	return !errors.Is(err, fs.ErrNotExist)
}
