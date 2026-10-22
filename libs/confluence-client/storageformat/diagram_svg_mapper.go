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

// sizedSVG is a drawn diagram as the HTML an HTML macro holds: the SVG shown at most maxWidth (any
// CSS max-width value, such as 600px or 50%) and never wider than it was drawn, scaling down to fit
// a narrower page and keeping its proportions. Mermaid sizes its pictures to fill their container,
// which makes a wide diagram enormous in a page; this gives the container the width instead.
func sizedSVG(svg string, maxWidth string) string {
	svg = svgProlog.ReplaceAllString(svg, "")
	root := svgRootTag.FindString(svg)
	if root == "" {
		return svg
	}
	width := widthLimit(maxWidth, naturalWidth(root))
	resized := strings.TrimSuffix(svgSizeAttrs.ReplaceAllString(root, ""), ">") + ` width="100%" style="height:auto;"` + ">"
	svg = strings.Replace(svg, root, resized, 1)
	if width == "" {
		return svg
	}

	return fmt.Sprintf(`<div style="max-width:%s;width:100%%;">%s</div>`, width, svg)
}

// widthLimit is the CSS max-width for a diagram drawn natural pixels wide ("" when nothing limits it).
func widthLimit(maxWidth string, natural int) string {
	maxWidth = strings.TrimSpace(maxWidth)
	switch {
	case maxWidth == "" || strings.EqualFold(maxWidth, "none"):
		if natural == 0 {
			return ""
		}

		return fmt.Sprintf("%dpx", natural)
	case natural == 0:
		return maxWidth
	}

	return fmt.Sprintf("min(%s, %dpx)", maxWidth, natural)
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
