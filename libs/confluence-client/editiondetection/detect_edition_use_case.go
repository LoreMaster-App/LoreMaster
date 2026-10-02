package editiondetection

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/httptransport"
)

// Detection is what DetectEdition found.
type Detection struct {
	Edition connection.Edition
	// Version is zero on Cloud.
	Version connection.Version
}

// cloudVersionsFrom is where Cloud's internal version numbers start ("1000.0.0-…"), in
// case a Cloud site behind a custom domain answers the manifest instead of systemInfo.
const cloudVersionsFrom = 1000

// lastServerMajor is the last Server release line (7.19); 8.0 onwards exists only as
// Data Center. Before 8.0 the two are the same software and REST cannot tell them
// apart, which does not matter here: both take the same APIs and credentials.
const lastServerMajor = 7

// manifest is the application-links manifest every Data Center and Server site serves,
// without authentication, at /rest/applinks/1.0/manifest.
type manifest struct {
	TypeID      string `xml:"typeId"`
	Version     string `xml:"version"`
	BuildNumber string `xml:"buildNumber"`
}

// DetectEdition probes the site at baseURL (normalised) through client:
//
//  1. A *.atlassian.net host is Cloud, without a request.
//  2. GET /rest/api/settings/systemInfo answers only on Cloud: 200 means a Cloud site
//     behind a custom domain.
//  3. GET /rest/applinks/1.0/manifest gives a Data Center or Server site's version:
//     8.0 or later is Data Center, earlier is reported as Server.
//
// A site that answers neither probe is not recognised, and the error asks the user to
// pick the edition.
func DetectEdition(ctx context.Context, client *httptransport.Client, baseURL string) (Detection, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return Detection{}, err
	}
	if connection.IsCloudHost(parsed.Host) {
		return Detection{Edition: connection.Cloud}, nil
	}

	var systemInfo map[string]any
	err = client.GetJSON(ctx, "/rest/api/settings/systemInfo", nil, &systemInfo)
	if err == nil {
		return Detection{Edition: connection.Cloud}, nil
	}
	if ctx.Err() != nil {
		return Detection{}, ctx.Err()
	}

	body, err := client.GetBytes(ctx, "/rest/applinks/1.0/manifest", nil, "application/xml")
	if err != nil {
		var apiError *httptransport.APIError
		if errors.As(err, &apiError) && apiError.Status == http.StatusNotFound {
			return Detection{}, fmt.Errorf("%s does not look like a Confluence site (no system information and no application-links manifest); check the address, or choose the edition by hand", baseURL)
		}

		return Detection{}, fmt.Errorf("could not detect the Confluence edition of %s: %w", baseURL, err)
	}
	var found manifest
	if err := xml.Unmarshal(body, &found); err != nil || found.TypeID != "confluence" {
		return Detection{}, fmt.Errorf("%s is not a Confluence site (its application-links manifest describes %q); check the address", baseURL, found.TypeID)
	}
	version, err := connection.ParseVersion(found.Version)
	if err != nil {
		return Detection{}, fmt.Errorf("%s reports an unreadable Confluence version: %w", baseURL, err)
	}
	switch {
	case version.Major >= cloudVersionsFrom:
		return Detection{Edition: connection.Cloud}, nil
	case version.Major > lastServerMajor:
		return Detection{Edition: connection.DataCenter, Version: version}, nil
	default:
		return Detection{Edition: connection.Server, Version: version}, nil
	}
}
