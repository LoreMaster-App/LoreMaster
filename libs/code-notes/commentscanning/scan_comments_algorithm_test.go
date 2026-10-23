package commentscanning

import (
	"testing"

	"lore-master/libs/code-notes/commentsyntax"
	"lore-master/libs/code-notes/notetags"
)

func scan(t *testing.T, file string, text string) []notetags.Note {
	t.Helper()
	syntax, ok := commentsyntax.SyntaxFor(file)
	if !ok {
		t.Fatalf("no syntax for %s", file)
	}

	return ScanText(file, text, syntax, notetags.NewMatcher(notetags.DefaultTags))
}

func TestFindsNotesInLineAndBlockComments(t *testing.T) {
	notes := scan(t, "a.ts", "let a = 1 // TODO: later\n/* FIXME(alice): broken */\n/**\n * HACK: ugly\n * not a note\n */\n/// NOTE: doc comment\n")

	want := []notetags.Note{
		{Path: "a.ts", Line: 1, Column: 14, Tag: "TODO", Text: "later"},
		{Path: "a.ts", Line: 2, Column: 4, Tag: "FIXME", Owner: "alice", Text: "broken"},
		{Path: "a.ts", Line: 4, Column: 4, Tag: "HACK", Text: "ugly"},
		{Path: "a.ts", Line: 7, Column: 5, Tag: "NOTE", Text: "doc comment"},
	}
	if len(notes) != len(want) {
		t.Fatalf("got %+v", notes)
	}
	for i := range want {
		if notes[i] != want[i] {
			t.Errorf("note %d: got %+v, want %+v", i, notes[i], want[i])
		}
	}
}

func TestIgnoresATagInAStringAndTakesTheCommentAfterIt(t *testing.T) {
	notes := scan(t, "a.go", "s := \"// TODO: not a note\" // TODO: a note\nraw := `\n// TODO: still a string\n` // FIXME: after the raw string\n")

	if len(notes) != 2 || notes[0].Text != "a note" || notes[1].Tag != "FIXME" {
		t.Fatalf("got %+v", notes)
	}
}

func TestAnEscapedQuoteDoesNotEndTheString(t *testing.T) {
	notes := scan(t, "a.js", "x = \"say \\\" // TODO: inside\" // TODO: outside\n")

	if len(notes) != 1 || notes[0].Text != "outside" {
		t.Fatalf("got %+v", notes)
	}
}

func TestReadsHashAndHTMLAndPythonDocstringComments(t *testing.T) {
	if notes := scan(t, "a.sh", "echo '# TODO: in a string' # TODO: real\n"); len(notes) != 1 || notes[0].Text != "real" {
		t.Errorf("shell: %+v", notes)
	}
	if notes := scan(t, "a.md", "text\n<!-- TODO: write this section -->\n<!--\nFIXME: second line\n-->\n"); len(notes) != 2 {
		t.Errorf("markdown: %+v", notes)
	}
	if notes := scan(t, "a.py", "def f():\n    \"\"\"\n    TODO(#4): document\n    \"\"\"\n    # BUG: off by one\n"); len(notes) != 2 || notes[0].Issue != "4" {
		t.Errorf("python: %+v", notes)
	}
}

func TestATagThatIsNotAtTheStartOfTheCommentIsNotANote(t *testing.T) {
	if notes := scan(t, "a.go", "// we should TODO this\n// TODOS are bad\n"); len(notes) != 0 {
		t.Fatalf("got %+v", notes)
	}
}

func TestWindowsLineEndsAndAnUnclosedBlockAreFine(t *testing.T) {
	notes := scan(t, "a.c", "int a; // TODO: crlf\r\n/* FIXME: never closed\r\nBUG: also here\r\n")

	if len(notes) != 3 || notes[0].Text != "crlf" || notes[1].Text != "never closed" {
		t.Fatalf("got %+v", notes)
	}
}

func TestSyntaxForKnowsFilesByExtensionAndByName(t *testing.T) {
	for _, file := range []string{"a/b.TS", "Dockerfile", "x/Makefile", "c.yaml", "d.sql"} {
		if _, ok := commentsyntax.SyntaxFor(file); !ok {
			t.Errorf("%s should be known", file)
		}
	}
	for _, file := range []string{"image.png", "data.bin", "README", "notes.txt"} {
		if _, ok := commentsyntax.SyntaxFor(file); ok {
			t.Errorf("%s should not be scanned", file)
		}
	}
}
