package docxexport

import (
	"regexp"
	"strconv"
	"strings"
)

// styleFragment is one style definition: the id the writer refers to it by, the name Word shows, and its XML.
type styleFragment struct {
	id, name, xml string
}

var (
	styleElement  = regexp.MustCompile(`(?s)<w:style\b[^>]*>.*?</w:style>`)
	styleID       = regexp.MustCompile(`w:styleId="([^"]*)"`)
	styleName     = regexp.MustCompile(`<w:name w:val="([^"]*)"`)
	numberID      = regexp.MustCompile(`<w:num w:numId="(\d+)"`)
	abstractID    = regexp.MustCompile(`<w:abstractNum w:abstractNumId="(\d+)"`)
	numberUse     = regexp.MustCompile(`<w:numId w:val="(\d+)"/>`)
	sectionProps  = regexp.MustCompile(`(?s)<w:sectPr\b[^>]*/>|<w:sectPr\b.*?</w:sectPr>`)
	styleUse      = regexp.MustCompile(`<w:(pStyle|rStyle|tblStyle) w:val="([^"]*)"/>`)
	relationships = regexp.MustCompile(`(?s)</Relationships>\s*$`)
)

func fragmentOf(xml string) styleFragment {
	id, name := "", ""
	if match := styleID.FindStringSubmatch(xml); match != nil {
		id = match[1]
	}
	if match := styleName.FindStringSubmatch(xml); match != nil {
		name = match[1]
	}

	return styleFragment{id: id, name: strings.ToLower(name), xml: xml}
}

// ownStyles are the styles the writer uses, taken from the definitions it ships with.
func ownStyles() []styleFragment {
	var fragments []styleFragment
	for _, xml := range styleElement.FindAllString(styles, -1) {
		fragments = append(fragments, fragmentOf(xml))
	}

	return fragments
}

// mergeStyles puts the styles the writer needs into a template's styles part, and says which style id to use for each of the
// writer's own. A style the template has under the same name (Word's "heading 1" whatever the template calls its id, and
// whatever language it is in) is used as it is, so the document takes the template's look; one the template lacks is added,
// based on the template's own default styles, so it fits in.
func mergeStyles(reference string) (string, map[string]string) {
	byName := map[string]string{}
	taken := map[string]bool{}
	for _, xml := range styleElement.FindAllString(reference, -1) {
		existing := fragmentOf(xml)
		taken[existing.id] = true
		if _, found := byName[existing.name]; !found {
			byName[existing.name] = existing.id
		}
	}

	own := ownStyles()
	ids := map[string]string{}
	var missing []styleFragment
	for _, fragment := range own {
		switch {
		case byName[fragment.name] != "":
			ids[fragment.id] = byName[fragment.name]
		case taken[fragment.id]:
			ids[fragment.id] = fragment.id
		default:
			ids[fragment.id] = fragment.id
			missing = append(missing, fragment)
		}
	}

	var added strings.Builder
	for _, fragment := range missing {
		xml := strings.ReplaceAll(fragment.xml, `<w:basedOn w:val="Normal"/>`, `<w:basedOn w:val="`+ids["Normal"]+`"/>`)
		xml = strings.ReplaceAll(xml, `<w:next w:val="Normal"/>`, `<w:next w:val="`+ids["Normal"]+`"/>`)
		xml = strings.ReplaceAll(xml, `<w:basedOn w:val="TableNormal"/>`, `<w:basedOn w:val="`+ids["TableNormal"]+`"/>`)
		added.WriteString(xml)
	}
	if end := strings.LastIndex(reference, "</w:styles>"); end >= 0 {
		return reference[:end] + added.String() + reference[end:], ids
	}

	return reference + added.String(), ids
}

// renameStyleReferences points the style references of a document body at the ids a template uses for them.
func renameStyleReferences(body string, ids map[string]string) string {
	return styleUse.ReplaceAllStringFunc(body, func(reference string) string {
		match := styleUse.FindStringSubmatch(reference)
		if renamed, found := ids[match[2]]; found && renamed != match[2] {
			return `<w:` + match[1] + ` w:val="` + renamed + `"/>`
		}

		return reference
	})
}

// highest is the largest number a pattern captures in some text, or zero.
func highest(pattern *regexp.Regexp, text string) int {
	largest := 0
	for _, match := range pattern.FindAllStringSubmatch(text, -1) {
		if number, err := strconv.Atoi(match[1]); err == nil && number > largest {
			largest = number
		}
	}

	return largest
}

// mergeNumbering adds the writer's list definitions to a template's numbering part, with ids above every one the template has,
// and returns how far the numbering instance ids of the body have to be moved. A template's own numbering is kept, so a
// template whose headings are numbered still numbers them. An empty template gets a numbering part of its own.
func mergeNumbering(reference string, lists []orderedList) (string, int) {
	if strings.TrimSpace(reference) == "" {
		return numberingPart(lists), 0
	}
	abstractOffset := highest(abstractID, reference) + 1
	instanceOffset := highest(numberID, reference)
	abstracts, instances := numberingDefinitions(lists, abstractOffset, instanceOffset)

	merged := reference
	if last := strings.LastIndex(merged, "</w:abstractNum>"); last >= 0 {
		end := last + len("</w:abstractNum>")
		merged = merged[:end] + abstracts + merged[end:]
	} else if open := strings.Index(merged, "<w:num "); open >= 0 {
		merged = merged[:open] + abstracts + merged[open:]
	} else if end := strings.LastIndex(merged, "</w:numbering>"); end >= 0 {
		merged = merged[:end] + abstracts + merged[end:]
	}
	if end := strings.LastIndex(merged, "</w:numbering>"); end >= 0 {
		merged = merged[:end] + instances + merged[end:]
	}

	return merged, instanceOffset
}

// moveNumberingUse moves the numbering instance ids a body refers to up by an amount.
func moveNumberingUse(body string, by int) string {
	if by == 0 {
		return body
	}

	return numberUse.ReplaceAllStringFunc(body, func(use string) string {
		number, _ := strconv.Atoi(numberUse.FindStringSubmatch(use)[1])

		return `<w:numId w:val="` + strconv.Itoa(number+by) + `"/>`
	})
}

// bodySectionProperties is the page setup at the end of a template's body: paper, margins, and the headers and footers it
// refers to. Empty when the template has none, and the writer's own page is used.
func bodySectionProperties(referenceDocument string) string {
	all := sectionProps.FindAllString(referenceDocument, -1)
	if len(all) == 0 {
		return ""
	}

	return all[len(all)-1]
}

// mergeContentTypes makes sure a template's content types cover what the writer adds: images, and the numbering part.
func mergeContentTypes(reference string, needNumbering bool) string {
	var add strings.Builder
	for extension, kind := range map[string]string{"png": "image/png", "jpeg": "image/jpeg", "gif": "image/gif"} {
		if !strings.Contains(reference, `Extension="`+extension+`"`) {
			add.WriteString(`<Default Extension="` + extension + `" ContentType="` + kind + `"/>`)
		}
	}
	if needNumbering && !strings.Contains(reference, `PartName="/word/numbering.xml"`) {
		add.WriteString(`<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>`)
	}
	if end := strings.LastIndex(reference, "</Types>"); end >= 0 {
		return reference[:end] + add.String() + reference[end:]
	}

	return reference
}

// mergeRelationships adds the writer's relationships (hyperlinks and images) to a template's, and the one to the numbering part
// when the template had none.
func mergeRelationships(reference string, ours []relationship, needNumbering bool) string {
	var add strings.Builder
	if needNumbering && !strings.Contains(reference, `/relationships/numbering"`) {
		add.WriteString(`<Relationship Id="rIdLoreMasterNumbering" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>`)
	}
	for _, each := range ours {
		mode := ""
		if each.external {
			mode = ` TargetMode="External"`
		}
		add.WriteString(`<Relationship Id="` + each.id + `" Type="` + each.kind + `" Target="` + escape(each.target) + `"` + mode + `/>`)
	}
	if relationships.MatchString(reference) {
		return relationships.ReplaceAllString(reference, add.String()+"</Relationships>")
	}

	return reference + add.String()
}
