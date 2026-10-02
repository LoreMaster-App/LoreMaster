package workspacesettings

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// secretLikeKeys are keys that would put a secret in a committed file.
var secretLikeKeys = []string{"token", "apitoken", "password", "pat", "secret", "apikey", "credential", "credentials"}

// LoadSettings reads the workspace's .lore-master.yaml. A missing file is not an
// error: it yields the defaults with FirstSync set, so the editor asks the first-sync
// questions. The file is decoded strictly, so a misspelt key is an error naming its
// line rather than a setting silently ignored. Empty fields take their defaults, and
// the result is validated.
func LoadSettings(workspaceRoot string) (Loaded, error) {
	filePath := filepath.Join(workspaceRoot, FileName)
	content, err := os.ReadFile(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return Loaded{Settings: Settings{Version: CurrentVersion, Outputs: []Output{defaultOutput()}}, FirstSync: true, path: filePath}, nil
	}
	if err != nil {
		return Loaded{}, err
	}

	var tree yaml.Node
	if err := yaml.Unmarshal(content, &tree); err != nil {
		return Loaded{}, fmt.Errorf("%s: %w", FileName, err)
	}
	if key := findSecretKey(&tree); key != nil {
		return Loaded{}, fmt.Errorf("%s line %d: %q looks like a secret; secrets never go in this committed file, the editor keeps them in its secret store", FileName, key.Line, key.Value)
	}
	var settings Settings
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&settings); err != nil {
		return Loaded{}, fmt.Errorf("%s: %w", FileName, err)
	}
	applyDefaults(&settings)
	if err := Validate(settings); err != nil {
		return Loaded{}, fmt.Errorf("%s:\n%w", FileName, err)
	}

	return Loaded{Settings: settings, FirstSync: needsFirstSync(settings), Exists: true, path: filePath, tree: content}, nil
}

func applyDefaults(settings *Settings) {
	if settings.Version == 0 {
		settings.Version = CurrentVersion
	}
	defaults := defaultOutput()
	for i := range settings.Outputs {
		output := &settings.Outputs[i]
		output.TitlePrefix = strings.TrimSpace(output.TitlePrefix)
		output.Platform = cmpOr(output.Platform, defaults.Platform)
		output.Direction = cmpOr(output.Direction, defaults.Direction)
		output.MermaidMode = cmpOr(output.MermaidMode, defaults.MermaidMode)
		output.TitleCollision = cmpOr(output.TitleCollision, defaults.TitleCollision)
		output.LinkMode = cmpOr(output.LinkMode, defaults.LinkMode)
		if len(output.Content) == 0 {
			output.Content = defaults.Content
		}
		for j := range output.Content {
			content := &output.Content[j]
			content.Type = cmpOr(content.Type, "markdown")
			content.Template = cmpOr(content.Template, "default")
			if len(content.Roots) == 0 {
				content.Roots = []string{"."}
			}
		}
	}
}

func needsFirstSync(settings Settings) bool {
	for _, output := range settings.Outputs {
		if output.BaseURL == "" || output.Space == "" || output.ParentPageID == "" || output.TitlePrefix == "" {
			return true
		}
	}

	return false
}

// findSecretKey walks every mapping key in the document.
func findSecretKey(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			normalised := strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(key.Value))
			for _, secret := range secretLikeKeys {
				if normalised == secret {
					return key
				}
			}
		}
	}
	for _, child := range node.Content {
		if found := findSecretKey(child); found != nil {
			return found
		}
	}

	return nil
}

func cmpOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
