// Package coveragereporting is the coverage generator: it reads coverage reports (Cobertura
// XML and LCOV) and writes the results of each file as Markdown, organised by the source
// folders so the page tree can be navigated folder by folder. It shows numbers only, never the
// source, and several reports for one file are merged line by line.
package coveragereporting
