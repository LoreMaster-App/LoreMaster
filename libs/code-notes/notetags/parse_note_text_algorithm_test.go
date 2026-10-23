package notetags

import "testing"

func TestParseReadsTheTagTheTextAndTheBrackets(t *testing.T) {
	matcher := NewMatcher(DefaultTags)
	cases := map[string]Note{
		"TODO: refactor this":              {Tag: "TODO", Text: "refactor this"},
		"TODO refactor this":               {Tag: "TODO", Text: "refactor this"},
		"FIXME(alice): crashes on empty":   {Tag: "FIXME", Owner: "alice", Text: "crashes on empty"},
		"TODO(#12): tracked":               {Tag: "TODO", Issue: "12", Text: "tracked"},
		"HACK(2026-12-01): remove then":    {Tag: "HACK", Due: "2026-12-01", Text: "remove then"},
		"TODO(alice, #7, 2027-01-31): all": {Tag: "TODO", Owner: "alice", Issue: "7", Due: "2027-01-31", Text: "all"},
		"NOTE:":                            {Tag: "NOTE"},
		"BUG : spaced":                     {Tag: "BUG", Text: "spaced"},
	}
	for text, want := range cases {
		got, ok := matcher.Parse(text)
		if !ok || got != want {
			t.Errorf("%q: got %+v ok=%v, want %+v", text, got, ok, want)
		}
	}
}

func TestParseOnlyTakesATagAtTheStartAsAWholeWord(t *testing.T) {
	matcher := NewMatcher(DefaultTags)
	for _, text := range []string{"this is a TODO later", "TODOS are bad", "todo: lower case", "TODOLIST", "", "FIXMEs"} {
		if note, ok := matcher.Parse(text); ok {
			t.Errorf("%q should not be a note, got %+v", text, note)
		}
	}
}

func TestACustomTagListAndTheLongerTagWins(t *testing.T) {
	matcher := NewMatcher([]string{"TO", "TODO", "WHY"})

	if note, ok := matcher.Parse("TODO: x"); !ok || note.Tag != "TODO" {
		t.Errorf("got %+v ok=%v", note, ok)
	}
	if note, ok := matcher.Parse("WHY: because"); !ok || note.Tag != "WHY" {
		t.Errorf("got %+v ok=%v", note, ok)
	}
	if _, ok := NewMatcher(nil).Parse("TODO: falls back to TODO"); !ok {
		t.Error("an empty list falls back to TODO")
	}
}
