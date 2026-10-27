package syncexecution

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	svgImageRoot = regexp.MustCompile(`(?s)<svg\b[^>]*>`)
	svgImageSize = regexp.MustCompile(`\s(width|height)="[^"]*"`)
	svgImageView = regexp.MustCompile(`viewBox="\s*[-\d.]+[\s,]+[-\d.]+[\s,]+([\d.]+)[\s,]+([\d.]+)\s*"`)
)

// pixelSizedSVG gives a drawn SVG the pixel width and height of its viewBox. Mermaid writes
// width="100%" and no height, which an <img> (how Confluence shows an attached picture) cannot
// resolve: it falls back to a tiny default box. SVG without a usable viewBox is returned unchanged.
func pixelSizedSVG(svg []byte) []byte {
	root := svgImageRoot.Find(svg)
	view := svgImageView.FindSubmatch(root)
	if root == nil || view == nil {
		return svg
	}
	width, errW := strconv.ParseFloat(string(view[1]), 64)
	height, errH := strconv.ParseFloat(string(view[2]), 64)
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return svg
	}
	tag := strings.TrimSuffix(string(svgImageSize.ReplaceAll(root, nil)), ">")
	tag += fmt.Sprintf(` width="%d" height="%d">`, int(width+0.5), int(height+0.5))

	return bytes.Replace(svg, root, []byte(tag), 1)
}
