package odtexport

import (
	"strconv"
	"strings"
)

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

// mimetype is the package's first entry, stored uncompressed, which is how a reader recognises it.
const mimetype = "application/vnd.oasis.opendocument.text"

// namespaces are declared on every part's root element.
const namespaces = `xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
	`xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" ` +
	`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" ` +
	`xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" ` +
	`xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" ` +
	`xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" ` +
	`xmlns:xlink="http://www.w3.org/1999/xlink" ` +
	`xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0" ` +
	`xmlns:dc="http://purl.org/dc/elements/1.1/" ` +
	`xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" ` +
	`office:version="1.2"`

// automaticStyles are the styles only this document's content uses: the list styles and the table.
func automaticStyles() string {
	return `<office:automatic-styles>` +
		listStyle("L_bullet", func(level int) string {
			glyphs := []string{"•", "◦", "▪"}

			return `<text:list-level-style-bullet text:level="` + strconv.Itoa(level) + `" text:bullet-char="` + glyphs[(level-1)%len(glyphs)] + `">` + listLevelProperties(level) + `</text:list-level-style-bullet>`
		}) +
		listStyle("L_number", func(level int) string {
			return `<text:list-level-style-number text:level="` + strconv.Itoa(level) + `" style:num-suffix="." style:num-format="1">` + listLevelProperties(level) + `</text:list-level-style-number>`
		}) +
		`<style:style style:name="Grid" style:family="table"><style:table-properties style:width="17cm" table:align="margins"/></style:style>` +
		`<style:style style:name="Cell" style:family="table-cell"><style:table-cell-properties fo:padding="0.1cm" fo:border="0.5pt solid #000000"/></style:style>` +
		`</office:automatic-styles>`
}

func listStyle(name string, level func(int) string) string {
	var out strings.Builder
	out.WriteString(`<text:list-style style:name="` + name + `">`)
	for each := 1; each <= 6; each++ {
		out.WriteString(level(each))
	}
	out.WriteString("</text:list-style>")

	return out.String()
}

func listLevelProperties(level int) string {
	indent := strconv.FormatFloat(0.63*float64(level), 'f', 2, 64)

	return `<style:list-level-properties text:list-level-position-and-space-mode="label-alignment"><style:list-level-label-alignment text:label-followed-by="listtab" text:list-tab-stop-position="` +
		indent + `cm" fo:text-indent="-0.63cm" fo:margin-left="` + indent + `cm"/></style:list-level-properties>`
}

// namedStyles is styles.xml's body: the paragraph and character styles the content refers to, by
// the names LibreOffice gives its own.
func namedStyles() string {
	var out strings.Builder
	out.WriteString(`<office:styles>`)
	out.WriteString(`<style:default-style style:family="paragraph"><style:paragraph-properties fo:margin-top="0cm" fo:margin-bottom="0.2cm"/><style:text-properties style:font-name="Calibri" fo:font-size="11pt"/></style:default-style>`)
	out.WriteString(`<style:style style:name="Standard" style:family="paragraph" style:class="text"/>`)
	out.WriteString(`<style:style style:name="Text_20_body" style:display-name="Text body" style:family="paragraph" style:parent-style-name="Standard" style:class="text"/>`)
	sizes := []string{"20pt", "16pt", "14pt", "12pt", "11pt", "11pt"}
	for level, size := range sizes {
		number := strconv.Itoa(level + 1)
		out.WriteString(`<style:style style:name="Heading_20_` + number + `" style:display-name="Heading ` + number + `" style:family="paragraph" style:parent-style-name="Standard" style:next-style-name="Text_20_body" style:default-outline-level="` + number + `" style:class="text">` +
			`<style:paragraph-properties fo:margin-top="0.4cm" fo:margin-bottom="0.15cm" fo:keep-with-next="always"/><style:text-properties fo:font-size="` + size + `" fo:font-weight="bold"/></style:style>`)
	}
	out.WriteString(`<style:style style:name="Preformatted_20_Text" style:display-name="Preformatted Text" style:family="paragraph" style:parent-style-name="Standard" style:class="html">` +
		`<style:paragraph-properties fo:background-color="#f2f2f2" fo:padding="0.15cm"/><style:text-properties style:font-name="Consolas" fo:font-size="9.5pt" fo:font-family="Consolas, 'Courier New', monospace"/></style:style>`)
	out.WriteString(`<style:style style:name="Quotations" style:family="paragraph" style:parent-style-name="Standard" style:class="html">` +
		`<style:paragraph-properties fo:margin-left="0.6cm" fo:border-left="1.5pt solid #bbbbbb" fo:padding-left="0.25cm"/><style:text-properties fo:color="#595959"/></style:style>`)
	out.WriteString(`<style:style style:name="Horizontal_20_Line" style:display-name="Horizontal Line" style:family="paragraph" style:parent-style-name="Standard" style:class="html">` +
		`<style:paragraph-properties fo:border-bottom="0.5pt solid #999999" fo:margin-bottom="0.3cm"/></style:style>`)
	out.WriteString(`<style:style style:name="List_20_Contents" style:display-name="List Contents" style:family="paragraph" style:parent-style-name="Standard" style:class="list">` +
		`<style:paragraph-properties fo:margin-bottom="0.1cm"/></style:style>`)
	out.WriteString(`<style:style style:name="Table_20_Contents" style:display-name="Table Contents" style:family="paragraph" style:parent-style-name="Standard" style:class="extra"/>`)
	out.WriteString(`<style:style style:name="Table_20_Heading" style:display-name="Table Heading" style:family="paragraph" style:parent-style-name="Table_20_Contents" style:class="extra">` +
		`<style:text-properties fo:font-weight="bold"/></style:style>`)
	out.WriteString(`<style:style style:name="Cell_20_Centre" style:display-name="Cell Centre" style:family="paragraph" style:parent-style-name="Table_20_Contents"><style:paragraph-properties fo:text-align="center"/></style:style>`)
	out.WriteString(`<style:style style:name="Cell_20_Right" style:display-name="Cell Right" style:family="paragraph" style:parent-style-name="Table_20_Contents"><style:paragraph-properties fo:text-align="end"/></style:style>`)
	out.WriteString(`<style:style style:name="Emphasis" style:family="text"><style:text-properties fo:font-style="italic"/></style:style>`)
	out.WriteString(`<style:style style:name="Strong_20_Emphasis" style:display-name="Strong Emphasis" style:family="text"><style:text-properties fo:font-weight="bold"/></style:style>`)
	out.WriteString(`<style:style style:name="Strikethrough" style:family="text"><style:text-properties style:text-line-through-style="solid"/></style:style>`)
	out.WriteString(`<style:style style:name="Source_20_Text" style:display-name="Source Text" style:family="text"><style:text-properties style:font-name="Consolas" fo:font-family="Consolas, 'Courier New', monospace" fo:background-color="#f2f2f2"/></style:style>`)
	out.WriteString(`<style:style style:name="Internet_20_link" style:display-name="Internet link" style:family="text"><style:text-properties fo:color="#0563c1" style:text-underline-style="solid" style:text-underline-width="auto" style:text-underline-color="font-color"/></style:style>`)
	out.WriteString(`</office:styles>`)

	return out.String()
}
