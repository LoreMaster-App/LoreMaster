package editorsettings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"lore-master/libs/documentation-sync/workspacesettings"
)

// builtinDefaults are what an output gets when it says nothing; writing them out would
// only make settings.json noisy.
var builtinDefaults = map[string]string{
	keyDefaultsDirect:  "to-platform",
	keyDefaultsMermaid: "image",
	keyDefaultsLink:    "title",
	keyDefaultsCollide: "fail",
}

// SaveFolderSettings writes configuration into the workspace folder's
// .vscode/settings.json (#308). Only the loreMaster.outputs, ignore, skipGitignored and
// generators keys are touched: every comment and every other setting stays where it was.
// Values an output would inherit anyway (the user's base URL, a default) are left out, so
// a base URL set once in User settings is not copied into the folder. The settings are
// validated first, and the file is written through a temporary file and a rename.
func SaveFolderSettings(workspaceRoot string, settings workspacesettings.Settings) error {
	if err := workspacesettings.Validate(settings); err != nil {
		return err
	}
	inherited, err := inheritedDefaults(workspaceRoot)
	if err != nil {
		return err
	}

	path := FolderSettingsPath(workspaceRoot)
	content, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	outputs, err := json.Marshal(minimalOutputs(settings.Outputs, inherited))
	if err != nil {
		return err
	}
	updates := []struct {
		key   string
		value any
		keep  bool
	}{
		{keyIgnore, settings.Ignore, len(settings.Ignore) > 0},
		{keySkipGitignored, settings.SkipGitignored, settings.SkipGitignored != nil},
		{keyGenerators, generatorsToWire(settings.Generators), len(settings.Generators) > 0},
	}
	if content, err = setKey(content, keyOutputs, outputs); err != nil {
		return err
	}
	for _, update := range updates {
		var value []byte
		if update.keep {
			if value, err = json.Marshal(update.value); err != nil {
				return err
			}
		}
		if content, err = setKey(content, update.key, value); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, content, 0o644); err != nil {
		return err
	}

	return os.Rename(temporary, path)
}

// inheritedDefaults is the user-level default for each key a folder's output would
// otherwise repeat: the Confluence base URL and the output enums. The folder's own
// values count too, as they are what an output inherits in this folder.
func inheritedDefaults(workspaceRoot string) (map[string]string, error) {
	var layers []Layer
	for _, path := range []string{CurrentUserSettingsPath(), FolderSettingsPath(workspaceRoot)} {
		if path == "" {
			continue
		}
		layer, err := ReadLayer(path)
		if err != nil {
			return nil, err
		}
		layers = append(layers, layer)
	}
	merged := mergeLayers(layers)
	result := map[string]string{}
	for _, key := range []string{keyConfluenceURL, keyDefaultsDirect, keyDefaultsMermaid, keyDefaultsLink, keyDefaultsCollide} {
		var value string
		if err := decodeKey(merged, key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}

	return result, nil
}

func minimalOutputs(outputs []workspacesettings.Output, inherited map[string]string) []outputWire {
	wires := make([]outputWire, 0, len(outputs))
	for _, output := range outputs {
		wire := outputWire{
			Platform: output.Platform, Space: output.Space, ParentPageID: output.ParentPageID, TitlePrefix: output.TitlePrefix,
			Repo: output.Repo, Branch: output.Branch, Path: output.Path, Include: output.Include, Exclude: output.Exclude,
			Direction:      unlessEffective(output.Direction, inherited[keyDefaultsDirect], builtinDefaults[keyDefaultsDirect]),
			MermaidMode:    unlessEffective(output.MermaidMode, inherited[keyDefaultsMermaid], builtinDefaults[keyDefaultsMermaid]),
			LinkMode:       unlessEffective(output.LinkMode, inherited[keyDefaultsLink], builtinDefaults[keyDefaultsLink]),
			TitleCollision: unlessEffective(output.TitleCollision, inherited[keyDefaultsCollide], builtinDefaults[keyDefaultsCollide]),
			BaseURL:        output.BaseURL,
		}
		if output.Platform == "confluence" || output.Platform == "" {
			wire.BaseURL = unlessEffective(output.BaseURL, inherited[keyConfluenceURL], "")
		}
		if !isDefaultContent(output.Content) {
			for _, content := range output.Content {
				wire.Content = append(wire.Content, contentWire{Type: content.Type, Roots: content.Roots, Excludes: content.Excludes, Template: content.Template})
			}
		}
		wires = append(wires, wire)
	}

	return wires
}

// unlessEffective is "" when value is what the output gets without saying so: the
// inherited default when one is set, the built-in one otherwise. An explicit value that
// differs from what would be inherited is kept.
func unlessEffective(value, inherited, builtin string) string {
	effective := inherited
	if effective == "" {
		effective = builtin
	}
	if value == effective {
		return ""
	}

	return value
}

// isDefaultContent is whether the content list is the one an output gets when it has none:
// the whole workspace as Markdown with the default template.
func isDefaultContent(content []workspacesettings.Content) bool {
	if len(content) != 1 {
		return len(content) == 0
	}
	c := content[0]

	return (c.Type == "" || c.Type == "markdown") && (c.Template == "" || c.Template == "default") &&
		len(c.Excludes) == 0 && (len(c.Roots) == 0 || slices.Equal(c.Roots, []string{"."}))
}

func generatorsToWire(generators []workspacesettings.Generator) []generatorWire {
	wires := make([]generatorWire, len(generators))
	for i, generator := range generators {
		wires[i] = generatorWire{Type: generator.Type, Input: generator.Input, Output: generator.Output, Title: generator.Title}
	}

	return wires
}
