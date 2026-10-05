package siterender

// SiteFile is one file in the generated static site: a path relative to the site root,
// always '/'-separated so it reads the same on every OS, and its bytes. A whole site is
// an ordered slice of these, ready to be written onto a gh-pages branch.
type SiteFile struct {
	Path    string
	Content []byte
}
