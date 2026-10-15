package childrenmarker

import (
	"regexp"
	"strconv"
	"strings"
)

// Options say which descendants a marker lists.
type Options struct {
	// All lists every descendant, not only the direct children.
	All bool
	// Depth limits how many levels below the page are listed; 0 means the direct children
	// only (unless All).
	Depth int
}

// marker matches a whole line: [children], [children all], [children depth=2] and the two
// together, in any case, with any spacing.
var marker = regexp.MustCompile(`(?i)^\[\s*children((?:\s+(?:all|depth\s*=\s*\d+))*)\s*\]$`)

var depthOption = regexp.MustCompile(`(?i)depth\s*=\s*(\d+)`)

// Parse reports whether text is exactly a children marker, and its options. The text is a
// whole paragraph or line; a marker inside other text is not one.
func Parse(text string) (Options, bool) {
	match := marker.FindStringSubmatch(strings.TrimSpace(text))
	if match == nil {
		return Options{}, false
	}
	var options Options
	for _, word := range strings.Fields(strings.ToLower(match[1])) {
		if word == "all" {
			options.All = true
		}
	}
	if depth := depthOption.FindStringSubmatch(match[1]); depth != nil {
		options.Depth, _ = strconv.Atoi(depth[1])
	}

	return options, true
}

// String is the marker as it is written in Markdown; Parse reads it back.
func (options Options) String() string {
	text := "[children"
	if options.All {
		text += " all"
	}
	if options.Depth > 0 {
		text += " depth=" + strconv.Itoa(options.Depth)
	}

	return text + "]"
}

// Levels is how many levels below the page the marker lists: 1 for the direct children, more
// for a depth, and 0 meaning every level.
func (options Options) Levels() int {
	switch {
	case options.All:
		return 0
	case options.Depth > 0:
		return options.Depth
	default:
		return 1
	}
}
