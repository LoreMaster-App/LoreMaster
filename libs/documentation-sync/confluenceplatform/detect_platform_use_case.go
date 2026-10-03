package confluenceplatform

import (
	"context"
	"log/slog"
	"net/http"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/editiondetection"
	"lore-master/libs/confluence-client/httptransport"
)

// DetectOptions is a site to probe for its edition, without signing in.
type DetectOptions struct {
	// BaseURL is the site's address; any page URL of the site is accepted.
	BaseURL string
	// HTTPClient and Logger default as in httptransport.
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// Detected is a probed site's edition.
type Detected struct {
	// BaseURL is normalised; Edition is "cloud", "datacenter" or "server"; Version is
	// the server version ("8.5.4") on Data Center and Server, empty on Cloud.
	BaseURL, Edition, Version string
}

// Detect reports a site's Confluence edition without a credential, so the editor can
// offer only the sign-in methods that edition takes. It normalises the address, then
// probes anonymously; it opens no session and keeps nothing. Failures are the same
// ConnectError values Connect uses, so a caller maps them once.
func Detect(ctx context.Context, options DetectOptions) (Detected, error) {
	baseURL, err := connection.NormalizeBaseURL(options.BaseURL)
	if err != nil {
		return Detected{}, &ConnectError{Failure: InvalidAddress, Err: err}
	}

	anonymous, err := httptransport.New(httptransport.Options{BaseURL: baseURL, HTTPClient: options.HTTPClient, Logger: options.Logger})
	if err != nil {
		return Detected{}, &ConnectError{Failure: InvalidAddress, Err: err}
	}
	detection, err := editiondetection.DetectEdition(ctx, anonymous, baseURL)
	if err != nil {
		return Detected{}, &ConnectError{Failure: Unreachable, Err: err}
	}

	detected := Detected{BaseURL: baseURL, Edition: string(detection.Edition)}
	if detection.Version.Known() {
		detected.Version = detection.Version.String()
	}

	return detected, nil
}
