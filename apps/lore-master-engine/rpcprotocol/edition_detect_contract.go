package rpcprotocol

// MethodEditionDetect probes which edition a site runs, without a credential, so the
// editor can offer only the sign-in methods that edition accepts before asking for one.
const MethodEditionDetect = "edition/detect"

// EditionDetectParams names the site to probe.
type EditionDetectParams struct {
	// BaseURL is the site's address; any page URL of the site is accepted too.
	BaseURL string `json:"baseUrl"`
}

// EditionDetectResult is the detected edition. No session is opened and nothing is kept.
type EditionDetectResult struct {
	// BaseURL is the normalised site address.
	BaseURL string `json:"baseUrl"`
	// Edition is "cloud", "datacenter" or "server".
	Edition string `json:"edition"`
	// Version is the server version on Data Center and Server ("8.5.4"); empty on Cloud.
	Version string `json:"version,omitempty"`
}
