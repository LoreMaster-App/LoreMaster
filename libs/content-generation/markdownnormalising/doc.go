// Package markdownnormalising turns the Markdown an external documentation tool wrote into
// pages LoreMaster can sync and keep stable: one top heading per page, a page title that is
// unique (a wiki space requires it), nothing the tool stamped in that changes without the code
// changing, and the same bytes every run. Relative links keep working because the folder
// layout is left as the tool made it.
package markdownnormalising
