package configcatalog

// Delivery is how a platform receives the pages.
const (
	// DeliveryPages syncs page by page: a plan, an execute, a page tree with annotations in the
	// files, rollback and watch mode (Confluence).
	DeliveryPages = "pages"
	// DeliverySite publishes everything as one unit with the user's own git (GitHub Pages, wiki).
	DeliverySite = "site"
)

// Field types a shell knows how to show.
const (
	// FieldText is a single line of text.
	FieldText = "text"
	// FieldChoice is one of Choices.
	FieldChoice = "choice"
	// FieldList is a list of lines (gitignore-style patterns, folders).
	FieldList = "list"
	// FieldBool is a switch; its default is "true" or "false".
	FieldBool = "bool"
	// FieldNumber is a whole number between Min and Max.
	FieldNumber = "number"
	// FieldPath is a file or folder on disk (see PathKind).
	FieldPath = "path"
)

// Lookup kinds a shell can offer a picker for. A shell that does not know a kind shows the
// field as plain text.
const (
	// LookupSpace picks one of the platform's spaces (engine method space/list).
	LookupSpace = "space"
	// LookupPage searches and browses the pages of a space (engine methods page/search and page/children).
	LookupPage = "page"
)

// Scope is who a setting belongs to.
const (
	// ScopeProject settings travel with the project (.vscode/settings.json): the team shares them.
	ScopeProject = "project"
	// ScopeUser settings are the user's own preferences, kept by the editor for them.
	ScopeUser = "user"
)

// Description is everything that can be configured.
type Description struct {
	// Platforms are the storages a project can sync to.
	Platforms []Platform `json:"platforms"`
	// OutputFields are the settings every storage has, whatever the platform.
	OutputFields []Field `json:"outputFields"`
	// Generators are the kinds of generator.
	Generators []GeneratorType `json:"generators"`
	// GeneratorFields are the settings every generator has; a GeneratorType adds its own.
	GeneratorFields []Field `json:"generatorFields"`
	// Sections group the remaining settings for a settings page.
	Sections []Section `json:"sections"`
}

// Platform is one storage the project can sync to.
type Platform struct {
	// ID is the value of an output's platform setting.
	ID string `json:"id"`
	// Label is what a user is shown ("Confluence").
	Label string `json:"label"`
	// Description is one sentence on where the pages go.
	Description string `json:"description"`
	// Icon is a neutral hint a shell maps to its own icon set: "confluence", "site", "wiki".
	Icon string `json:"icon"`
	// Delivery is DeliveryPages or DeliverySite and decides which actions apply: a page tree, sync
	// and rollback, or check, build and publish.
	Delivery string `json:"delivery"`
	// NameTemplate makes up a name for an output that has none. {key} is the value of that
	// field; {key|fallback} uses the fallback when it is empty.
	NameTemplate string `json:"nameTemplate"`
	// Connection says how a user signs in to the platform; absent when it needs no sign-in here
	// (the site platforms use the user's own git).
	Connection *Connection `json:"connection,omitempty"`
	// Fields are the output's settings specific to this platform, in the order to show them.
	Fields []Field `json:"fields"`
}

// Connection is the sign-in to a platform.
type Connection struct {
	// Field is the key of the output field holding the site address; the shell shows a
	// connection picker for it.
	Field string `json:"field"`
	// DetectMethod is the engine method that finds the platform's edition from the address
	// (edition/detect), so only the credential kinds that edition takes are offered.
	DetectMethod string `json:"detectMethod,omitempty"`
	// Kinds are the ways to sign in.
	Kinds []CredentialKind `json:"kinds"`
}

// CredentialKind is one way to sign in.
type CredentialKind struct {
	// ID is the credential kind sent to the engine ("apitoken", "pat", "basic").
	ID    string `json:"id"`
	Label string `json:"label"`
	Help  string `json:"help,omitempty"`
	// Editions are the editions of the platform that take it; empty means all.
	Editions []string `json:"editions,omitempty"`
	// Fields are what the user types.
	Fields []CredentialField `json:"fields"`
}

// CredentialField is one thing a user types to sign in. Key is the engine's credential field.
type CredentialField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Secret fields are masked and kept in the editor's secret store, never in a file.
	Secret bool `json:"secret,omitempty"`
}

// Field is one setting.
type Field struct {
	// Key is the setting's name: in an output ("space", "titlePrefix"), in a generator
	// ("output"), or the full name of an editor setting ("export.pageSize").
	Key   string `json:"key"`
	Label string `json:"label"`
	Help  string `json:"help,omitempty"`
	// Type is one of the Field constants.
	Type        string `json:"type"`
	Placeholder string `json:"placeholder,omitempty"`
	// Required fields must be filled before the output can sync.
	Required bool `json:"required,omitempty"`
	// Default is the value that applies when the field is left empty ("true"/"false" for a bool).
	Default string `json:"default,omitempty"`
	// Choices are the values of a FieldChoice.
	Choices []Choice `json:"choices,omitempty"`
	// Suggestions are values a text field often takes, offered but not required (built-in templates).
	Suggestions []string `json:"suggestions,omitempty"`
	// Min and Max bound a FieldNumber.
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
	// PathKind is "file" or "folder" for a FieldPath.
	PathKind string `json:"pathKind,omitempty"`
	// Advanced fields are folded away until asked for.
	Advanced bool `json:"advanced,omitempty"`
	// Lookup, when present, lets the user pick the value from the platform.
	Lookup *Lookup `json:"lookup,omitempty"`
}

// Choice is one value of a FieldChoice.
type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Help  string `json:"help,omitempty"`
}

// Lookup is a picker backed by the platform.
type Lookup struct {
	// Kind is LookupSpace or LookupPage.
	Kind string `json:"kind"`
	// Needs are the keys of fields that must be chosen first (a page is looked up in a space).
	Needs []string `json:"needs,omitempty"`
	// Fills copies a property of the chosen item into another field when it is still empty:
	// the chosen parent page's title becomes the title prefix.
	Fills []Fill `json:"fills,omitempty"`
}

// Fill copies one property of a looked-up item into a field.
type Fill struct {
	// Field is the key of the field to fill.
	Field string `json:"field"`
	// From is the property: "title" or "id" for a page, "key", "name" or "homepageId" for a space.
	From string `json:"from"`
}

// GeneratorType is one kind of generator.
type GeneratorType struct {
	// ID is the generator's type setting.
	ID    string `json:"id"`
	Label string `json:"label"`
	// Description says what it reads and writes.
	Description string `json:"description"`
	// Fields are the settings this type adds to GeneratorFields.
	Fields []Field `json:"fields,omitempty"`
}

// Section is a group of settings on a settings page.
type Section struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Scope is ScopeProject or ScopeUser.
	Scope string `json:"scope"`
	// Settings are the section's fields; a key is the setting's name after "loreMaster.".
	Settings []Field `json:"settings"`
}
