package markdownwriting

import "testing"

func TestCodeBlockFencesTheTextWithTheLanguage(t *testing.T) {
	if got, want := CodeBlock("go", "func A()\n"), "```go\nfunc A()\n```\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got, want := CodeBlock("", "plain"), "```\nplain\n```\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCodeBlockUsesAFenceLongerThanAnyBackticksInside(t *testing.T) {
	if got, want := CodeBlock("text", "a ````` b"), "``````text\na ````` b\n``````\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got, want := CodeBlock("text", "``` x"), "````text\n``` x\n````\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
