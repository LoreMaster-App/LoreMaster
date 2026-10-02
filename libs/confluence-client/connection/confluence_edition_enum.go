package connection

import "fmt"

// Edition is the kind of Confluence deployment. The editions differ in API version
// (Cloud has REST v2), in authentication, and in what a site's URL looks like.
type Edition string

// The three editions. Server reached end of life in February 2024 but is still run.
const (
	Cloud      Edition = "cloud"
	DataCenter Edition = "datacenter"
	Server     Edition = "server"
)

// ParseEdition accepts the values above, as stored in settings.
func ParseEdition(value string) (Edition, error) {
	switch edition := Edition(value); edition {
	case Cloud, DataCenter, Server:
		return edition, nil
	}

	return "", fmt.Errorf("unknown Confluence edition %q; expected cloud, datacenter or server", value)
}
