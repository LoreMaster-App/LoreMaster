package templatereference

// Kind says where a template comes from.
type Kind string

const (
	// Builtin is a template shipped with LoreMaster, by name.
	Builtin Kind = "builtin"
	// Page is a Confluence page used as the template.
	Page Kind = "page"
	// File is a path in the workspace: a folder (GitHub Pages) or a file (wiki, Confluence).
	File Kind = "file"
)

// DefaultName is the template an output has when it names none: the platform's own layout.
const DefaultName = "default"

// Reference is a parsed `template` value.
type Reference struct {
	Kind Kind
	// Name is the built-in's name (Builtin).
	Name string
	// PageID is the Confluence page's id (Page).
	PageID string
	// Path is the workspace-relative, '/'-separated path (File).
	Path string
}

// IsDefault reports whether the reference is the platform's own layout, which needs no
// template at all.
func (reference Reference) IsDefault() bool {
	return reference.Kind == Builtin && reference.Name == DefaultName
}
