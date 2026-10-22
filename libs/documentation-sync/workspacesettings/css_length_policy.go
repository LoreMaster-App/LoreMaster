package workspacesettings

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// DefaultMermaidMaxWidth is how wide an inline diagram is shown unless an output says otherwise.
const DefaultMermaidMaxWidth = "800px"

var (
	bareNumber = regexp.MustCompile(`^\d+(\.\d+)?$`)
	// cssLengthText is what a CSS max-width value is made of: words, numbers, units, %, the
	// operators and brackets of min(), max(), clamp() and calc(), and spaces. Nothing that could end
	// the style attribute it is put in or start another property.
	cssLengthText = regexp.MustCompile(`^[a-zA-Z0-9 .,%()+*/-]+$`)
	forbiddenCSS  = regexp.MustCompile(`(?i)url\(|expression|javascript|@import|var\(|attr\(`)
)

// NormalizeCSSLength is value as a CSS length: a bare number is pixels, anything else is trimmed
// and kept as written.
func NormalizeCSSLength(value string) string {
	value = strings.TrimSpace(value)
	if bareNumber.MatchString(value) {
		return value + "px"
	}

	return value
}

// CSSLengthProblem says what is wrong with value as a CSS max-width ("" when nothing is): any
// length, percentage or min()/max()/clamp()/calc() of them is accepted, since it is the page's CSS
// that reads it.
func CSSLengthProblem(value string) string {
	value = NormalizeCSSLength(value)
	switch {
	case value == "":
		return ""
	case len(value) > 100:
		return "is too long"
	case !cssLengthText.MatchString(value) || forbiddenCSS.MatchString(value):
		return "must be a CSS length such as 800px, 50% or min(800px, 100%)"
	case strings.Count(value, "(") != strings.Count(value, ")"):
		return "has unbalanced brackets"
	}

	return ""
}

// CSSLength is a CSS length in a settings file, written as text ("50%", "600px") or, as the
// setting first was, as a number of pixels.
type CSSLength string

// UnmarshalJSON accepts text or a number.
func (l *CSSLength) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*l = CSSLength(text)

		return nil
	}
	var pixels float64
	if err := json.Unmarshal(data, &pixels); err != nil {
		return fmt.Errorf("a CSS length is text such as \"50%%\" or a number of pixels")
	}
	*l = CSSLength(NormalizeCSSLength(fmt.Sprint(pixels)))

	return nil
}
