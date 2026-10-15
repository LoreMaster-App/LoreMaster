package siterender

// Template is how a site's pages are laid out: an HTML shell with placeholders, the
// stylesheet and files it ships with, or (Headless) no shell at all.
type Template struct {
	// Name is the built-in's name, or the folder a custom template was read from.
	Name string
	// Shell is the HTML of one page; see Placeholders for what it can hold.
	Shell string
	// LayoutCSS is appended to the shared content styles in assets/lore-master.css. A custom
	// template brings its own styles as Assets and leaves this empty.
	LayoutCSS string
	// Assets are extra files the shell refers to, at their site paths.
	Assets []SiteFile
	// Headless emits the Markdown and a site.json for a team's own front end instead of pages.
	Headless bool
}

// Placeholders are the names a shell may use, written {{name}}.
var Placeholders = []string{"content", "title", "site_title", "nav", "breadcrumbs", "root", "style", "source"}
