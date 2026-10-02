package pagecontent

import "strings"

// cqlString quotes value as a CQL string literal: backslashes and double quotes are
// escaped with a backslash, so a title like `Say "hi"` cannot end the literal early and
// inject CQL.
func cqlString(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)

	return `"` + escaped + `"`
}
