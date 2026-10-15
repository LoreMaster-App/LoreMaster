package sessionlifecycle

import (
	"context"

	"lore-master/libs/documentation-sync/confluenceplatform"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// Session is one open, verified connection.
type Session struct {
	ID string
	// BaseURL is normalised; Edition is "cloud", "datacenter" or "server".
	BaseURL, Edition string
	// Platform carries the credential inside its HTTP client.
	Platform Platform
}

// Platform is a documentation platform as the engine uses it: the sync's port, plus the
// same connection set up for one output's rendering choices.
type Platform interface {
	platformport.DocumentationPlatform
	ForOutput(output workspacesettings.Output) platformport.DocumentationPlatform
	// ResolveTemplate reads the page layout an output names; the zero Template is none.
	ResolveTemplate(ctx context.Context, workspaceRoot string, value string) (confluenceplatform.Template, error)
	// ForOutputWithTemplate is ForOutput with every page put in the template.
	ForOutputWithTemplate(output workspacesettings.Output, template confluenceplatform.Template) platformport.DocumentationPlatform
}

// confluence adapts the Confluence platform to Platform.
type confluence struct{ *confluenceplatform.Platform }

func (c confluence) ForOutputWithTemplate(output workspacesettings.Output, template confluenceplatform.Template) platformport.DocumentationPlatform {
	return c.Platform.ForOutput(output).WithTemplate(template)
}

func (c confluence) ForOutput(output workspacesettings.Output) platformport.DocumentationPlatform {
	return c.Platform.ForOutput(output)
}
