package settingscommands

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// SaveSettings handles settings/save: validate, refusing with each problem named, then
// merge into the file so the author's comments and key order survive, or write a new
// file with an explanatory header.
func SaveSettings() rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SettingsSaveParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if err := absolute(params.WorkspaceRoot); err != nil {
			return nil, err
		}
		loaded, err := workspacesettings.LoadSettings(params.WorkspaceRoot)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "the existing %s cannot be updated: %s", workspacesettings.FileName, err.Error())
		}
		if err := workspacesettings.SaveSettings(loaded, fromWire(params.Settings)); err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}

		return nil, nil
	}
}
