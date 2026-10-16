package docxexport

import (
	"strconv"
	"strings"
)

// bulletGlyphs are the marks the nine bullet levels cycle through.
var bulletGlyphs = []string{"•", "◦", "▪"}

const numberingLevels = 9

// numberingPart writes numbering.xml: abstract definition 0 for bullets and 1 for numbers, nine
// levels each, then the numbering instance every list refers to. Each ordered list gets an
// instance of its own, so a list restarts at its own first number instead of continuing the one
// before it.
func numberingPart(lists []orderedList) string {
	var out strings.Builder
	out.WriteString(xmlHeader + `<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	out.WriteString(abstractDefinition(0, func(level int) string {
		return `<w:numFmt w:val="bullet"/><w:lvlText w:val="` + bulletGlyphs[level%len(bulletGlyphs)] + `"/>`
	}))
	out.WriteString(abstractDefinition(1, func(level int) string {
		return `<w:numFmt w:val="decimal"/><w:lvlText w:val="%` + strconv.Itoa(level+1) + `."/>`
	}))
	out.WriteString(`<w:num w:numId="` + strconv.Itoa(bulletNumID) + `"><w:abstractNumId w:val="0"/></w:num>`)
	for position, list := range lists {
		out.WriteString(`<w:num w:numId="` + strconv.Itoa(orderedBase+position) + `"><w:abstractNumId w:val="1"/>` +
			`<w:lvlOverride w:ilvl="` + strconv.Itoa(list.level) + `"><w:startOverride w:val="` + strconv.Itoa(list.start) + `"/></w:lvlOverride></w:num>`)
	}
	out.WriteString("</w:numbering>")

	return out.String()
}

func abstractDefinition(id int, format func(level int) string) string {
	var out strings.Builder
	out.WriteString(`<w:abstractNum w:abstractNumId="` + strconv.Itoa(id) + `"><w:multiLevelType w:val="hybridMultilevel"/>`)
	for level := range numberingLevels {
		out.WriteString(`<w:lvl w:ilvl="` + strconv.Itoa(level) + `"><w:start w:val="1"/>` + format(level) +
			`<w:lvlJc w:val="left"/><w:pPr><w:ind w:left="` + strconv.Itoa(720*(level+1)) + `" w:hanging="360"/></w:pPr></w:lvl>`)
	}
	out.WriteString("</w:abstractNum>")

	return out.String()
}
