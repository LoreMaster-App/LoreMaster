package settingscommands

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/editorsettings"
)

// SaveSettings handles settings/save: validate, refusing with each problem named, then write the
// configuration where the user keeps it (the open .code-workspace file, else the folder's
// .vscode/settings.json), keeping comments and every other setting.
func SaveSettings() rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SettingsSaveParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if err := absolute(params.WorkspaceRoot); err != nil {
			return nil, err
		}

		if _, err := editorsettings.LoadEditorSettings(params.WorkspaceRoot, editorsettings.CurrentUserSettingsPath()); err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "the existing settings cannot be updated: %s", err.Error())
		}
		if _, err := editorsettings.SaveConfiguration(params.WorkspaceRoot, fromWire(params.Settings)); err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}

		return nil, nil
	}
}
