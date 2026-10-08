// Package testreporting is the test-results generator: it finds JUnit XML reports in the
// workspace (Go, Jest, pytest, Maven and most other runners emit them) and writes an index
// page with the totals and one page per test suite, listing every failure with its message
// and stack. The pages are plain Markdown, so they sync like any other.
package testreporting
