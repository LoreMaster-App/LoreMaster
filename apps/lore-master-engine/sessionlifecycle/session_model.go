package sessionlifecycle

import "lore-master/libs/documentation-sync/confluenceplatform"

// Session is one open, verified connection.
type Session struct {
	ID string
	// BaseURL is normalised; Edition is "cloud", "datacenter" or "server".
	BaseURL, Edition string
	// Platform carries the credential inside its HTTP client.
	Platform *confluenceplatform.Platform
}
