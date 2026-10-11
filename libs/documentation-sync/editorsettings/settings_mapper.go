package editorsettings

import (
	"bytes"
	"encoding/json"
	"fmt"

	"lore-master/libs/documentation-sync/workspacesettings"
)

type outputWire struct {
	Platform       string        `json:"platform,omitempty"`
	BaseURL        string        `json:"baseUrl,omitempty"`
	Space          string        `json:"space,omitempty"`
	ParentPageID   string        `json:"parentPageId,omitempty"`
	TitlePrefix    string        `json:"titlePrefix,omitempty"`
	Direction      string        `json:"direction,omitempty"`
	Content        []contentWire `json:"content,omitempty"`
	MermaidMode    string        `json:"mermaidMode,omitempty"`
	TitleCollision string        `json:"titleCollision,omitempty"`
	LinkMode       string        `json:"linkMode,omitempty"`
	Repo           string        `json:"repo,omitempty"`
	Branch         string        `json:"branch,omitempty"`
	Path           string        `json:"path,omitempty"`
	Include        []string      `json:"include,omitempty"`
	Exclude        []string      `json:"exclude,omitempty"`
}

type contentWire struct {
	Type     string   `json:"type,omitempty"`
	Roots    []string `json:"roots,omitempty"`
	Excludes []string `json:"excludes,omitempty"`
	Template string   `json:"template,omitempty"`
}

type generatorWire struct {
	Type   string   `json:"type,omitempty"`
	Input  []string `json:"input,omitempty"`
	Output string   `json:"output,omitempty"`
	Title  string   `json:"title,omitempty"`
}

// toSettings turns the merged keys into settings. The user-level defaults
// (loreMaster.confluence.baseUrl and loreMaster.defaults.*) fill an output's empty
// fields, so a base URL set once in the user's settings reaches every workspace. It
// does not validate; workspacesettings.ResolveSettings does, as for a file.
func toSettings(values map[string]merged) (workspacesettings.Settings, error) {
	settings := workspacesettings.Settings{Version: workspacesettings.CurrentVersion}

	var wires []outputWire
	if err := decodeKey(values, keyOutputs, &wires); err != nil {
		return settings, err
	}
	var generators []generatorWire
	if err := decodeKey(values, keyGenerators, &generators); err != nil {
		return settings, err
	}
	if err := decodeKey(values, keyIgnore, &settings.Ignore); err != nil {
		return settings, err
	}
	if err := decodeKey(values, keySkipGitignored, &settings.SkipGitignored); err != nil {
		return settings, err
	}
	var confluenceURL string
	if err := decodeKey(values, keyConfluenceURL, &confluenceURL); err != nil {
		return settings, err
	}
	defaults := map[string]*string{}
	for _, key := range []string{keyDefaultsDirect, keyDefaultsMermaid, keyDefaultsLink, keyDefaultsCollide} {
		var value string
		if err := decodeKey(values, key, &value); err != nil {
			return settings, err
		}
		defaults[key] = &value
	}

	for _, wire := range wires {
		output := workspacesettings.Output{
			Platform: wire.Platform, BaseURL: wire.BaseURL, Space: wire.Space, ParentPageID: wire.ParentPageID,
			TitlePrefix: wire.TitlePrefix, Direction: wire.Direction, MermaidMode: wire.MermaidMode,
			TitleCollision: wire.TitleCollision, LinkMode: wire.LinkMode, Repo: wire.Repo, Branch: wire.Branch,
			Path: wire.Path, Include: wire.Include, Exclude: wire.Exclude,
		}
		for _, content := range wire.Content {
			output.Content = append(output.Content, workspacesettings.Content{
				Type: content.Type, Roots: content.Roots, Excludes: content.Excludes, Template: content.Template,
			})
		}
		if output.Platform == "" || output.Platform == "confluence" {
			fillEmpty(&output.BaseURL, confluenceURL)
		}
		fillEmpty(&output.Direction, *defaults[keyDefaultsDirect])
		fillEmpty(&output.MermaidMode, *defaults[keyDefaultsMermaid])
		fillEmpty(&output.LinkMode, *defaults[keyDefaultsLink])
		fillEmpty(&output.TitleCollision, *defaults[keyDefaultsCollide])
		settings.Outputs = append(settings.Outputs, output)
	}
	for _, generator := range generators {
		settings.Generators = append(settings.Generators, workspacesettings.Generator{
			Type: generator.Type, Input: generator.Input, Output: generator.Output, Title: generator.Title,
		})
	}

	return settings, nil
}

func fillEmpty(field *string, fallback string) {
	if *field == "" {
		*field = fallback
	}
}

// decodeKey decodes one key strictly, so a misspelt field names itself instead of being
// ignored. An absent key leaves target as it was.
func decodeKey(values map[string]merged, key string, target any) error {
	entry, present := values[key]
	if !present {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(entry.value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%s: %s: %w", entry.source, key, err)
	}

	return nil
}
