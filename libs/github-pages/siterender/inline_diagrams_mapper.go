package siterender

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// openingFence matches the line that opens a fenced code block: up to three spaces, three or
// more backticks or tildes, then the info string.
var openingFence = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})\\s*([^`\\s]*)")

// InlineDiagrams replaces each fenced block of language (a diagram language the browser cannot
// draw by itself, such as "d2") with an image whose source is the drawing as a data URI, so the
// page needs no script for it. draw turns a block's source into an SVG; a block it fails on is
// left as the code it was and named in the notes. Everything else in markdown is untouched.
func InlineDiagrams(markdown []byte, language string, draw func(source string) ([]byte, error)) ([]byte, []string) {
	lines := strings.Split(string(markdown), "\n")
	var out []string
	var notes []string

	for i := 0; i < len(lines); i++ {
		match := openingFence.FindStringSubmatch(lines[i])
		if match == nil || !strings.EqualFold(match[3], language) {
			out = append(out, lines[i])

			continue
		}
		fence := match[2]
		end := -1
		for j := i + 1; j < len(lines); j++ {
			if closesFence(lines[j], fence) {
				end = j

				break
			}
		}
		if end < 0 {
			out = append(out, lines[i:]...)

			break
		}

		source := strings.Join(lines[i+1:end], "\n")
		svg, err := draw(source)
		if err != nil {
			notes = append(notes, fmt.Sprintf("a %s diagram on line %d could not be drawn (%v); it stays as code", strings.ToUpper(language), i+1, err))
			out = append(out, lines[i:end+1]...)
			i = end

			continue
		}
		out = append(out, fmt.Sprintf("![%s diagram](data:image/svg+xml;base64,%s)", strings.ToUpper(language), base64.StdEncoding.EncodeToString(bytes.TrimSpace(svg))))
		i = end
	}

	return []byte(strings.Join(out, "\n")), notes
}

// closesFence reports whether line closes a block opened by fence: the same character, at
// least as many times, and nothing else.
func closesFence(line string, fence string) bool {
	trimmed := strings.TrimRight(strings.TrimLeft(line, " "), " \r")
	if len(trimmed) < len(fence) || len(line)-len(strings.TrimLeft(line, " ")) > 3 {
		return false
	}

	return strings.Trim(trimmed, fence[:1]) == "" && len(trimmed) >= len(fence)
}
