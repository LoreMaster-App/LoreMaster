package workspacesettings

// DiscoveryScope is the part of the settings that decides which Markdown files a scan
// leaves out for every output, whatever its content entries say.
type DiscoveryScope struct {
	// SkipGitignored leaves out files the workspace's .gitignore files ignore.
	SkipGitignored bool
	// Ignore is gitignore-syntax patterns applied to every content entry.
	Ignore []string
}

// DiscoveryScope reads the scope from the settings. Git-ignored files are skipped unless
// the file says skipGitignored: false: they are usually drafts, vendored copies or build
// output that nobody wants on the platform.
func (settings Settings) DiscoveryScope() DiscoveryScope {
	return DiscoveryScope{
		SkipGitignored: settings.SkipGitignored == nil || *settings.SkipGitignored,
		Ignore:         settings.Ignore,
	}
}

// ScanFor is what a scan of one content entry of output leaves out and takes back: the scope's
// ignore list, the output's Exclude and the entry's excludes, and the output's Include.
func (scope DiscoveryScope) ScanFor(output Output, content Content) (excludes []string, includes []string) {
	excludes = scope.ExcludesFor(content)
	excludes = append(excludes, output.Exclude...)

	return excludes, output.Include
}

// ExcludesFor is the patterns to leave out for one content entry: the scope's ignore list
// followed by the entry's own excludes.
func (scope DiscoveryScope) ExcludesFor(content Content) []string {
	excludes := make([]string, 0, len(scope.Ignore)+len(content.Excludes))
	excludes = append(excludes, scope.Ignore...)

	return append(excludes, content.Excludes...)
}
