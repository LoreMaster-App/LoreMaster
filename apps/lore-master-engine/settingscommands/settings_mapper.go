package settingscommands

import (
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/libs/documentation-sync/workspacesettings"
)

func toWire(settings workspacesettings.Settings) rpcprotocol.Settings {
	wire := rpcprotocol.Settings{Version: settings.Version, SkipGitignored: settings.SkipGitignored, Ignore: settings.Ignore, Generators: generatorsToWire(settings.Generators), Outputs: make([]rpcprotocol.Output, len(settings.Outputs))}
	for i, output := range settings.Outputs {
		wire.Outputs[i] = rpcprotocol.Output{
			Platform: output.Platform, BaseURL: output.BaseURL, Space: output.Space, ParentPageID: output.ParentPageID,
			TitlePrefix: output.TitlePrefix, Direction: output.Direction, MermaidMode: output.MermaidMode,
			TitleCollision: output.TitleCollision, LinkMode: output.LinkMode, Repo: output.Repo, Branch: output.Branch,
			Content: make([]rpcprotocol.Content, len(output.Content)),
		}
		for j, content := range output.Content {
			wire.Outputs[i].Content[j] = rpcprotocol.Content(content)
		}
	}

	return wire
}

func fromWire(wire rpcprotocol.Settings) workspacesettings.Settings {
	settings := workspacesettings.Settings{Version: wire.Version, SkipGitignored: wire.SkipGitignored, Ignore: wire.Ignore, Generators: generatorsFromWire(wire.Generators), Outputs: make([]workspacesettings.Output, len(wire.Outputs))}
	for i, output := range wire.Outputs {
		settings.Outputs[i] = workspacesettings.Output{
			Platform: output.Platform, BaseURL: output.BaseURL, Space: output.Space, ParentPageID: output.ParentPageID,
			TitlePrefix: output.TitlePrefix, Direction: output.Direction, MermaidMode: output.MermaidMode,
			TitleCollision: output.TitleCollision, LinkMode: output.LinkMode, Repo: output.Repo, Branch: output.Branch,
			Content: make([]workspacesettings.Content, len(output.Content)),
		}
		for j, content := range output.Content {
			settings.Outputs[i].Content[j] = workspacesettings.Content(content)
		}
	}

	return settings
}

func generatorsToWire(generators []workspacesettings.Generator) []rpcprotocol.Generator {
	if len(generators) == 0 {
		return nil
	}
	wire := make([]rpcprotocol.Generator, len(generators))
	for i, generator := range generators {
		wire[i] = rpcprotocol.Generator(generator)
	}

	return wire
}

func generatorsFromWire(wire []rpcprotocol.Generator) []workspacesettings.Generator {
	if len(wire) == 0 {
		return nil
	}
	generators := make([]workspacesettings.Generator, len(wire))
	for i, generator := range wire {
		generators[i] = workspacesettings.Generator(generator)
	}

	return generators
}
