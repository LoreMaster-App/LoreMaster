package odtexport

import (
	"regexp"
	"strings"
)

var (
	styleElements = regexp.MustCompile(`(?s)<style:style\b[^>]*/>|<style:style\b.*?</style:style>`)
	styleNameAttr = regexp.MustCompile(`style:name="([^"]*)"`)
)

// templateStyles puts the styles this package needs into a template's styles part, so the document takes the template's page,
// headers, footers and look. A named style the template already has is left as it is (that is how a heading takes the
// template's heading); one it lacks is added. The template's own styles are never replaced.
func templateStyles(template string) string {
	present := map[string]bool{}
	for _, element := range styleElements.FindAllString(template, -1) {
		if name := styleNameAttr.FindStringSubmatch(element); name != nil {
			present[name[1]] = true
		}
	}

	var added strings.Builder
	for _, element := range styleElements.FindAllString(namedStyles(), -1) {
		if name := styleNameAttr.FindStringSubmatch(element); name != nil && !present[name[1]] {
			added.WriteString(element)
		}
	}
	if added.Len() == 0 {
		return template
	}
	if end := strings.LastIndex(template, "</office:styles>"); end >= 0 {
		return template[:end] + added.String() + template[end:]
	}
	if open := strings.Index(template, "<office:styles/>"); open >= 0 {
		return template[:open] + "<office:styles>" + added.String() + "</office:styles>" + template[open+len("<office:styles/>"):]
	}

	return template
}
