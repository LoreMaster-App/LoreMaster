package editorsettings

import (
	"os"
	"path/filepath"

	"lore-master/libs/documentation-sync/workspacesettings"
)

// FolderSettingsPath is the workspace folder's own settings file.
func FolderSettingsPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".vscode", "settings.json")
}

// WorkspaceFileVariable names the environment variable that points at the .code-workspace
// file of a multi-root workspace, whose `settings` are the layer between user and folder.
const WorkspaceFileVariable = "LORE_MASTER_WORKSPACE_FILE"

// OverridesVariable names the environment variable that carries overrides: the JSON of
// `loreMaster.*` keys (or "@" and a file path), the most specific layer, for CI and the
// GitHub Action (#313).
const OverridesVariable = "LORE_MASTER_SETTINGS"

// LoadEditorSettings resolves the configuration from the user's settings, the
// .code-workspace file's, the workspace folder's and finally the overrides, each more
// specific one winning (#308, #313, #314). userSettingsPath may be empty. When no layer sets loreMaster.outputs the result is not Configured and holds
// no settings, so the caller reads the defaults of a workspace not set up. Otherwise the settings
// are completed and validated, and an error names the file and key.
func LoadEditorSettings(workspaceRoot, userSettingsPath string) (Result, error) {
	layers, err := readLayers(workspaceRoot, userSettingsPath)
	if err != nil {
		return Result{}, err
	}

	var sources []string
	for _, layer := range layers {
		if len(layer.Values) > 0 {
			sources = append(sources, layer.Source)
		}
	}
	values := mergeLayers(layers)
	if _, configured := values[keyOutputs]; !configured {
		return Result{Sources: sources}, nil
	}

	settings, err := toSettings(values)
	if err != nil {
		return Result{}, err
	}
	settings, firstSync, err := workspacesettings.ResolveEditorSettings(settings)
	if err != nil {
		return Result{}, err
	}

	return Result{Configured: true, FirstSync: firstSync, Settings: settings, Sources: sources}, nil
}

// readLayers reads the settings layers in the order they are merged: the user's, the workspace file's, the folder's, then the overrides.
func readLayers(workspaceRoot, userSettingsPath string) ([]Layer, error) {
	var layers []Layer
	add := func(layer Layer, err error) error {
		if err != nil {
			return err
		}
		layers = append(layers, layer)

		return nil
	}
	if userSettingsPath != "" {
		if err := add(ReadLayer(userSettingsPath)); err != nil {
			return nil, err
		}
	}
	if workspaceFile := os.Getenv(WorkspaceFileVariable); workspaceFile != "" {
		if err := add(ReadWorkspaceFileLayer(workspaceFile)); err != nil {
			return nil, err
		}
	}
	if err := add(ReadLayer(FolderSettingsPath(workspaceRoot))); err != nil {
		return nil, err
	}
	if err := add(ParseOverridesLayer(os.Getenv(OverridesVariable))); err != nil {
		return nil, err
	}

	return layers, nil
}
