package settingscommands

import (
	"context"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// ReadSettings handles settings/read. A file that is not valid settings (bad YAML, an
// unknown or secret-looking key, a value out of range) is an invalid-settings error
// naming each problem.
func ReadSettings() rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SettingsReadParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if err := absolute(params.WorkspaceRoot); err != nil {
			return nil, err
		}
		loaded, err := workspacesettings.LoadSettings(params.WorkspaceRoot)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}
		return rpcprotocol.SettingsReadResult{Exists: loaded.Exists, FirstSync: loaded.FirstSync, Settings: toWire(loaded.Settings)}, nil
	}
}

func absolute(root string) error {
	if !filepath.IsAbs(root) {
		return rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "workspaceRoot must be an absolute path, got %q", root)
	}

	return nil
}
