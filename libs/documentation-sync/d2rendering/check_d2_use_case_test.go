package d2rendering

import "testing"

func TestAValidDiagramHasNoProblems(t *testing.T) {
	if problems := CheckSyntax("web -> api: calls\napi -> db"); len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if problems := CheckSyntax("  \n"); len(problems) != 0 {
		t.Fatalf("an empty diagram is not a syntax error: %v", problems)
	}
}

func TestASyntaxErrorCarriesItsLineAndColumn(t *testing.T) {
	problems := CheckSyntax("a -> b\nc -> {\nd -> e")
	if len(problems) == 0 {
		t.Fatal("an unclosed map should be refused")
	}
	if problems[0].Line < 2 || problems[0].Message == "" {
		t.Fatalf("want a message and a line from the second line on, got %+v", problems[0])
	}
}

func TestASemanticErrorIsReportedToo(t *testing.T) {
	if problems := CheckSyntax("a: {shape: not-a-shape}"); len(problems) == 0 || problems[0].Line != 1 {
		t.Fatalf("an unknown shape should be refused on line 1, got %v", problems)
	}
}
