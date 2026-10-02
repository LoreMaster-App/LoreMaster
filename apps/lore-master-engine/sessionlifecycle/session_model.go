package sessionlifecycle

import (
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
}

// confluence adapts the Confluence platform to Platform.
type confluence struct{ *confluenceplatform.Platform }

func (c confluence) ForOutput(output workspacesettings.Output) platformport.DocumentationPlatform {
	return c.Platform.ForOutput(output)
}
