// Package externaltool runs the command-line tools some generators build on (TypeDoc,
// pydoc-markdown, DefaultDocumentation): it finds the tool, runs it in a given folder with a
// time limit, stops it and everything it started when the run is cancelled, and says exactly
// what to install when the tool is not there. It runs only what its caller names.
package externaltool
