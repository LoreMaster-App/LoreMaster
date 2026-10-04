package storageformat

import (
	"fmt"
	"strings"
)

// codeLanguages maps a fence's info string (lowercased) to the code macro's language
// name. A language not listed is omitted, which renders as plain text rather than
// failing on an edition that does not know the name. The list is the overlap of what
// Cloud and Data Center's code macro accept; #29/#30 check it against real sites.
var codeLanguages = map[string]string{
	"sh": "bash", "shell": "bash", "bash": "bash", "zsh": "bash", "console": "bash",
	"c#": "csharp", "cs": "csharp", "csharp": "csharp",
	"c++": "cpp", "cpp": "cpp", "c": "cpp", "h": "cpp",
	"css": "css", "diff": "diff", "patch": "diff", "erlang": "erlang", "groovy": "groovy",
	"html": "html", "xml": "xml", "svg": "xml",
	"java": "java", "js": "javascript", "javascript": "javascript", "jsx": "javascript", "mjs": "javascript",
	"ts": "typescript", "typescript": "typescript", "tsx": "typescript",
	"json": "json", "go": "go", "golang": "go", "kotlin": "kotlin", "kt": "kotlin",
	"perl": "perl", "php": "php", "powershell": "powershell", "ps1": "powershell",
	"py": "python", "python": "python", "rb": "ruby", "ruby": "ruby",
	"sass": "sass", "scss": "sass", "scala": "scala", "sql": "sql",
	"vb": "vb", "vbnet": "vb", "yaml": "yaml", "yml": "yaml",
}

// codeMacro writes Confluence's code macro. The body goes in CDATA, where only "]]>"
// is special: it is split across two CDATA sections. Characters XML cannot carry are
// dropped, as they are from text. mermaid tags the macro as a diagram's source, so the
// two-way pull recognises it without relying on the collapse flag (which a Confluence-side
// edit can drop).
func (r *renderer) codeMacro(info string, code string, collapse bool, mermaid bool) {
	r.out.WriteString(`<ac:structured-macro ac:name="code" ac:schema-version="1">`)
	if language, known := codeLanguages[strings.ToLower(firstWord(info))]; known {
		r.out.WriteString(`<ac:parameter ac:name="language">` + language + `</ac:parameter>`)
	}
	if collapse {
		r.out.WriteString(`<ac:parameter ac:name="collapse">true</ac:parameter>`)
	}
	if mermaid {
		r.out.WriteString(`<ac:parameter ac:name="lore-master">mermaid</ac:parameter>`)
	}
	r.out.WriteString(`<ac:plain-text-body><![CDATA[` + cdata(code) + `]]></ac:plain-text-body></ac:structured-macro>`)
}

// cdata makes text safe inside one CDATA section.
func cdata(text string) string {
	return strings.ReplaceAll(keepXMLCharacters(text), "]]>", "]]]]><![CDATA[>")
}

// firstWord is the language part of an info string such as "go title=main.go".
func firstWord(info string) string {
	if fields := strings.Fields(info); len(fields) > 0 {
		return fields[0]
	}

	return ""
}

// mermaid writes a diagram per the configured mode.
func (r *renderer) mermaid(diagram Mermaid) error {
	switch r.options.MermaidMode {
	case MermaidImage:
		if diagram.Image != nil {
			r.out.WriteString("<p>")
			if err := r.image(Image{Source: diagram.Image, Alt: "Mermaid diagram"}); err != nil {
				return err
			}
			r.out.WriteString("</p>")
			r.codeMacro("", diagram.Source, true, true)

			return nil
		}
		r.codeMacro("", diagram.Source, false, true)
	case MermaidCode:
		r.codeMacro("", diagram.Source, false, true)
	default:
		return fmt.Errorf("mermaid mode %d is not available yet (#40)", r.options.MermaidMode)
	}

	return nil
}
