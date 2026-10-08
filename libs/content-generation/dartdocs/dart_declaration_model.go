package dartdocs

// unit is one parsed Dart file as dartdoc_json writes it.
type unit struct {
	Source       string        `json:"source"`
	Declarations []declaration `json:"declarations"`
}

// declaration is a top-level declaration or a member of one; which fields are set depends on
// its kind (class, mixin, enum, extension, extension-type, typedef, function, variable, and for
// members constructor, method, getter, setter, field).
type declaration struct {
	Kind           string          `json:"kind"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	TypeParameters []typeParameter `json:"typeParameters"`
	Abstract       bool            `json:"abstract"`
	Sealed         bool            `json:"sealed"`
	Base           bool            `json:"base"`
	Final          bool            `json:"final"`
	Interface      bool            `json:"interface"`
	MixinClass     bool            `json:"mixinClass"`
	Extends        string          `json:"extends"`
	With           []string        `json:"with"`
	Implements     []string        `json:"implements"`
	// On is a string for an extension ("on String") and a list for a mixin.
	On             any            `json:"on"`
	Values         []enumValue    `json:"values"`
	Members        []declaration  `json:"members"`
	Parameters     *parameterList `json:"parameters"`
	Returns        string         `json:"returns"`
	Type           string         `json:"type"`
	Const          bool           `json:"const"`
	Late           bool           `json:"late"`
	Static         bool           `json:"static"`
	Factory        bool           `json:"factory"`
	Annotations    []annotation   `json:"annotations"`
	Representation *parameter     `json:"representation"`
}

type typeParameter struct {
	Name    string `json:"name"`
	Extends string `json:"extends"`
}

type enumValue struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// parameterList holds every parameter in order, and how many of the last ones are named or
// optional positional.
type parameterList struct {
	All        []parameter `json:"all"`
	Named      int         `json:"named"`
	Positional int         `json:"positional"`
}

type parameter struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
}

type annotation struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments"`
}
