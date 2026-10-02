package connection

import (
	"fmt"
	"strconv"
	"strings"
)

// Connection is one Confluence site as the client sees it.
type Connection struct {
	// BaseURL is normalised by NormalizeBaseURL: no trailing slash, "/wiki" on Cloud.
	BaseURL string
	Edition Edition
	// Version is the server version on Data Center and Server; zero when unknown, and
	// always zero on Cloud, which has no version.
	Version Version
}

// Version is a Confluence server version, compared on major and minor only.
type Version struct {
	Major, Minor, Patch int
}

// ParseVersion reads "7.4", "8.5.3" or "7.19.16-rc1" (anything after the numbers is
// ignored).
func ParseVersion(value string) (Version, error) {
	numbers := strings.FieldsFunc(value, func(r rune) bool { return r == '.' || r == '-' })
	var parts [3]int
	for i := 0; i < len(numbers) && i < 3; i++ {
		n, err := strconv.Atoi(numbers[i])
		if err != nil {
			if i < 2 {
				return Version{}, fmt.Errorf("%q is not a Confluence version", value)
			}

			break
		}
		parts[i] = n
	}
	if len(numbers) < 2 {
		return Version{}, fmt.Errorf("%q is not a Confluence version", value)
	}

	return Version{Major: parts[0], Minor: parts[1], Patch: parts[2]}, nil
}

// Known reports whether the version was detected.
func (v Version) Known() bool {
	return v != Version{}
}

// AtLeast compares major and minor.
func (v Version) AtLeast(major int, minor int) bool {
	return v.Major > major || (v.Major == major && v.Minor >= minor)
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
