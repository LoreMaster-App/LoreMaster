package workspacesettings

import "strings"

// NotSetUp is what a workspace that has no configuration yet reads as: the defaults, with FirstSync set so the editor asks
// the first-sync questions, and Exists false.
func NotSetUp() Loaded {
	return Loaded{Settings: Settings{Version: CurrentVersion, Outputs: []Output{defaultOutput()}}, FirstSync: true}
}

// applyDefaults fills what a settings entry leaves empty: the version, each output's platform, direction and modes, and the
// whole workspace as an output's content. A title prefix is trimmed.
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
		if output.MermaidMaxWidth == 0 {
			output.MermaidMaxWidth = defaults.MermaidMaxWidth
		}
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

// needsFirstSync is true while an output lacks its site, space, parent page or title prefix.
func needsFirstSync(settings Settings) bool {
	for _, output := range settings.Outputs {
		if output.BaseURL == "" || output.Space == "" || output.ParentPageID == "" || output.TitlePrefix == "" {
			return true
		}
	}

	return false
}

func cmpOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
