package connection

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// NormalizeBaseURL turns what a user pastes into the site's base URL. It trims trailing
// slashes, lowercases the host, and on Atlassian Cloud reduces any page URL to the
// site's "/wiki" root, which is where every Cloud API lives. Data Center and Server keep
// their path: it is the context path ("/confluence") under which the APIs live. Only
// https is accepted, except for a loopback host used in development, and a URL may not
// carry credentials, a query or a fragment.
func NormalizeBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", fmt.Errorf("%q is not a Confluence address; expected something like https://example.atlassian.net", raw)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("the Confluence address must not contain a user name or password")
	}
	if parsed.Scheme == "http" && !isLoopback(parsed.Hostname()) {
		return "", fmt.Errorf("the Confluence address must use https; plain http is accepted only for localhost")
	}

	host := strings.ToLower(parsed.Host)
	path := strings.TrimRight(parsed.Path, "/")
	if IsCloudHost(host) {
		path = "/wiki"
	}

	return parsed.Scheme + "://" + host + path, nil
}

// IsCloudHost reports whether host (with or without a port) is an Atlassian Cloud site.
// A Cloud site behind a custom domain is not recognised here; edition detection probes
// for it.
func IsCloudHost(host string) bool {
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}

	return strings.HasSuffix(strings.ToLower(hostname), ".atlassian.net")
}

func isLoopback(hostname string) bool {
	if hostname == "localhost" {
		return true
	}
	ip := net.ParseIP(hostname)

	return ip != nil && ip.IsLoopback()
}
