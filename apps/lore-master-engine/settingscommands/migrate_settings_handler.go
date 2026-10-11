package settingscommands

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/editorsettings"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// MigrateSettings handles settings/migrate: a workspace that is still configured by a
// .lore-master.yaml gets that configuration written into its .vscode/settings.json (#308).
// Nothing happens when the editor's settings already hold the configuration or there is no
// yaml. The yaml is not deleted; the editor tells the user it is now unused.
func MigrateSettings() rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SettingsReadParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if err := absolute(params.WorkspaceRoot); err != nil {
			return nil, err
		}

		result, err := editorsettings.LoadEditorSettings(params.WorkspaceRoot, editorsettings.CurrentUserSettingsPath())
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}
		if result.Configured || !yamlExists(params.WorkspaceRoot) {
			return rpcprotocol.SettingsMigrateResult{}, nil
		}
		loaded, err := workspacesettings.LoadSettings(params.WorkspaceRoot)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}
		if err := editorsettings.SaveFolderSettings(params.WorkspaceRoot, loaded.Settings); err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}

		return rpcprotocol.SettingsMigrateResult{Migrated: true, Path: editorsettings.FolderSettingsPath(params.WorkspaceRoot)}, nil
	}
}
