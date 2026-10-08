package languagedetection

// Language is a programming language the documentation generators know.
type Language string

// Languages, named after the ecosystems mnci supports.
const (
	TypeScript Language = "typescript"
	Python     Language = "python"
	Go         Language = "go"
	Dart       Language = "dart"
	CSharp     Language = "csharp"
)

// Project is one project of a workspace: where it is and the file that marks it.
type Project struct {
	Language Language
	// Dir is the project's folder, workspace-relative and '/'-separated; "." is the workspace root.
	Dir string
	// MarkerFile is the file that identified it, relative to Dir (the first found when several
	// could have).
	MarkerFile string
}
