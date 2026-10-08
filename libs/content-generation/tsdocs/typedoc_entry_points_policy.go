package tsdocs

// entryChoice is what to tell TypeDoc to document for one project.
type entryChoice struct {
	// Found is false for a project with nothing to document, which is skipped.
	Found bool
	// UseOwnConfig leaves TypeDoc to the project's typedoc.json: nothing is passed.
	UseOwnConfig bool
	// Paths are the entry points, relative to the project folder.
	Paths []string
	// Expand documents every module under the folder instead of one entry point's exports.
	Expand bool
}

var (
	libraryIndexFiles = []string{"src/index.ts", "src/index.tsx"}
	rootIndexFiles    = []string{"index.ts", "index.tsx"}
)

// chooseEntryPoints decides what to document in a project from what is in its folder:
//
//   - a typedoc.json means the project configured TypeDoc itself, so nothing is passed;
//   - a library index (src/index.ts) documents what the index exports, which is its public API;
//   - any other src folder documents every module in it (an app has no index to export from);
//   - a root index.ts documents what it exports;
//   - a project with none of these has nothing to document.
func chooseEntryPoints(exists func(relative string) bool) entryChoice {
	if exists("typedoc.json") {
		return entryChoice{Found: true, UseOwnConfig: true}
	}
	for _, file := range libraryIndexFiles {
		if exists(file) {
			return entryChoice{Found: true, Paths: []string{file}}
		}
	}
	if exists("src") {
		return entryChoice{Found: true, Paths: []string{"src"}, Expand: true}
	}
	for _, file := range rootIndexFiles {
		if exists(file) {
			return entryChoice{Found: true, Paths: []string{file}}
		}
	}

	return entryChoice{}
}

// projectTSConfig is the tsconfig TypeDoc should read: the library or app one Nx writes beside
// a "solution" tsconfig.json that only holds references, else tsconfig.json. Empty when there
// is none.
func projectTSConfig(exists func(relative string) bool) string {
	for _, file := range []string{"tsconfig.lib.json", "tsconfig.app.json", "tsconfig.json", "jsconfig.json"} {
		if exists(file) {
			return file
		}
	}

	return ""
}
