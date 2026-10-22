package storageformat

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	svgProlog     = regexp.MustCompile(`(?s)^\s*(<\?xml.*?\?>|<!DOCTYPE.*?>|<!--.*?-->|\s)*`)
	svgRootTag    = regexp.MustCompile(`(?s)<svg\b[^>]*>`)
	svgSizeAttrs  = regexp.MustCompile(`\s(width|height|style)="[^"]*"`)
	svgViewBox    = regexp.MustCompile(`viewBox="\s*[-\d.]+[\s,]+[-\d.]+[\s,]+([\d.]+)[\s,]+[\d.]+\s*"`)
	svgPlainWidth = regexp.MustCompile(`\swidth="([\d.]+)(?:px)?"`)
)

// sizedSVG is a drawn diagram as the HTML an HTML macro holds: the SVG shown at most maxWidth
// pixels wide (0: at the width it was drawn), scaling down to fit a narrower page and keeping its
// proportions. Mermaid sizes its pictures to fill their container, which makes a wide diagram
// enormous in a page; this gives the container the width instead.
func sizedSVG(svg string, maxWidth int) string {
	svg = svgProlog.ReplaceAllString(svg, "")
	root := svgRootTag.FindString(svg)
	if root == "" {
		return svg
	}
	natural := naturalWidth(root)
	width := natural
	if maxWidth > 0 && (natural == 0 || maxWidth < natural) {
		width = maxWidth
	}
	resized := strings.TrimSuffix(svgSizeAttrs.ReplaceAllString(root, ""), ">") + ` width="100%" style="height:auto;"` + ">"
	svg = strings.Replace(svg, root, resized, 1)
	if width == 0 {
		return svg
	}

	return fmt.Sprintf(`<div style="max-width:%dpx;width:100%%;">%s</div>`, width, svg)
}

// naturalWidth is the width the diagram was drawn at, in pixels, or 0 when it does not say.
func naturalWidth(root string) int {
	for _, pattern := range []*regexp.Regexp{svgViewBox, svgPlainWidth} {
		if match := pattern.FindStringSubmatch(root); match != nil {
			if value, err := strconv.ParseFloat(match[1], 64); err == nil && value > 0 {
				return int(value + 0.5)
			}
		}
	}

	return 0
}
