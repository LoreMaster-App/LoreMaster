package d2rendering

import (
	"errors"
	"strings"

	"oss.terrastruct.com/d2/d2compiler"
	"oss.terrastruct.com/d2/d2parser"

	"lore-master/libs/documentation-sync/platformport"
)

// CheckSyntax compiles source without laying it out and reports what D2 refuses, each with the
// line and column D2 gives. It is fast enough to run on every edit. A diagram D2 accepts, and an
// empty one, have no problems.
func CheckSyntax(source string) []platformport.DiagramProblem {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	_, _, err := d2compiler.Compile("", strings.NewReader(source), nil)
	if err == nil {
		return nil
	}
	var refused *d2parser.ParseError
	if !errors.As(err, &refused) {
		return []platformport.DiagramProblem{{Message: err.Error()}}
	}
	problems := make([]platformport.DiagramProblem, 0, len(refused.Errors))
	for _, each := range refused.Errors {
		problems = append(problems, platformport.DiagramProblem{Line: each.Range.Start.Line + 1, Column: each.Range.Start.Column + 1, Message: each.Message})
	}

	return problems
}
