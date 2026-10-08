// Package documentloading reads and parses the Markdown an output syncs: every "markdown"
// content entry of the output, discovered with its own roots and excludes plus the
// workspace's discovery scope (gitignore handling and the top-level ignore list), merged
// without repeats. It is the one place that turns an output's settings into documents, so
// the sync, the page tree and the site publisher cannot disagree about what is in scope.
package documentloading
