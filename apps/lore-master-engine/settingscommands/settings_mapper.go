package settingscommands

import (
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/libs/documentation-sync/workspacesettings"
)

func toWire(settings workspacesettings.Settings) rpcprotocol.Settings {
	wire := rpcprotocol.Settings{Version: settings.Version, Outputs: make([]rpcprotocol.Output, len(settings.Outputs))}
	for i, output := range settings.Outputs {
		wire.Outputs[i] = rpcprotocol.Output{
			Platform: output.Platform, BaseURL: output.BaseURL, Space: output.Space, ParentPageID: output.ParentPageID,
			TitlePrefix: output.TitlePrefix, Direction: output.Direction, MermaidMode: output.MermaidMode,
			TitleCollision: output.TitleCollision, LinkMode: output.LinkMode,
			Content: make([]rpcprotocol.Content, len(output.Content)),
		}
		for j, content := range output.Content {
			wire.Outputs[i].Content[j] = rpcprotocol.Content(content)
		}
	}

	return wire
}

func fromWire(wire rpcprotocol.Settings) workspacesettings.Settings {
	settings := workspacesettings.Settings{Version: wire.Version, Outputs: make([]workspacesettings.Output, len(wire.Outputs))}
	for i, output := range wire.Outputs {
		settings.Outputs[i] = workspacesettings.Output{
			Platform: output.Platform, BaseURL: output.BaseURL, Space: output.Space, ParentPageID: output.ParentPageID,
			TitlePrefix: output.TitlePrefix, Direction: output.Direction, MermaidMode: output.MermaidMode,
			TitleCollision: output.TitleCollision, LinkMode: output.LinkMode,
			Content: make([]workspacesettings.Content, len(output.Content)),
		}
		for j, content := range output.Content {
			settings.Outputs[i].Content[j] = workspacesettings.Content(content)
		}
	}

	return settings
}
