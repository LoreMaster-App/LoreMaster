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
	abstracts, instances := numberingDefinitions(lists, 0, 0)

	return xmlHeader + `<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` + abstracts + instances + "</w:numbering>"
}

// numberingDefinitions writes the abstract definitions and the numbering instances the lists use, with their ids moved up by
// the given amounts, so they can sit beside the ones a template already has without taking their ids. All the abstract
// definitions have to come before any instance in a numbering part, which is why they are two strings.
func numberingDefinitions(lists []orderedList, abstractOffset, instanceOffset int) (string, string) {
	bullets, numbers := abstractOffset, abstractOffset+1
	abstracts := abstractDefinition(bullets, func(level int) string {
		return `<w:numFmt w:val="bullet"/><w:lvlText w:val="` + bulletGlyphs[level%len(bulletGlyphs)] + `"/>`
	}) + abstractDefinition(numbers, func(level int) string {
		return `<w:numFmt w:val="decimal"/><w:lvlText w:val="%` + strconv.Itoa(level+1) + `."/>`
	})

	var instances strings.Builder
	instances.WriteString(`<w:num w:numId="` + strconv.Itoa(instanceOffset+bulletNumID) + `"><w:abstractNumId w:val="` + strconv.Itoa(bullets) + `"/></w:num>`)
	for position, list := range lists {
		instances.WriteString(`<w:num w:numId="` + strconv.Itoa(instanceOffset+orderedBase+position) + `"><w:abstractNumId w:val="` + strconv.Itoa(numbers) + `"/>` +
			`<w:lvlOverride w:ilvl="` + strconv.Itoa(list.level) + `"><w:startOverride w:val="` + strconv.Itoa(list.start) + `"/></w:lvlOverride></w:num>`)
	}

	return abstracts, instances.String()
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
